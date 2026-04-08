package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	mockdb "github.com/hualinli/go-simplebank/db/mock"
	db "github.com/hualinli/go-simplebank/db/sqlc"
	"github.com/hualinli/go-simplebank/token"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateTransferAPI(t *testing.T) {
	user1, _ := randomUser(t)
	user2, _ := randomUser(t)
	account1 := db.Account{
		ID:       2,
		Owner:    user1.Username,
		Currency: "USD",
		Balance:  1000,
	}
	account2 := db.Account{
		ID:       3,
		Owner:    user2.Username,
		Currency: "USD",
		Balance:  1000,
	}
	entry1 := db.Entry{
		ID:        1,
		AccountID: account1.ID,
		Amount:    -100,
	}
	entry2 := db.Entry{
		ID:        2,
		AccountID: account2.ID,
		Amount:    100,
	}
	transfer := db.Transfer{
		ID:            1,
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        100,
	}
	result := db.TransferTxResult{
		Transfer:    transfer,
		FromAccount: account1,
		ToAccount:   account2,
		FromEntry:   entry1,
		ToEntry:     entry2,
	}
	tests := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			body: gin.H{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          100,
				"currency":        "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
					Times(1).
					Return(account1, nil)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account2.ID)).
					Times(1).
					Return(account2, nil)

				store.EXPECT().
					TransferTx(gomock.Any(), gomock.Eq(db.TransferTxParams{
						FromAccountID: account1.ID,
						ToAccountID:   account2.ID,
						Amount:        100,
					})).
					Times(1).
					Return(result, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var rsp createTransferResponse
				err := sonic.Unmarshal(recorder.Body.Bytes(), &rsp)
				require.NoError(t, err)
				require.Equal(t, result.Transfer.ID, rsp.Transfer.ID)
				require.Equal(t, result.FromAccount.ID, rsp.FromAccount.ID)
				require.Equal(t, result.ToAccount.ID, rsp.ToAccount.ID)
				require.Equal(t, result.FromEntry.ID, rsp.FromEntry.ID)
				require.Equal(t, result.Transfer.Amount, rsp.Transfer.Amount)
				require.Equal(t, result.Transfer.FromAccountID, rsp.Transfer.FromAccountID)
				require.Equal(t, result.Transfer.ToAccountID, rsp.Transfer.ToAccountID)
			},
		},
		{
			name: "self transfer",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			body: gin.H{
				"from_account_id": account1.ID,
				"to_account_id":   account1.ID,
				"amount":          100,
				"currency":        "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// should not call any store method
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "bad request - missing fields",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			body: gin.H{
				"from_account_id": account1.ID,
				// "to_account_id":   account2.ID, // missing to_account_id
				"amount":   100,
				"currency": "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// should not call any store method
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "forbidden - from account doesn't belong to the authenticated user",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			body: gin.H{
				"from_account_id": account2.ID, // account2 belongs to user2, not user1
				"to_account_id":   account1.ID,
				"amount":          100,
				"currency":        "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account2.ID)).
					Times(1).
					Return(account2, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "currency mismatch",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			body: gin.H{
				"from_account_id": account1.ID,
				"to_account_id":   account2.ID,
				"amount":          100,
				"currency":        "EUR", // currency mismatch
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
					Times(1).
					Return(account1, nil)

				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account2.ID)).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewTestServer(t, store)
			data, err := sonic.Marshal(tc.body)
			require.NoError(t, err)

			url := "/transfers"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestGetTransferAPI(t *testing.T) {
	user1, _ := randomUser(t)
	user2, _ := randomUser(t)
	account1 := db.Account{
		ID:       2,
		Owner:    user1.Username,
		Currency: "USD",
		Balance:  1000,
	}
	account2 := db.Account{
		ID:       3,
		Owner:    user2.Username,
		Currency: "USD",
		Balance:  1000,
	}
	transfer := db.Transfer{
		ID:            1,
		FromAccountID: account1.ID,
		ToAccountID:   account2.ID,
		Amount:        100,
	}
	tests := []struct {
		name          string
		accountID     int64
		transferID    int64
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:       "OK",
			accountID:  account1.ID,
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
					Times(1).
					Return(account1, nil)

				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Eq(transfer.ID)).
					Times(1).
					Return(transfer, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var rsp getTransferResponse
				err := sonic.Unmarshal(recorder.Body.Bytes(), &rsp)
				require.NoError(t, err)
				require.Equal(t, transfer.ID, rsp.Transfer.ID)
				require.Equal(t, transfer.FromAccountID, rsp.Transfer.FromAccountID)
				require.Equal(t, transfer.ToAccountID, rsp.Transfer.ToAccountID)
				require.Equal(t, transfer.Amount, rsp.Transfer.Amount)
			},
		},
		{
			name:       "bad request - invalid id",
			accountID:  account1.ID,
			transferID: 0, // invalid id
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
					Times(0)

				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:       "account no match",
			accountID:  account1.ID,
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user2.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
					Times(1).
					Return(account1, nil)

				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Eq(transfer.ID)).
					Times(0)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:       "transfer not found",
			accountID:  account1.ID,
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
					Times(1).
					Return(account1, nil)

				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Eq(transfer.ID)).
					Times(1).
					Return(db.Transfer{}, db.ErrRecordNotFound)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:       "transfer no match",
			accountID:  account1.ID,
			transferID: transfer.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set("Authorization", authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account1.ID)).
					Times(1).
					Return(account1, nil)

				store.EXPECT().
					GetTransfer(gomock.Any(), gomock.Eq(transfer.ID)).
					Times(1).
					Return(db.Transfer{
						ID:            transfer.ID,
						FromAccountID: account2.ID, // transfer doesn't involve account1
						ToAccountID:   account2.ID,
						Amount:        transfer.Amount,
					}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewTestServer(t, store)

			url := fmt.Sprintf("/transfers/%d/%d", tc.accountID, tc.transferID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

