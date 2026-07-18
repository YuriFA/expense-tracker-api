package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/storage"
	"github.com/yurifa/expense-tracker-api/internal/storage/sqlite"
	"github.com/yurifa/expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateTransaction(t *testing.T) {
	t.Parallel()
	f := newFixture(t)
	user2 := seedUser(t, f.DB)
	account := seedAccount(t, f.DB, f.User.ID, 100000)
	categoryParams := defaultCategoryParams(f.User.ID)
	categoryParams.Type = storage.TransactionTypeIncome
	category := seedCategory(t, f.DB, categoryParams)
	account2 := seedAccount(t, f.DB, f.User.ID, 100000)

	cases := map[string]struct {
		params      storage.CreateTransactionParams
		respError   bool
		expectedErr error
	}{
		"cashflow with existing category and account": {
			params: storage.CreateTransactionParams{
				UserID:      f.User.ID,
				Type:        storage.TransactionTypeIncome,
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				AccountID:   &account.ID,
				CategoryID:  &category.ID,
			},
			respError: false,
		},
		"cashflow with existing category and account but category type not same": {
			params: storage.CreateTransactionParams{
				UserID:      f.User.ID,
				Type:        storage.TransactionTypeExpense,
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				AccountID:   &account.ID,
				CategoryID:  &category.ID,
			},
			respError:   true,
			expectedErr: storage.ErrCategoryTypeMismatch,
		},
		"cashflow with non existing category and account": {
			params: storage.CreateTransactionParams{
				UserID:      f.User.ID,
				Type:        storage.TransactionTypeIncome,
				Amount:      1000,
				Description: "Salary",
				OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				AccountID:   new(uuid.NewString()),
				CategoryID:  new(uuid.NewString()),
			},
			respError:   true,
			expectedErr: storage.ErrAccountNotFound,
		},
		"transfer with existing account": {
			params: storage.CreateTransactionParams{
				UserID:        f.User.ID,
				Type:          storage.TransactionTypeTransfer,
				Amount:        1000,
				Description:   "Transfer",
				OccurredAt:    *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				FromAccountID: &account.ID,
				ToAccountID:   &account2.ID,
			},
			respError: false,
		},
		"transfer with non existing account": {
			params: storage.CreateTransactionParams{
				UserID:        f.User.ID,
				Type:          storage.TransactionTypeTransfer,
				Amount:        1000,
				Description:   "Transfer",
				OccurredAt:    *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				FromAccountID: &account.ID,
				ToAccountID:   new(uuid.NewString()),
			},
			respError:   true,
			expectedErr: storage.ErrAccountNotFound,
		},
		"transfer with same from and to account": {
			params: storage.CreateTransactionParams{
				UserID:        f.User.ID,
				Type:          storage.TransactionTypeTransfer,
				Amount:        1000,
				Description:   "Transfer",
				OccurredAt:    *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				FromAccountID: &account.ID,
				ToAccountID:   &account.ID,
			},
			respError:   true,
			expectedErr: storage.ErrSameAccountTransfer,
		},
		"transfer for another user not found": {
			params: storage.CreateTransactionParams{
				UserID:        user2.ID,
				Type:          storage.TransactionTypeTransfer,
				Amount:        1000,
				Description:   "Transfer",
				OccurredAt:    *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
				FromAccountID: &account.ID,
				ToAccountID:   &account2.ID,
			},
			respError:   true,
			expectedErr: storage.ErrAccountNotFound,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			transaction, err := f.DB.CreateTransaction(context.Background(), tc.params)
			if tc.respError {
				require.ErrorIs(t, err, tc.expectedErr)
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
			require.Equal(t, tc.params.AccountID, transaction.AccountID)
			require.Equal(t, tc.params.CategoryID, transaction.CategoryID)

			testutil.AssertValidUUID(t, transaction.ID)

			require.Equal(t, 1, transaction.Version)

			createdAt := testutil.ParseDatetime(t, transaction.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, transaction.UpdatedAt)
			require.Equal(t, createdAt, updatedAt)
		})
	}
}

func TestUpdateTransaction(t *testing.T) {
	t.Parallel()
	t.Run("cashflow full params updates", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)
		params := storage.UpdateTransactionParams{
			Version:     1,
			Amount:      new(int64(2000)),
			Description: new("Updated Salary"),
			OccurredAt:  testutil.GetTimeFromStr(t, "2024-06-02T00:00:00Z"),
			AccountID:   new(account.ID),
			CategoryID:  new(category.ID),
		}

		updatedTransaction, err := f.DB.UpdateTransaction(
			context.Background(),
			f.User.ID,
			transaction.ID,
			params,
		)
		require.NoError(t, err)

		require.Equal(t, *params.Amount, updatedTransaction.Amount)
		require.Equal(t, *params.Description, updatedTransaction.Description)
		require.Equal(
			t,
			params.OccurredAt.Format(time.RFC3339),
			updatedTransaction.OccurredAt,
		)
		require.Equal(t, *params.AccountID, *updatedTransaction.AccountID)
		require.Equal(t, *params.CategoryID, *updatedTransaction.CategoryID)
	})

	t.Run("transfer full params updates", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 20000)
		account2 := seedAccount(t, f.DB, f.User.ID, 30000)
		transaction := seedTransferTransaction(
			t,
			f.DB,
			seedTransferTransactionParams{
				userID:        f.User.ID,
				amount:        100,
				fromAccountID: account1.ID,
				toAccountID:   account2.ID,
			},
		)
		params := storage.UpdateTransactionParams{
			Version:       1,
			Amount:        new(int64(20000)),
			Description:   new("Updated Salary"),
			OccurredAt:    testutil.GetTimeFromStr(t, "2024-06-02T00:00:00Z"),
			FromAccountID: new(account1.ID),
			ToAccountID:   new(account2.ID),
		}

		updatedTransaction, err := f.DB.UpdateTransaction(
			context.Background(),
			f.User.ID,
			transaction.ID,
			params,
		)
		require.NoError(t, err)

		require.Equal(t, *params.Amount, updatedTransaction.Amount)
		require.Equal(t, *params.Description, updatedTransaction.Description)
		require.Equal(
			t,
			params.OccurredAt.Format(time.RFC3339),
			updatedTransaction.OccurredAt,
		)
		require.Equal(t, *params.FromAccountID, *updatedTransaction.FromAccountID)
		require.Equal(t, *params.ToAccountID, *updatedTransaction.ToAccountID)
	})

	t.Run("cashflow only category change", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Name = "salary"
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		expenseCategoryParams := defaultCategoryParams(f.User.ID)
		expenseCategoryParams.Name = "shopping"
		expenseCategoryParams.Type = storage.TransactionTypeExpense
		expenseCategory := seedCategory(t, f.DB, expenseCategoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)
		params := storage.UpdateTransactionParams{
			Version:    1,
			CategoryID: new(expenseCategory.ID),
		}

		_, err := f.DB.UpdateTransaction(context.Background(), f.User.ID, transaction.ID, params)
		require.ErrorIs(t, err, storage.ErrCategoryTypeMismatch)
	})

	t.Run("transfer only fromAccountID change", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 20000)
		account2 := seedAccount(t, f.DB, f.User.ID, 30000)
		account3 := seedAccount(t, f.DB, f.User.ID, 40000)
		transaction := seedTransferTransaction(
			t,
			f.DB,
			seedTransferTransactionParams{
				userID:        f.User.ID,
				amount:        1000,
				fromAccountID: account1.ID,
				toAccountID:   account2.ID,
			},
		)
		params := storage.UpdateTransactionParams{
			Version:       1,
			FromAccountID: new(account3.ID),
		}

		updatedTransaction, err := f.DB.UpdateTransaction(
			context.Background(),
			f.User.ID,
			transaction.ID,
			params,
		)
		require.NoError(t, err)

		require.Equal(t, transaction.Amount, updatedTransaction.Amount)
		require.Equal(t, transaction.Description, updatedTransaction.Description)
		require.Equal(t, transaction.OccurredAt, updatedTransaction.OccurredAt)
		require.Equal(t, *params.FromAccountID, *updatedTransaction.FromAccountID)
		require.Equal(t, *transaction.ToAccountID, *updatedTransaction.ToAccountID)
	})

	t.Run("transfer same fromAccountID toAccountID change", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 20000)
		account2 := seedAccount(t, f.DB, f.User.ID, 30000)
		transaction := seedTransferTransaction(
			t,
			f.DB,
			seedTransferTransactionParams{
				userID:        f.User.ID,
				amount:        1000,
				fromAccountID: account1.ID,
				toAccountID:   account2.ID,
			},
		)
		params := storage.UpdateTransactionParams{
			Version:       1,
			FromAccountID: new(account2.ID),
		}

		_, err := f.DB.UpdateTransaction(context.Background(), f.User.ID, transaction.ID, params)
		require.ErrorIs(t, err, storage.ErrSameAccountTransfer)
	})

	t.Run("transfer with cashflow params", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 20000)
		account2 := seedAccount(t, f.DB, f.User.ID, 30000)
		transaction := seedTransferTransaction(
			t,
			f.DB,
			seedTransferTransactionParams{
				userID:        f.User.ID,
				amount:        1000,
				fromAccountID: account1.ID,
				toAccountID:   account2.ID,
			},
		)
		params := storage.UpdateTransactionParams{
			Version:       1,
			AccountID:     new(account1.ID),
			FromAccountID: new(account2.ID),
		}

		_, err := f.DB.UpdateTransaction(context.Background(), f.User.ID, transaction.ID, params)
		require.ErrorIs(t, err, storage.ErrInvalidRefs)
	})

	t.Run("cashflow with transfer params", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Name = "salary"
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		expenseCategoryParams := defaultCategoryParams(f.User.ID)
		expenseCategoryParams.Name = "shopping"
		expenseCategoryParams.Type = storage.TransactionTypeExpense
		expenseCategory := seedCategory(t, f.DB, expenseCategoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)
		params := storage.UpdateTransactionParams{
			Version:       1,
			CategoryID:    new(expenseCategory.ID),
			FromAccountID: new(account.ID),
		}

		_, err := f.DB.UpdateTransaction(context.Background(), f.User.ID, transaction.ID, params)
		require.ErrorIs(t, err, storage.ErrInvalidRefs)
	})

	t.Run("wrong transaction id return not found", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		_, err := f.DB.UpdateTransaction(
			context.Background(),
			f.User.ID,
			uuid.NewString(),
			storage.UpdateTransactionParams{Version: 1},
		)
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})

	t.Run("transaction for another user return not found", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)
		_, err := f.DB.UpdateTransaction(
			context.Background(),
			user2.ID,
			transaction.ID,
			storage.UpdateTransactionParams{Version: 1},
		)
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})

	t.Run("stale version returns conflict", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)

		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Name = "customsalary"
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)
		params := storage.UpdateTransactionParams{
			Version: 1,
			Amount:  new(int64(100)),
		}
		firstUpdate, err := f.DB.UpdateTransaction(
			context.Background(),
			f.User.ID,
			transaction.ID,
			params,
		)
		require.NoError(t, err)
		require.Equal(t, 2, firstUpdate.Version)
		_, err = f.DB.UpdateTransaction(
			context.Background(),
			f.User.ID,
			transaction.ID,
			storage.UpdateTransactionParams{Version: 1, Amount: new(int64(200))},
		)
		require.ErrorIs(t, err, storage.ErrTransactionVersionConflict)
	})
}

func TestDeleteTransaction(t *testing.T) {
	t.Parallel()
	t.Run("existing transaction", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)

		err := f.DB.DeleteTransaction(context.Background(), f.User.ID, transaction.ID)
		require.NoError(t, err)
	})

	t.Run("non existing transaction", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		err := f.DB.DeleteTransaction(context.Background(), f.User.ID, uuid.NewString())
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})

	t.Run("double delete transaction", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)

		err := f.DB.DeleteTransaction(context.Background(), f.User.ID, transaction.ID)
		require.NoError(t, err)
		err = f.DB.DeleteTransaction(context.Background(), f.User.ID, transaction.ID)
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})

	t.Run("transaction for another user return not found", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)
		err := f.DB.DeleteTransaction(context.Background(), user2.ID, transaction.ID)
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})
}

func TestGetTransaction(t *testing.T) {
	t.Parallel()
	t.Run("cashflow", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)
		user2 := seedUser(t, f.DB)
		account2 := seedAccount(t, f.DB, user2.ID, 100000)
		category2Params := defaultCategoryParams(user2.ID)
		category2Params.Type = storage.TransactionTypeIncome
		category2 := seedCategory(t, f.DB, category2Params)
		transaction2 := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          user2.ID,
				amount:          1000,
				accountID:       account2.ID,
				categoryID:      category2.ID,
				transactionType: storage.TransactionTypeIncome,
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
				id:        transaction.ID,
				respError: false,
			},
			"not found for another user": {
				id:          transaction2.ID,
				respError:   true,
				expectedErr: storage.ErrTransactionNotFound,
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				fetched, err := f.DB.GetTransaction(context.Background(), f.User.ID, tc.id)

				if tc.respError {
					require.ErrorIs(t, err, tc.expectedErr)
					return
				}

				require.NoError(t, err)
				require.Equal(t, tc.id, fetched.ID)
			})
		}
	})

	t.Run("transfer", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 100000)
		account2 := seedAccount(t, f.DB, f.User.ID, 100000)
		transaction := seedTransferTransaction(
			t,
			f.DB,
			seedTransferTransactionParams{
				userID:        f.User.ID,
				amount:        100,
				fromAccountID: account1.ID,
				toAccountID:   account2.ID,
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
				id:        transaction.ID,
				respError: false,
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				fetched, err := f.DB.GetTransaction(context.Background(), f.User.ID, tc.id)

				if tc.respError {
					require.ErrorIs(t, err, tc.expectedErr)
					return
				}

				require.NoError(t, err)
				require.Equal(t, tc.id, fetched.ID)
			})
		}
	})

	t.Run("transaction for another user return not found", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)
		_, err := f.DB.GetTransaction(context.Background(), user2.ID, transaction.ID)
		require.ErrorIs(t, err, storage.ErrTransactionNotFound)
	})
}

func createTestTransactions(
	t *testing.T,
	db *sqlite.Storage,
	userID string,
) ([]storage.Transaction, error) {
	t.Helper()
	account := seedAccount(t, db, userID, 100000)
	incomeCategoryParams := defaultCategoryParams(userID)
	incomeCategoryParams.Type = storage.TransactionTypeIncome
	incomeCategoryParams.Name = "salary"
	incomeCategory := seedCategory(t, db, incomeCategoryParams)
	account2 := seedAccount(t, db, userID, 100000)
	expenseCategoryParams := defaultCategoryParams(userID)
	expenseCategoryParams.Type = storage.TransactionTypeExpense
	expenseCategoryParams.Name = "shopping"
	expenseCategory := seedCategory(t, db, expenseCategoryParams)
	transactionCreationParams := []storage.CreateTransactionParams{
		{
			UserID:      userID,
			Type:        storage.TransactionTypeIncome,
			Amount:      1000,
			Description: "Salary1",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
			AccountID:   &account.ID,
			CategoryID:  &incomeCategory.ID,
		},
		{
			UserID:      userID,
			Type:        storage.TransactionTypeExpense,
			Amount:      2000,
			Description: "Shopping1",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-07-01T14:30:00Z"),
			AccountID:   &account.ID,
			CategoryID:  &expenseCategory.ID,
		},
		{
			UserID:      userID,
			Type:        storage.TransactionTypeIncome,
			Amount:      5000,
			Description: "Salary2",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-05-01T23:59:00Z"),
			AccountID:   &account.ID,
			CategoryID:  &incomeCategory.ID,
		},
		{
			UserID:      userID,
			Type:        storage.TransactionTypeExpense,
			Amount:      3000,
			Description: "Shopping2",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-07-01T00:00:00Z"),
			AccountID:   &account.ID,
			CategoryID:  &expenseCategory.ID,
		},
		{
			UserID:      userID,
			Type:        storage.TransactionTypeIncome,
			Amount:      1000,
			Description: "Salary3",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-05-01T00:00:00Z"),
			AccountID:   &account.ID,
			CategoryID:  &incomeCategory.ID,
		},
		{
			UserID:      userID,
			Type:        storage.TransactionTypeExpense,
			Amount:      1000,
			Description: "Game1",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-05-04T00:00:00Z"),
			AccountID:   &account.ID,
			CategoryID:  &expenseCategory.ID,
		},
		{
			UserID:        userID,
			Type:          storage.TransactionTypeTransfer,
			Amount:        100,
			Description:   "Transfer1",
			OccurredAt:    *testutil.GetTimeFromStr(t, "2024-05-02T00:00:00Z"),
			FromAccountID: &account.ID,
			ToAccountID:   &account2.ID,
		},
		{
			UserID:        userID,
			Type:          storage.TransactionTypeTransfer,
			Amount:        300,
			Description:   "Transfer2",
			OccurredAt:    *testutil.GetTimeFromStr(t, "2024-05-03T00:00:00Z"),
			FromAccountID: &account.ID,
			ToAccountID:   &account2.ID,
		},
		{
			UserID:        userID,
			Type:          storage.TransactionTypeTransfer,
			Amount:        200,
			Description:   "Transfer3",
			OccurredAt:    *testutil.GetTimeFromStr(t, "2024-06-04T00:00:00Z"),
			FromAccountID: &account.ID,
			ToAccountID:   &account2.ID,
		},
	}

	result := []storage.Transaction{}

	for _, params := range transactionCreationParams {
		transaction, err := db.CreateTransaction(context.Background(), params)
		if err != nil {
			return nil, err
		}

		result = append(result, *transaction)
	}

	return result, nil
}

func TestGetTransactions(t *testing.T) {
	t.Parallel()
	t.Run("empty transactions in database", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
			storage.GetTransactionsParams{},
		)
		require.NoError(t, err)
		require.Empty(t, transactions)
	})

	t.Run("no params", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		createdTransactions, err := createTestTransactions(t, f.DB, f.User.ID)
		require.NoError(t, err)

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
			storage.GetTransactionsParams{},
		)
		require.NoError(t, err)
		require.Len(t, transactions, len(createdTransactions))
	})

	t.Run("account id", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		createdTransactions, err := createTestTransactions(t, f.DB, f.User.ID)
		require.NoError(t, err)

		accID := createdTransactions[0].AccountID

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
			storage.GetTransactionsParams{AccountID: accID},
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
				return c.AccountID != nil && *c.AccountID == *accID ||
					c.FromAccountID != nil &&
						*c.FromAccountID == *accID ||
					c.ToAccountID != nil && *c.ToAccountID == *accID
			},
		)
		require.Equal(t, expected, transactions)
	})

	t.Run("type param = income", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		createdTransactions, err := createTestTransactions(t, f.DB, f.User.ID)
		require.NoError(t, err)

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
			storage.GetTransactionsParams{Type: new(storage.TransactionTypeIncome)},
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
		t.Parallel()
		f := newFixture(t)
		createdTransactions, err := createTestTransactions(t, f.DB, f.User.ID)
		require.NoError(t, err)

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
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
		t.Parallel()
		f := newFixture(t)
		createdTransactions, err := createTestTransactions(t, f.DB, f.User.ID)
		require.NoError(t, err)

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
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
		t.Parallel()
		f := newFixture(t)
		createdTransactions, err := createTestTransactions(t, f.DB, f.User.ID)
		require.NoError(t, err)

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
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
		t.Parallel()
		f := newFixture(t)
		createdTransactions, err := createTestTransactions(t, f.DB, f.User.ID)
		require.NoError(t, err)

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
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
		t.Parallel()
		f := newFixture(t)
		createdTransactions, err := createTestTransactions(t, f.DB, f.User.ID)
		require.NoError(t, err)

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			f.User.ID,
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

	t.Run("not found for another user", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = storage.TransactionTypeIncome
		category := seedCategory(t, f.DB, categoryParams)
		_ = seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          1000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: storage.TransactionTypeIncome,
			},
		)

		transactions, err := f.DB.GetTransactions(
			context.Background(),
			user2.ID,
			storage.GetTransactionsParams{},
		)
		require.NoError(t, err)
		require.Empty(t, transactions)
	})
}
