package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/stretchr/testify/require"
)

func seedCategory(t *testing.T, db *sqlite.Storage, categoryType string) *storage.Category {
	t.Helper()
	category, err := db.CreateCategory(storage.CreateCategoryParams{
		Name:      "Category1",
		Type:      categoryType,
		Icon:      "icon2",
		Color:     "blue",
		IsDefault: false,
	})
	require.NoError(t, err)
	return category
}

func seedCategories(t *testing.T, db *sqlite.Storage, count int) []*storage.Category {
	t.Helper()
	results := make([]*storage.Category, 0, count)
	for i := range count {
		if i%2 == 0 {
			results = append(results, seedCategory(t, db, "income"))
		} else {
			results = append(results, seedCategory(t, db, "expense"))
		}
	}
	return results
}

func seedAccount(t *testing.T, db *sqlite.Storage, openingBalance int64) *storage.Account {
	t.Helper()
	account, err := db.CreateAccount(storage.CreateAccountParams{
		Name:           "Account1",
		Currency:       "USD",
		OpeningBalance: openingBalance,
	})
	require.NoError(t, err)
	return account
}

func seedAccounts(t *testing.T, db *sqlite.Storage, count int) []*storage.Account {
	t.Helper()
	results := make([]*storage.Account, 0, count)
	for i := range count {
		results = append(results, seedAccount(t, db, int64(i+10)*100))
	}
	return results
}

func seedAccountAndCategory(
	t *testing.T,
	db *sqlite.Storage,
	categoryType string,
) (*storage.Account, *storage.Category) {
	t.Helper()

	account := seedAccount(t, db, 100000)
	category := seedCategory(t, db, categoryType)

	return account, category
}

type seedCashflowTransactionParams struct {
	amount          int64
	accountId       string
	categoryId      string
	transactionType string
}

func seedCashflowTransaction(
	t *testing.T,
	db *sqlite.Storage,
	params seedCashflowTransactionParams,
) *storage.Transaction {
	t.Helper()
	transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
		Type:        params.transactionType,
		Amount:      params.amount,
		Description: "Transaction",
		OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
		AccountId:   &params.accountId,
		CategoryId:  &params.categoryId,
	})
	require.NoError(t, err)
	return transaction
}

type seedTransferTransactionParams struct {
	amount        int64
	fromAccountId string
	toAccountId   string
}

func seedTransferTransaction(
	t *testing.T,
	db *sqlite.Storage,
	params seedTransferTransactionParams,
) *storage.Transaction {
	t.Helper()
	transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
		Type:          "transfer",
		Amount:        params.amount,
		Description:   "Transaction",
		OccurredAt:    *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
		FromAccountId: &params.fromAccountId,
		ToAccountId:   &params.toAccountId,
	})
	require.NoError(t, err)
	return transaction
}
