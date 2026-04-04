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
	"github.com/hualinli/go-simplebank/utils"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSample(t *testing.T) {
	// 为当前测试创建一个新的gomock控制器（通用步骤）
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建一个新的MockStore实例，传入控制器作为参数，这个store就像一个数据库
	// 可以自定义它的行为，也可以监测它的调用情况
	store := mockdb.NewMockStore(ctrl)

	user, _ := randomUser(t)

	// 准备好随机账户
	account := db.Account{
		ID:       utils.RandomInt(2, 100),
		Owner:    user.Username,
		Currency: utils.RandomCurrency(),
		Balance:  0,
	}

	// 定义数据库期望
	store.EXPECT().
		GetAccount(gomock.Any(), gomock.Eq(account.ID)). // 第一个是ctx，所以任意值都能接受，第二个参数必须等于account.ID
		Times(1).                                        // 期望这个函数被调用一次，不对的话测试会失败
		Return(account, nil)                             // 当这个函数被调用时，mock数据库会按设置好的方式返回account和nil错误

		// 创建一个新的服务器实例，传入mock store
	server := NewTestServer(t, store)

	// 但是我们不需要真正启动服务器，所以我们直接创建一个HTTP请求来测试getAccount处理函数
	url := fmt.Sprintf("/accounts/%d", account.ID)
	request := httptest.NewRequest(http.MethodGet, url, nil)

	// 设置认证头
	token, _, err := server.tokenMaker.CreateToken(user.Username, time.Minute)
	require.NoError(t, err)
	authorizationHeader := fmt.Sprintf("Bearer %s", token)
	request.Header.Set(authorizationHeaderKey, authorizationHeader)

	// 创建一个HTTP响应记录器，这个记录器会捕获服务器的响应，供我们后续检查
	recorder := httptest.NewRecorder()

	// 直接调用服务器的路由器来处理这个请求，路由器会根据URL找到对应的处理函数（getAccount），并执行它
	server.router.ServeHTTP(recorder, request)

	// 检查响应状态码是否是200 OK，如果不是，测试会失败
	require.Equal(t, http.StatusOK, recorder.Code)

	// 从响应体中解析出账户信息，并检查它是否与我们预期的一样
	var gotAccount getAccountResponse
	err = sonic.Unmarshal((recorder.Body.Bytes()), &gotAccount)
	require.NoError(t, err)
	require.Equal(t, account.ID, gotAccount.ID)
	require.Equal(t, account.Currency, gotAccount.Currency)
	require.Equal(t, account.Balance, gotAccount.Balance)
}

func TestGetAccountAPI(t *testing.T) {
	// 这个测试函数的结构和TestSample类似，但它使用表驱动实现全覆盖
	user, _ := randomUser(t)
	tests := []struct {
		name          string
		accountID     int64
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountID: 1,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				account := db.Account{
					ID:       1,
					Owner:    user.Username,
					Currency: "USD",
					Balance:  100,
				}
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var gotAccount getAccountResponse
				err := sonic.Unmarshal((recorder.Body.Bytes()), &gotAccount)
				require.NoError(t, err)
				require.Equal(t, int64(1), gotAccount.ID)
				require.Equal(t, "USD", gotAccount.Currency)
				require.Equal(t, int64(100), gotAccount.Balance)
			},
		},
		{
			name:      "BadRequest",
			accountID: 0, // 无效的账户ID
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// 这个测试不需要设置任何期望，因为请求无效，处理函数会在调用数据库之前就返回错误
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "InternalError",
			accountID: 1,
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
					GetAccount(gomock.Any(), gomock.Eq(int64(1))).
					Times(1).
					Return(db.Account{}, fmt.Errorf("db error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "NotFound",
			accountID: 1,
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
					GetAccount(gomock.Any(), gomock.Eq(int64(1))).
					Times(1).
					Return(db.Account{}, fmt.Errorf("notfound"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code) // 因为还没有做测试的错误包装，所以目前会返回500错误，后续可以改成404错误
			},
		},
		{
			name:      "Unauthorized",
			accountID: 1,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// 不设置认证头，模拟未授权访问
			},
			buildStubs: func(store *mockdb.MockStore) {
				// 这个测试不需要设置任何期望，因为请求无效，处理函数会在调用数据库之前就返回错误
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:      "Forbidden",
			accountID: 1,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// 设置一个不同用户的认证头，模拟访问不属于自己的账户
				username := "otheruser"
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				account := db.Account{
					ID:       1,
					Owner:    "someuser", // 账户属于其他用户
					Currency: "USD",
					Balance:  100,
				}
				store.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
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
			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			request := httptest.NewRequest(http.MethodGet, url, nil)
			tc.setupAuth(t, request, server.tokenMaker)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestCreateAccountAPI(t *testing.T) {
	user, _ := randomUser(t)
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
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			body: gin.H{
				"currency": "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				account := db.Account{
					ID:       1,
					Owner:    user.Username,
					Currency: "USD",
					Balance:  0,
				}
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(db.CreateAccountParams{
						Owner:    user.Username,
						Currency: "USD",
					})).
					Times(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var gotAccount createAccountResponse
				err := sonic.Unmarshal((recorder.Body.Bytes()), &gotAccount)
				require.NoError(t, err)
				require.Equal(t, int64(1), gotAccount.ID)
				require.Equal(t, "USD", gotAccount.Currency)
				require.Equal(t, int64(0), gotAccount.Balance)
			},
		},
		{
			name: "BadRequest",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			body: gin.H{},
			buildStubs: func(store *mockdb.MockStore) {
				// 这个测试不需要设置任何期望，因为请求无效，处理函数会在调用数据库之前就返回错误
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			body: gin.H{
				"currency": "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(db.CreateAccountParams{
						Owner:    user.Username,
						Currency: "USD",
					})).
					Times(1).
					Return(db.Account{}, fmt.Errorf("db error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "DuplicateAccount",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			body: gin.H{
				"currency": "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(db.CreateAccountParams{
						Owner:    user.Username,
						Currency: "USD",
					})).
					Times(1).
					Return(db.Account{}, fmt.Errorf("duplicate account"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code) // 因为还没有做测试的错误包装，所以目前会返回500错误，后续可以改成400错误
			},
		},
		{
			name: "Unauthorized",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				// 不设置认证头，模拟未授权访问
			},
			body: gin.H{
				"currency": "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				// 这个测试不需要设置任何期望，因为请求无效，处理函数会在调用数据库之前就返回错误
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
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
			body, err := sonic.Marshal(tc.body)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
			tc.setupAuth(t, request, server.tokenMaker)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListAccountsAPI(t *testing.T) {
	user, _ := randomUser(t)
	tests := []struct {
		name          string
		query         string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			query: "page_id=1&page_size=2",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				accounts := []db.Account{
					{ID: 1, Owner: user.Username, Currency: "USD", Balance: 100},
					{ID: 2, Owner: user.Username, Currency: "USD", Balance: 200},
				}
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(db.ListAccountsParams{
						Owner:  user.Username,
						Limit:  2,
						Offset: 0,
					})).
					Times(1).
					Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var gotAccounts listAccountsResponse
				err := sonic.Unmarshal((recorder.Body.Bytes()), &gotAccounts)
				require.NoError(t, err)
				require.Len(t, gotAccounts.Accounts, 2)
				require.Equal(t, int64(1), gotAccounts.Accounts[0].ID)
				require.Equal(t, "USD", gotAccounts.Accounts[0].Currency)
				require.Equal(t, int64(100), gotAccounts.Accounts[0].Balance)
				require.Equal(t, int64(2), gotAccounts.Accounts[1].ID)
				require.Equal(t, "USD", gotAccounts.Accounts[1].Currency)
				require.Equal(t, int64(200), gotAccounts.Accounts[1].Balance)
			},
		},
		{
			name:  "BadRequest",
			query: "page_id=0&page_size=2",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			buildStubs: func(store *mockdb.MockStore) {
				// 这个测试不需要设置任何期望，因为请求无效，处理函数会在调用数据库之前就返回错误
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:  "InternalError",
			query: "page_id=1&page_size=2",
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
					ListAccounts(gomock.Any(), gomock.Eq(db.ListAccountsParams{
						Owner:  user.Username,
						Limit:  2,
						Offset: 0,
					})).
					Times(1).
					Return(nil, fmt.Errorf("db error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
			url := fmt.Sprintf("/accounts?%s", tc.query)
			request := httptest.NewRequest(http.MethodGet, url, nil)
			recorder := httptest.NewRecorder()

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeleteAccountAPI(t *testing.T) {
	user, _ := randomUser(t)
	tests := []struct {
		name          string
		setupAuth     func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		accountID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			accountID: 1,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(db.DeleteAccountParams{
						ID:    1,
						Owner: user.Username,
					})).
					Times(1).
					Return(db.Account{}, nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "BadRequest",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			accountID: 0,
			buildStubs: func(store *mockdb.MockStore) {
				// 这个测试不需要设置任何期望，因为请求无效，处理函数会在调用数据库之前就返回错误
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			accountID: 1,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(db.DeleteAccountParams{
						ID:    1,
						Owner: user.Username,
					})).
					Times(1).
					Return(db.Account{}, fmt.Errorf("db error"))
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			request := httptest.NewRequest(http.MethodDelete, url, nil)
			recorder := httptest.NewRecorder()

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
