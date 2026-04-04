package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/hualinli/go-simplebank/db/mock"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/token"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetEntryAPI(t *testing.T) {
	user, _ := randomUser(t)
	account := db.Account{
		ID:       1,
		Owner:    user.Username,
		Balance:  1000,
		Currency: "USD",
	}
	entry := db.Entry{
		ID:        1,
		AccountID: account.ID,
		Amount:    100,
	}
	tests := []struct {
		name          string
		accountID     int64
		entryID       int64
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountID: account.ID,
			entryID:   entry.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)

				store.EXPECT().
					GetEntry(gomock.Any(), gomock.Eq(entry.ID)).
					Times(1).
					Return(entry, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:      "bad request",
			accountID: 0,
			entryID:   0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					GetEntry(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "account not found",
			accountID: account.ID,
			entryID:   entry.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(db.Account{}, db.ErrRecordNotFound)

				store.EXPECT().
					GetEntry(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "entry not found",
			accountID: account.ID,
			entryID:   entry.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)

				store.EXPECT().
					GetEntry(gomock.Any(), gomock.Eq(entry.ID)).
					Times(1).
					Return(db.Entry{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:      "account not match",
			accountID: account.ID,
			entryID:   entry.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := "other_user"
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)

				store.EXPECT().
					GetEntry(gomock.Any(), gomock.Eq(entry.ID)).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:      "unauthorized",
			accountID: account.ID,
			entryID:   entry.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// do not set authorization header
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					GetEntry(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for i := range tests {
		tc := tests[i]

		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewTestServer(t, store)

			url := fmt.Sprintf("/entries/%d/%d", tc.accountID, tc.entryID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestListEntriesAPI(t *testing.T) {
	// TODO: add test cases for ListEntriesAPI
}
