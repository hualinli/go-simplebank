package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	mockdb "github.com/hualinli/go-simplebank/db/mock"
	db "github.com/hualinli/go-simplebank/db/sqlc"
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
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := NewServer(store)

			body, err := sonic.Marshal(tc.body)
			require.NoError(t, err)

			request := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
			recorder := httptest.NewRecorder()

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}
