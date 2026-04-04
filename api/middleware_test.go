package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hualinli/go-simplebank/token"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := "user1"
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "NoAuthorization",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// Do not set authorization header
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "UnsupportedAuthorizationType",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := "user1"
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Basic %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidAuthorizationFormat",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := "user1"
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				// Missing "Bearer" prefix
				request.Header.Set(authorizationHeaderKey, token)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "InvalidToken",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// Set an invalid token
				authorizationHeader := fmt.Sprintf("Bearer %s", "invalidtoken")
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a new Server
			server := NewTestServer(t, nil)

			authPath := "/auth"
			server.router.GET(authPath, authMiddleware(server.tokenMaker), func(ctx *gin.Context) {
				ctx.JSON(http.StatusOK, gin.H{"message": "authorized"})
			})

			recorder := httptest.NewRecorder()
			request, err := http.NewRequest(http.MethodGet, authPath, nil)
			require.NoError(t, err)

			// Setup authorization for the request
			tc.setupAuth(t, request, server.tokenMaker)

			// Send the request
			server.router.ServeHTTP(recorder, request)

			// Check the response
			tc.checkResponse(t, recorder)
		})
	}
}
