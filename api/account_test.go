package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	mockdb "github.com/hualinli/go-simplebank/db/mock"
	db "github.com/hualinli/go-simplebank/db/sqlc"
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

	// 准备好随机账户
	account := db.Account{
		ID:       utils.RandomInt(2, 100),
		Owner:    utils.RandomOwner(),
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

	// 创建一个HTTP响应记录器，这个记录器会捕获服务器的响应，供我们后续检查
	recorder := httptest.NewRecorder()

	// 直接调用服务器的路由器来处理这个请求，路由器会根据URL找到对应的处理函数（getAccount），并执行它
	server.router.ServeHTTP(recorder, request)

	// 检查响应状态码是否是200 OK，如果不是，测试会失败
	require.Equal(t, http.StatusOK, recorder.Code)

	// 从响应体中解析出账户信息，并检查它是否与我们预期的一样
	var gotAccount getAccountResponse
	err := sonic.Unmarshal((recorder.Body.Bytes()), &gotAccount)
	require.NoError(t, err)
	require.Equal(t, account.ID, gotAccount.ID)
	require.Equal(t, account.Owner, gotAccount.Owner)
	require.Equal(t, account.Currency, gotAccount.Currency)
	require.Equal(t, account.Balance, gotAccount.Balance)
}

func TestGetAccountAPI(t *testing.T) {
	// 这个测试函数的结构和TestSample类似，但它使用表驱动实现全覆盖
	tests := []struct {
		name          string
		accountID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountID: 1,
			buildStubs: func(store *mockdb.MockStore) {
				account := db.Account{
					ID:       1,
					Owner:    "Alice",
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
				require.Equal(t, "Alice", gotAccount.Owner)
				require.Equal(t, "USD", gotAccount.Currency)
				require.Equal(t, int64(100), gotAccount.Balance)
			},
		},
		{
			name: "BadRequest",
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
	}
	// TODO: Not Found Case

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewTestServer(t, store)
			url := fmt.Sprintf("/accounts/%d", tc.accountID)
			request := httptest.NewRequest(http.MethodGet, url, nil)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestCreateAccountAPI(t *testing.T) {
	tests := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"owner":    "Alice",
				"currency": "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				account := db.Account{
					ID:       1,
					Owner:    "Alice",
					Currency: "USD",
					Balance:  0,
				}
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(db.CreateAccountParams{
						Owner:    "Alice",
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
				require.Equal(t, "Alice", gotAccount.Owner)
				require.Equal(t, "USD", gotAccount.Currency)
				require.Equal(t, int64(0), gotAccount.Balance)
			},
		},
		{
			name: "BadRequest",
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
			body: gin.H{
				"owner":    "Alice",
				"currency": "USD",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateAccount(gomock.Any(), gomock.Eq(db.CreateAccountParams{
						Owner:    "Alice",
						Currency: "USD",
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
			body, err := sonic.Marshal(tc.body)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, "/accounts", bytes.NewReader(body))
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestListAccountsAPI(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			query: "page_id=1&page_size=2",
			buildStubs: func(store *mockdb.MockStore) {
				accounts := []db.Account{
					{ID: 1, Owner: "Alice", Currency: "USD", Balance: 100},
					{ID: 2, Owner: "Bob", Currency: "USD", Balance: 200},
				}
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(db.ListAccountsParams{
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
				require.Equal(t, "Alice", gotAccounts.Accounts[0].Owner)
				require.Equal(t, "USD", gotAccounts.Accounts[0].Currency)
				require.Equal(t, int64(100), gotAccounts.Accounts[0].Balance)
				require.Equal(t, int64(2), gotAccounts.Accounts[1].ID)
				require.Equal(t, "Bob", gotAccounts.Accounts[1].Owner)
				require.Equal(t, "USD", gotAccounts.Accounts[1].Currency)
				require.Equal(t, int64(200), gotAccounts.Accounts[1].Balance)
			},
		},
		{
			name:  "BadRequest",
			query: "page_id=0&page_size=2",
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
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListAccounts(gomock.Any(), gomock.Eq(db.ListAccountsParams{
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

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestDeleteAccountAPI(t *testing.T) {
	tests := []struct {
		name          string
		accountID     int64
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name:      "OK",
			accountID: 1,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(int64(1))).
					Times(1).
					Return(nil)
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name:      "BadRequest",
			accountID: 0,
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
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					DeleteAccount(gomock.Any(), gomock.Eq(int64(1))).
					Times(1).
					Return(fmt.Errorf("db error"))
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

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}