package sqlite_test

import (
	"testing"
	"time"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateTransaction(t *testing.T) {
	db := sqlite.NewTestDB(t)

	account, category := seedAccountAndCategory(t, db, "income")

	cases := map[string]struct {
		params    storage.CreateTransactionParams
		respError bool
	}{
		"with existing category and account": {
			params: storage.CreateTransactionParams{
				Type:        "income",
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				AccountId:   account.Id,
				CategoryId:  category.Id,
			},
			respError: false,
		},
		"with existing category and account but category type not same": {
			params: storage.CreateTransactionParams{
				Type:        "expense",
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				AccountId:   account.Id,
				CategoryId:  category.Id,
			},
			respError: true,
		},
		"with non existing category and account": {
			params: storage.CreateTransactionParams{
				Type:        "income",
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				AccountId:   uuid.NewString(),
				CategoryId:  uuid.NewString(),
			},
			respError: true,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			transaction, err := db.CreateTransaction(tc.params)
			if tc.respError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tc.params.Type, transaction.Type)
			require.Equal(t, tc.params.Amount, transaction.Amount)
			require.Equal(t, tc.params.Description, transaction.Description)
			require.Equal(
				t,
				tc.params.OccurredAt.Format(time.RFC3339),
				transaction.OccurredAt,
			)
			require.Equal(t, tc.params.AccountId, transaction.AccountId)
			require.Equal(t, tc.params.CategoryId, transaction.CategoryId)

			testutil.AssertValidUUID(t, transaction.Id)

			createdAt := testutil.ParseDatetime(t, transaction.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, transaction.UpdatedAt)
			require.Equal(t, createdAt, updatedAt)
		})
	}
}

func TestUpdateTransaction(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("full params updates", func(t *testing.T) {
		account, category := seedAccountAndCategory(t, db, "income")
		expenseCategory := seedCategory(t, db, "expense")
		transaction := seedTransaction(
			t,
			db,
			seedTransactionParams{
				amount:          1000,
				accountId:       account.Id,
				categoryId:      category.Id,
				transactionType: "income",
			},
		)
		params := storage.UpdateTransactionParams{
			Type:        new("expense"),
			Amount:      new(2000.0),
			Description: new("Updated Salary"),
			OccurredAt:  testutil.GetTimeFromStr(t, "2024-06-02T00:00:00Z"),
			AccountId:   new(account.Id),
			CategoryId:  new(expenseCategory.Id),
		}

		updatedTransaction, err := db.UpdateTransaction(transaction.Id, params)
		require.NoError(t, err)

		require.Equal(t, *params.Type, updatedTransaction.Type)
		require.Equal(t, *params.Amount, updatedTransaction.Amount)
		require.Equal(t, *params.Description, updatedTransaction.Description)
		require.Equal(
			t,
			params.OccurredAt.Format(time.RFC3339),
			updatedTransaction.OccurredAt,
		)
		require.Equal(t, *params.AccountId, updatedTransaction.AccountId)
		require.Equal(t, *params.CategoryId, updatedTransaction.CategoryId)
	})

	t.Run("only type change", func(t *testing.T) {
		account, category := seedAccountAndCategory(t, db, "income")
		transaction := seedTransaction(
			t,
			db,
			seedTransactionParams{
				amount:          1000,
				accountId:       account.Id,
				categoryId:      category.Id,
				transactionType: "income",
			},
		)
		params := storage.UpdateTransactionParams{
			Type: new("expense"),
		}

		_, err := db.UpdateTransaction(transaction.Id, params)
		require.ErrorIs(t, err, storage.ErrCategoryTypeMismatch)
	})

	t.Run("wrong transaction id return not found", func(t *testing.T) {
		_, err := db.UpdateTransaction(uuid.NewString(), storage.UpdateTransactionParams{})
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})
}

func TestDeleteTransaction(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("existing transaction", func(t *testing.T) {
		account, category := seedAccountAndCategory(t, db, "income")
		transaction := seedTransaction(
			t,
			db,
			seedTransactionParams{
				amount:          1000,
				accountId:       account.Id,
				categoryId:      category.Id,
				transactionType: "income",
			},
		)

		err := db.DeleteTransaction(transaction.Id)
		require.NoError(t, err)
	})

	t.Run("non existing transaction", func(t *testing.T) {
		err := db.DeleteTransaction(uuid.NewString())
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})

	t.Run("double delete transaction", func(t *testing.T) {
		account, category := seedAccountAndCategory(t, db, "income")
		transaction := seedTransaction(
			t,
			db,
			seedTransactionParams{
				amount:          1000,
				accountId:       account.Id,
				categoryId:      category.Id,
				transactionType: "income",
			},
		)

		err := db.DeleteTransaction(transaction.Id)
		require.NoError(t, err)
		err = db.DeleteTransaction(transaction.Id)
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})
}

func TestGetTransaction(t *testing.T) {
	db := sqlite.NewTestDB(t)

	account, category := seedAccountAndCategory(t, db, "income")
	testTransaction := seedTransaction(
		t,
		db,
		seedTransactionParams{
			amount:          1000,
			accountId:       account.Id,
			categoryId:      category.Id,
			transactionType: "income",
		},
	)

	cases := map[string]struct {
		id          string
		respError   bool
		expectedErr error
	}{
		"random non exist uuid": {
			id:          uuid.NewString(),
			respError:   true,
			expectedErr: storage.ErrTransactionNotFound,
		},
		"non uuid string": {
			id:          "some id",
			respError:   true,
			expectedErr: storage.ErrTransactionNotFound,
		},
		"existing transaction id": {
			id:        testTransaction.Id,
			respError: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			transaction, err := db.GetTransaction(tc.id)

			if tc.respError {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.id, transaction.Id)
		})
	}
}

func createTestTransactions(t *testing.T, db *sqlite.Storage) ([]storage.Transaction, error) {
	t.Helper()
	account, incomeCategory := seedAccountAndCategory(t, db, "income")
	expenseCategory := seedCategory(t, db, "expense")
	transactionCreationParams := []storage.CreateTransactionParams{
		{
			Type:        "income",
			Amount:      1000,
			Description: "Salary1",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
			AccountId:   account.Id,
			CategoryId:  incomeCategory.Id,
		},
		{
			Type:        "expense",
			Amount:      2000,
			Description: "Shopping1",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-07-01T14:30:00Z"),
			AccountId:   account.Id,
			CategoryId:  expenseCategory.Id,
		},
		{
			Type:        "income",
			Amount:      5000,
			Description: "Salary2",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-05-01T23:59:00Z"),
			AccountId:   account.Id,
			CategoryId:  incomeCategory.Id,
		},
		{
			Type:        "expense",
			Amount:      3000,
			Description: "Shopping2",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-07-01T00:00:00Z"),
			AccountId:   account.Id,
			CategoryId:  expenseCategory.Id,
		},
		{
			Type:        "income",
			Amount:      1000,
			Description: "Salary3",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-05-02T00:00:00Z"),
			AccountId:   account.Id,
			CategoryId:  incomeCategory.Id,
		},
		{
			Type:        "expense",
			Amount:      1000,
			Description: "Game1",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-05-02T00:00:00Z"),
			AccountId:   account.Id,
			CategoryId:  expenseCategory.Id,
		},
	}

	result := []storage.Transaction{}

	for _, params := range transactionCreationParams {
		transaction, err := db.CreateTransaction(params)
		if err != nil {
			return nil, err
		}

		result = append(result, *transaction)
	}

	return result, nil
}

func TestGetTransactions(t *testing.T) {
	t.Run("empty transactions in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		transactions, err := db.GetTransactions(storage.GetTransactionsParams{})
		require.NoError(t, err)
		require.Empty(t, transactions)
	})

	t.Run("no params", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(t, db)
		require.NoError(t, err)

		transactions, err := db.GetTransactions(storage.GetTransactionsParams{})
		require.NoError(t, err)
		require.Equal(t, len(createdTransactions), len(transactions))
	})

	t.Run("type param = income", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(t, db)
		require.NoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{Type: new("income")},
		)
		require.NoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		expected = testutil.Filter(
			expected,
			func(c storage.Transaction) bool {
				return c.Type == "income"
			},
		)
		require.Equal(t, expected, transactions)
	})

	t.Run("sort param occurred_at DESC", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(t, db)
		require.NoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{Sort: new(storage.OccurredAtDesc)},
		)
		require.NoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		require.Equal(t, expected, transactions)
	})

	t.Run("from date and to date", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(t, db)
		require.NoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{
				FromDate: testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				ToDate:   testutil.GetTimeFromStr(t, "2024-07-01T00:00:00Z"),
			},
		)
		require.NoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		expected = testutil.Filter(expected, func(c storage.Transaction) bool {
			return c.OccurredAt >= "2024-06-01T00:00:00Z" && c.OccurredAt <= "2024-07-01T00:00:00Z"
		})
		require.Equal(t, expected, transactions)
	})

	t.Run("from date only", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(t, db)
		require.NoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{
				FromDate: testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
			},
		)
		require.NoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		expected = testutil.Filter(expected, func(c storage.Transaction) bool {
			return c.OccurredAt >= "2024-06-01T00:00:00Z"
		})
		require.Equal(t, expected, transactions)
	})

	t.Run("to date only", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(t, db)
		require.NoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{
				ToDate: testutil.GetTimeFromStr(t, "2024-07-01T00:00:00Z"),
			},
		)
		require.NoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		expected = testutil.Filter(expected, func(c storage.Transaction) bool {
			return c.OccurredAt <= "2024-07-01T00:00:00Z"
		})
		require.Equal(t, expected, transactions)
	})

	t.Run("limit = 2", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdTransactions, err := createTestTransactions(t, db)
		require.NoError(t, err)

		transactions, err := db.GetTransactions(
			storage.GetTransactionsParams{Limit: new(2)},
		)
		require.NoError(t, err)

		expected := testutil.Sort(
			createdTransactions,
			func(a storage.Transaction, b storage.Transaction) bool {
				return a.OccurredAt > b.OccurredAt
			},
		)
		require.Equal(t, expected[0:2], transactions)
	})
}
