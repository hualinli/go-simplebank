package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	mockdb "github.com/hualinli/go-simplebank/db/mock"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/token"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRefreshTokenAPI(t *testing.T) {
	tests := []struct {
		name       string
		body       gin.H
		buildStubs func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload)
		checkResp  func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(payloadID)).
					Times(1).
					Return(db.Session{
						Username:     payload.Username,
						RefreshToken: refreshToken,
						IsBlocked:    false,
						ExpiresAt:    time.Now().Add(time.Minute),
					}, nil)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "Bad Request - Missing Refresh Token",
			body: gin.H{},
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "Unauthorized - Refresh Token Invalid",
			body: gin.H{"refresh_token": "invalid_token"},
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Unauthorized - Session Not Found",
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(payloadID)).
					Times(1).
					Return(db.Session{}, db.ErrRecordNotFound)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Internal Server Error - Database Error",
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(payloadID)).
					Times(1).
					Return(db.Session{}, db.ErrInternalError)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "Unauthorized - Session Blocked",
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(payloadID)).
					Times(1).
					Return(db.Session{
						Username:     payload.Username,
						RefreshToken: refreshToken,
						IsBlocked:    true,
						ExpiresAt:    time.Now().Add(time.Minute),
					}, nil)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Unauthorized - Session Username Mismatch",
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(payloadID)).
					Times(1).
					Return(db.Session{
						Username:     "otheruser",
						RefreshToken: refreshToken,
						IsBlocked:    false,
						ExpiresAt:    time.Now().Add(time.Minute),
					}, nil)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Unauthorized - Session Refresh Token Mismatch",
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(payloadID)).
					Times(1).
					Return(db.Session{
						Username:     payload.Username,
						RefreshToken: "other_refresh_token",
						IsBlocked:	false,
						ExpiresAt:    time.Now().Add(time.Minute),
					}, nil)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "Unauthorized - Session Expired",
			buildStubs: func(store *mockdb.MockStore, payloadID uuid.UUID, refreshToken string, payload *token.Payload) {
				store.EXPECT().
					GetSession(gomock.Any(), gomock.Eq(payloadID)).
					Times(1).
					Return(db.Session{
						Username:     payload.Username,
						RefreshToken: refreshToken,
						IsBlocked:    false,
						ExpiresAt:    time.Now().Add(-time.Minute),
					}, nil)
			},
			checkResp: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			server := NewTestServer(t, store)

			// create a valid refresh token using server's token maker
			refreshToken, payload, err := server.tokenMaker.CreateToken("testuser", time.Minute)
			require.NoError(t, err)
			payloadID, err := uuid.Parse(payload.TokenID)
			require.NoError(t, err)

			tc.buildStubs(store, payloadID, refreshToken, payload)

			var body gin.H
			if tc.body == nil {
				body = gin.H{"refresh_token": refreshToken}
			} else {
				body = tc.body
			}

			reqBody, err := sonic.Marshal(body)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, "/users/refresh", bytes.NewReader(reqBody))
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResp(recorder)
		})
	}
}
