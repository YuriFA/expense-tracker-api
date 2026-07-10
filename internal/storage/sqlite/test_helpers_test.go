package sqlite_test

import (
	"fmt"
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/stretchr/testify/require"
)

func seedUser(t *testing.T, db *sqlite.Storage, email string) *storage.User {
	t.Helper()
	user, err := db.RegisterUser(storage.RegisterUserParams{
		Email:        email,
		PasswordHash: "strongpasswordhash",
	})
	require.NoError(t, err)
	return user
}

func seedCategory(
	t *testing.T,
	db *sqlite.Storage,
	name string,
	userID string,
	categoryType string,
) *storage.Category {
	t.Helper()
	category, err := db.CreateCategory(storage.CreateCategoryParams{
		UserID: userID,
		Name:   name,
		Type:   categoryType,
		Icon:   "icon2",
		Color:  "blue",
	})
	require.NoError(t, err)
	return category
}

func seedCategories(
	t *testing.T,
	db *sqlite.Storage,
	userID string,
	count int,
) []*storage.Category {
	t.Helper()
	results := make([]*storage.Category, 0, count)
	for i := range count {
		if i%2 == 0 {
			results = append(
				results,
				seedCategory(t, db, fmt.Sprintf("incomeCategory%d", i), userID, "income"),
			)
		} else {
			results = append(
				results,
				seedCategory(t, db, fmt.Sprintf("expenseCategory%d", i), userID, "expense"),
			)
		}
	}
	return results
}

func seedAccount(
	t *testing.T,
	db *sqlite.Storage,
	userID string,
	openingBalance int64,
) *storage.Account {
	t.Helper()
	account, err := db.CreateAccount(storage.CreateAccountParams{
		UserID:         userID,
		Name:           "Account1",
		Currency:       "USD",
		OpeningBalance: openingBalance,
	})
	require.NoError(t, err)
	return account
}

func seedAccounts(t *testing.T, db *sqlite.Storage, userID string, count int) []*storage.Account {
	t.Helper()
	results := make([]*storage.Account, 0, count)
	for i := range count {
		results = append(results, seedAccount(t, db, userID, int64(i+10)*100))
	}
	return results
}

type seedCashflowTransactionParams struct {
	userID          string
	amount          int64
	accountID       string
	categoryID      string
	transactionType string
}

func seedCashflowTransaction(
	t *testing.T,
	db *sqlite.Storage,
	params seedCashflowTransactionParams,
) *storage.Transaction {
	t.Helper()
	transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
		UserID:      params.userID,
		Type:        params.transactionType,
		Amount:      params.amount,
		Description: "Transaction",
		OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
		AccountID:   &params.accountID,
		CategoryID:  &params.categoryID,
	})
	require.NoError(t, err)
	return transaction
}

type seedTransferTransactionParams struct {
	userID        string
	amount        int64
	fromAccountID string
	toAccountID   string
}

func seedTransferTransaction(
	t *testing.T,
	db *sqlite.Storage,
	params seedTransferTransactionParams,
) *storage.Transaction {
	t.Helper()
	transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
		UserID:        params.userID,
		Type:          "transfer",
		Amount:        params.amount,
		Description:   "Transaction",
		OccurredAt:    *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
		FromAccountID: &params.fromAccountID,
		ToAccountID:   &params.toAccountID,
	})
	require.NoError(t, err)
	return transaction
}
