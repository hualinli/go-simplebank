package db

import (
	"context"
	"testing"

	"github.com/hualinli/go-simplebank/utils"
	"github.com/stretchr/testify/require"
)

func createRandomUser(t *testing.T) User {
	ctx := context.Background()
	arg := CreateUserParams{
		Username:       utils.RandomUsername(),
		HashedPassword: "hashedpassword", // TODO: implement password hashing
		FullName:       utils.RandomFullName(),
		Email:          utils.RandomEmail(),
	}
	user, err := testQueries.CreateUser(ctx, arg)
	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.Username, user.Username)
	require.Equal(t, arg.HashedPassword, user.HashedPassword)
	require.Equal(t, arg.FullName, user.FullName)
	require.Equal(t, arg.Email, user.Email)

	require.NotZero(t, user.CreatedAt)

	return user
}

func TestCreateUser(t *testing.T) {
	createRandomUser(t)
}

func TestGetUser(t *testing.T) {
	ctx := context.Background()
	user1 := createRandomUser(t)
	user2, err := testQueries.GetUser(ctx, user1.Username)
	require.NoError(t, err)
	require.NotEmpty(t, user2)

	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, user1.HashedPassword, user2.HashedPassword)
	require.Equal(t, user1.FullName, user2.FullName)
	require.Equal(t, user1.Email, user2.Email)
	require.WithinDuration(t, user1.CreatedAt.Time, user2.CreatedAt.Time, 0)
}

func TestDeleteUser(t *testing.T) {
	ctx := context.Background()
	user1 := createRandomUser(t)
	err := testQueries.DeleteUser(ctx, user1.Username)
	require.NoError(t, err)

	user2, err := testQueries.GetUser(ctx, user1.Username)
	require.Error(t, err)
	require.Empty(t, user2)
}

func TestUpdateUser(t *testing.T) {
	ctx := context.Background()
	user1 := createRandomUser(t)

	arg := UpdateUserParams{
		Username: user1.Username,
		FullName: utils.RandomFullName(),
		Email:    utils.RandomEmail(),
	}

	user2, err := testQueries.UpdateUser(ctx, arg)
	require.NoError(t, err)
	require.NotEmpty(t, user2)
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, arg.FullName, user2.FullName)
	require.Equal(t, arg.Email, user2.Email)
	require.WithinDuration(t, user1.CreatedAt.Time, user2.CreatedAt.Time, 0)
}

func TestUpdateUserPassword(t *testing.T) {
	ctx := context.Background()
	user1 := createRandomUser(t)

	arg := UpdateUserPasswordParams{
		Username:       user1.Username,
		HashedPassword: "newhashedpassword", // TODO: implement password hashing
	}

	user2, err := testQueries.UpdateUserPassword(ctx, arg)
	require.NoError(t, err)
	require.NotEmpty(t, user2)
	require.Equal(t, user1.Username, user2.Username)
	require.Equal(t, arg.HashedPassword, user2.HashedPassword)
	require.WithinDuration(t, user1.CreatedAt.Time, user2.CreatedAt.Time, 0)
}
