package db

import (
	"context"
	"testing"

	"github.com/hualinli/go-simplebank/utils"
	"github.com/stretchr/testify/require"
)

func createRandomAccount(t *testing.T) Account {
	ctx := context.Background()
	user := createRandomUser(t)
	arg := CreateAccountParams{
		Owner:    user.Username,
		Balance:  utils.RandomMoney(),
		Currency: utils.RandomCurrency(),
	}
	account, err := testQueries.CreateAccount(ctx, arg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	require.Equal(t, arg.Owner, account.Owner)
	require.Equal(t, arg.Balance, account.Balance)
	require.Equal(t, arg.Currency, account.Currency)

	require.NotZero(t, account.ID)
	require.NotZero(t, account.CreatedAt)

	return account
}

func TestCreateAccount(t *testing.T) {
	createRandomAccount(t)
}

func TestGetAccount(t *testing.T) {
	ctx := context.Background()
	account1 := createRandomAccount(t)
	account2, err := testQueries.GetAccount(ctx, account1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt.Time, account2.CreatedAt.Time, 0)
}

func TestGetAccountForUpdate(t *testing.T) {
	ctx := context.Background()
	account1 := createRandomAccount(t)
	account2, err := testQueries.GetAccountForUpdate(ctx, account1.ID)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, account1.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt.Time, account2.CreatedAt.Time, 0)
}

func TestDeleteAccount(t *testing.T) {
	ctx := context.Background()
	account1 := createRandomAccount(t)
	arg := DeleteAccountParams{
		ID:    account1.ID,
		Owner: account1.Owner,
	}
	result, err := testQueries.DeleteAccount(ctx, arg)
	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, account1.ID, result.ID)
	account2, err := testQueries.GetAccount(ctx, account1.ID)
	require.Error(t, err)
	require.Empty(t, account2)
}

func TestListAccounts(t *testing.T) {
	ctx := context.Background()
	user := createRandomUser(t)
	arg := CreateAccountParams{
		Owner:    user.Username,
		Balance:  utils.RandomMoney(),
		Currency: "USD", // make sure two accounts have different currency
	}
	account, err := testQueries.CreateAccount(ctx, arg)
	require.NoError(t, err)
	require.NotEmpty(t, account)
	arg = CreateAccountParams{
		Owner:    user.Username,
		Balance:  utils.RandomMoney(),
		Currency: "EUR", // make sure two accounts have different currency
	}
	account, err = testQueries.CreateAccount(ctx, arg)
	require.NoError(t, err)
	require.NotEmpty(t, account)

	listArg := ListAccountsParams{
		Owner:  user.Username,
		Limit:  5,
		Offset: 0,
	}

	accounts, err := testQueries.ListAccounts(ctx, listArg)
	require.NoError(t, err)
	require.Len(t, accounts, 2)

	for _, account := range accounts {
		require.NotEmpty(t, account)
	}
}

func TestUpdateAccount(t *testing.T) {
	ctx := context.Background()
	account1 := createRandomAccount(t)

	arg := UpdateAccountParams{
		ID:      account1.ID,
		Balance: utils.RandomMoney(),
	}

	account2, err := testQueries.UpdateAccount(ctx, arg)
	require.NoError(t, err)
	require.NotEmpty(t, account2)

	require.Equal(t, account1.ID, account2.ID)
	require.Equal(t, account1.Owner, account2.Owner)
	require.Equal(t, arg.Balance, account2.Balance)
	require.Equal(t, account1.Currency, account2.Currency)
	require.WithinDuration(t, account1.CreatedAt.Time, account2.CreatedAt.Time, 0)
}
