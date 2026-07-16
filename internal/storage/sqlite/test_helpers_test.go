package sqlite_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/storage"
	"github.com/yurifa/expense-tracker-api/internal/storage/sqlite"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fixture struct {
	DB   *sqlite.Storage
	User *storage.User
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	db := sqlite.NewTestDB(t)
	user := seedUser(t, db)
	return &fixture{DB: db, User: user}
}

func seedUser(t *testing.T, db *sqlite.Storage) *storage.User {
	t.Helper()
	email := uuid.NewString()[:8] + "@test.com"
	user, err := db.RegisterUser(context.Background(), storage.RegisterUserParams{
		Email:        email,
		PasswordHash: "hash",
	})
	require.NoError(t, err)
	return user
}

func defaultCategoryParams(userID string) storage.CreateCategoryParams {
	return storage.CreateCategoryParams{
		UserID: userID,
		Name:   "DefaultIncomeCategory",
		Type:   storage.TransactionTypeIncome,
		Icon:   "🍔",
		Color:  "#FF0000",
	}
}

func seedCategory(
	t *testing.T,
	db *sqlite.Storage,
	params storage.CreateCategoryParams,
) *storage.Category {
	t.Helper()
	category, err := db.CreateCategory(context.Background(), params)
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
	params := defaultCategoryParams(userID)
	for i := range count {
		if i%2 == 0 {
			params.Type = storage.TransactionTypeIncome
			params.Name = fmt.Sprintf("incomeCategory%d", i)
			results = append(
				results,
				seedCategory(t, db, params),
			)
		} else {
			params.Type = storage.TransactionTypeExpense
			params.Name = fmt.Sprintf("expenseCategory%d", i)
			results = append(
				results,
				seedCategory(t, db, params),
			)
		}
	}
	return results
}

func defaultAccountParams(userID string) storage.CreateAccountParams {
	return storage.CreateAccountParams{
		UserID:         userID,
		Name:           "Bank",
		Currency:       "USD",
		OpeningBalance: 10000,
	}
}

func seedAccount(
	t *testing.T,
	db *sqlite.Storage,
	userID string,
	openingBalance int64,
) *storage.Account {
	t.Helper()
	params := defaultAccountParams(userID)
	params.OpeningBalance = openingBalance
	account, err := db.CreateAccount(context.Background(), params)
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

func defaultCashflowTransactionParams(
	userID, accountID, categoryID string,
) storage.CreateTransactionParams {
	return storage.CreateTransactionParams{
		UserID:     userID,
		Type:       storage.TransactionTypeExpense,
		Amount:     1000,
		AccountID:  &accountID,
		CategoryID: &categoryID,
		OccurredAt: time.Now(),
	}
}

type seedCashflowTransactionParams struct {
	userID          string
	amount          int64
	accountID       string
	categoryID      string
	transactionType storage.TransactionType
}

func seedCashflowTransaction(
	t *testing.T,
	db *sqlite.Storage,
	params seedCashflowTransactionParams,
) *storage.Transaction {
	t.Helper()
	transactionParams := defaultCashflowTransactionParams(
		params.userID,
		params.accountID,
		params.categoryID,
	)
	transactionParams.Type = params.transactionType
	transactionParams.Amount = params.amount
	transaction, err := db.CreateTransaction(context.Background(), transactionParams)
	require.NoError(t, err)
	return transaction
}

func defaultTransferTransactionParams(
	userID, fromAccountID, toAccountID string,
) storage.CreateTransactionParams {
	return storage.CreateTransactionParams{
		UserID:        userID,
		Type:          storage.TransactionTypeTransfer,
		Amount:        1000,
		FromAccountID: &fromAccountID,
		ToAccountID:   &toAccountID,
		OccurredAt:    time.Now(),
	}
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
	transactionParams := defaultTransferTransactionParams(
		params.userID,
		params.fromAccountID,
		params.toAccountID,
	)
	transactionParams.Amount = params.amount
	transaction, err := db.CreateTransaction(context.Background(), transactionParams)
	require.NoError(t, err)
	return transaction
}
