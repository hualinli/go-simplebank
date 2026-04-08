package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
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

type equalCreateUserParamsMatcher struct {
	arg      db.CreateUserParams
	password string
}

func (e equalCreateUserParamsMatcher) Matches(x any) bool {
	arg, ok := x.(db.CreateUserParams)
	if !ok {
		return false
	}
	err := utils.CheckPassword(e.password, arg.HashedPassword)
	if err != nil {
		return false
	}
	e.arg.HashedPassword = arg.HashedPassword
	return reflect.DeepEqual(e.arg, arg)
}

func (e equalCreateUserParamsMatcher) String() string {
	return fmt.Sprintf("matches arg %v and password %v", e.arg, e.password)
}

func EqualCreateUserParams(arg db.CreateUserParams, password string) gomock.Matcher {
	return equalCreateUserParamsMatcher{arg, password}
}

func randomUser(t *testing.T) (db.User, string) {
	username := utils.RandomUsername()
	password := utils.RandomString(6)
	fullName := utils.RandomFullName()
	email := utils.RandomEmail()

	hashedPassword, err := utils.HashPassword(password)
	require.NoError(t, err)

	return db.User{
		Username:       username,
		HashedPassword: hashedPassword,
		FullName:       fullName,
		Email:          email,
	}, password
}

func TestCreateUserAPI(t *testing.T) {
	user, password := randomUser(t)
	tests := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.CreateUserParams{
					Username: user.Username,
					FullName: user.FullName,
					Email:    user.Email,
				}
				store.EXPECT().
					CreateUser(gomock.Any(), EqualCreateUserParams(arg, password)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var gotUser userResponse
				err := sonic.Unmarshal(recorder.Body.Bytes(), &gotUser)
				require.NoError(t, err)
				require.Equal(t, user.Username, gotUser.Username)
				require.Equal(t, user.FullName, gotUser.FullName)
				require.Equal(t, user.Email, gotUser.Email)
				require.NotEmpty(t, gotUser.CreatedAt)
			},
		},
		{
			name: "bad request",
			body: gin.H{
				"username":  "1234",
				"password":  "123",
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "internal error",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, db.ErrInternalError)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "duplicate username or email",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, db.ErrUniqueViolation)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "unknown error",
			body: gin.H{
				"username":  user.Username,
				"password":  password,
				"full_name": user.FullName,
				"email":     user.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					CreateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, fmt.Errorf("some error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
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

			request := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestGetUserAPI(t *testing.T) {
	user, _ := randomUser(t)
	otherUser, _ := randomUser(t)
	tests := []struct {
		name          string
		setupAuth     func(request *http.Request, tokenMaker token.Maker)
		username      string
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var gotUser userResponse
				err := sonic.Unmarshal(recorder.Body.Bytes(), &gotUser)
				require.NoError(t, err)
				require.Equal(t, user.Username, gotUser.Username)
				require.Equal(t, user.FullName, gotUser.FullName)
				require.Equal(t, user.Email, gotUser.Email)
				require.NotEmpty(t, gotUser.CreatedAt)
			},
		},
		{
			name: "user not found",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				usernmae := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(usernmae, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "bad request",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: "invalid-username!",
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "forbidden, user no match",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := otherUser.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name: "internal error",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, db.ErrInternalError)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "unknown error",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)

				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, fmt.Errorf("some error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
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

			url := fmt.Sprintf("/users/%s", tc.username)
			request := httptest.NewRequest(http.MethodGet, url, nil)
			tc.setupAuth(request, server.tokenMaker)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestUpdateUserAPI(t *testing.T) {
	user1, _ := randomUser(t)
	newFullName := utils.RandomFullName()
	newEmail := utils.RandomEmail()
	user2, _ := randomUser(t)
	tests := []struct {
		name          string
		setupAuth     func(request *http.Request, tokenMaker token.Maker)
		username      string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user1.Username,
			body: gin.H{
				"full_name": newFullName,
				"email":     newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					Username: user1.Username,
					FullName: newFullName,
					Email:    newEmail,
				}
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(db.User{
						Username:  user1.Username,
						FullName:  newFullName,
						Email:     newEmail,
						CreatedAt: user1.CreatedAt,
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var gotUser userResponse
				err := sonic.Unmarshal(recorder.Body.Bytes(), &gotUser)
				require.NoError(t, err)
				require.Equal(t, user1.Username, gotUser.Username)
				require.Equal(t, newFullName, gotUser.FullName)
				require.Equal(t, newEmail, gotUser.Email)
				require.NotEmpty(t, gotUser.CreatedAt)
			},
		},
		{
			name: "user not found",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user1.Username,
			body: gin.H{
				"full_name": newFullName,
				"email":     newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "bad request",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user1.Username,
			body: gin.H{
				"full_name": "",
				"email":     "invalid-email",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "email conflict",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user1.Username,
			body: gin.H{
				"full_name": newFullName,
				"email":     user2.Email,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, db.ErrUniqueViolation)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, recorder.Code)
			},
		},
		{
			name: "internal error",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user1.Username,
			body: gin.H{
				"full_name": newFullName,
				"email":     newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, db.ErrInternalError)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "unknown error",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user1.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user1.Username,
			body: gin.H{
				"full_name": newFullName,
				"email":     newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{}, fmt.Errorf("some error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
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

			url := fmt.Sprintf("/users/%s", tc.username)
			request := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
			tc.setupAuth(request, server.tokenMaker)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestUpdatePasswordAPI(t *testing.T) {
	user, oldPassword := randomUser(t)
	newPassword := utils.RandomString(6)
	tests := []struct {
		name          string
		setupAuth     func(request *http.Request, tokenMaker token.Maker)
		username      string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			body: gin.H{
				"old_password": oldPassword,
				"new_password": newPassword,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					UpdateUserPassword(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.User{
						Username: user.Username,
						FullName: user.FullName,
						Email:    user.Email,
					}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var gotResp updateUserPasswordResponse
				err := sonic.Unmarshal(recorder.Body.Bytes(), &gotResp)
				require.NoError(t, err)
				require.Equal(t, user.Username, gotResp.Username)
				require.NotEmpty(t, gotResp.ChangeAt)
				require.WithinDuration(t, time.Now(), gotResp.ChangeAt, time.Second)
			},
		},
		{
			name: "bad request",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			body: gin.H{
				"old_password": oldPassword,
				"new_password": "123",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)

				store.EXPECT().
					UpdateUserPassword(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "internal error",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			body: gin.H{
				"old_password": oldPassword,
				"new_password": newPassword,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, db.ErrInternalError)

				store.EXPECT().
					UpdateUserPassword(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "unknown error",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			body: gin.H{
				"old_password": oldPassword,
				"new_password": newPassword,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, fmt.Errorf("some error"))

				store.EXPECT().
					UpdateUserPassword(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "incorrect old password",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			body: gin.H{
				"old_password": "wrong-old-password",
				"new_password": newPassword,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)

				store.EXPECT().
					UpdateUserPassword(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "same old and new password",
			setupAuth: func(request *http.Request, tokenMaker token.Maker) {
				username := user.Username
				duration := time.Minute
				token, _, err := tokenMaker.CreateToken(username, duration)
				require.NoError(t, err)
				authorizationHeader := fmt.Sprintf("Bearer %s", token)
				request.Header.Set(authorizationHeaderKey, authorizationHeader)
			},
			username: user.Username,
			body: gin.H{
				"old_password": oldPassword,
				"new_password": oldPassword,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(0) // 走不到这里，因为在handler里会先检查新旧密码是否相同，如果相同就直接返回400了

				store.EXPECT().
					UpdateUserPassword(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
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

			body, err := sonic.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/users/%s/password", tc.username)
			request := httptest.NewRequest(http.MethodPut, url, bytes.NewReader(body))
			tc.setupAuth(request, server.tokenMaker)
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestLoginUserAPI(t *testing.T) {
	user, password := randomUser(t)
	tests := []struct {
		name          string
		body          gin.H
		buildStubs    func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)
				store.EXPECT().
					CreateSession(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Session{}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				var gotResp loginUserResponse
				err := sonic.Unmarshal(recorder.Body.Bytes(), &gotResp)
				require.NoError(t, err)
				require.NotEmpty(t, gotResp.AccessToken)
				require.NotEmpty(t, gotResp.User)
				require.Equal(t, user.Username, gotResp.User.Username)
				require.Equal(t, user.FullName, gotResp.User.FullName)
				require.Equal(t, user.Email, gotResp.User.Email)
				require.NotEmpty(t, gotResp.User.CreatedAt)
			},
		},
		{
			name: "invalid username",
			body: gin.H{
				"username": "invalid-username!",
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "user not found",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "incorrect password",
			body: gin.H{
				"username": user.Username,
				"password": "wrong-password",
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(user, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name: "internal error",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, db.ErrInternalError)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "unknown error",
			body: gin.H{
				"username": user.Username,
				"password": password,
			},
			buildStubs: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetUser(gomock.Any(), gomock.Eq(user.Username)).
					Times(1).
					Return(db.User{}, fmt.Errorf("some error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
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

			request := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(body))
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}
