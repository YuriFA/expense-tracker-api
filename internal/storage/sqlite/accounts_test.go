package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	cases := map[string]struct {
		name           string
		openingBalance float64
		respError      bool
	}{
		"success": {
			name:           "Account1",
			openingBalance: 100.0,
			respError:      false,
		},
		"negative opening balance": {
			name:           "Account2",
			openingBalance: -100.0,
			respError:      false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			account, err := db.CreateAccount(tc.name, tc.openingBalance)
			if tc.respError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tc.name, account.Name)
			require.Equal(t, tc.openingBalance, account.OpeningBalance)

			testutil.AssertValidUUID(t, account.Id)
			assert.Equal(t, 0.0, account.ManualAdjustment)
			assert.Equal(t, account.OpeningBalance+account.ManualAdjustment, account.Balance)

			createdAt := testutil.ParseDatetime(t, account.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, account.UpdatedAt)
			assert.Equal(t, createdAt, updatedAt)
		})
	}

	t.Run("non duplicate account ids", func(t *testing.T) {
		account1 := seedAccount(t, db, 100.0)
		account2 := seedAccount(t, db, 200.0)
		require.NotEqual(t, account1.Id, account2.Id)
	})
}

func TestUpdateAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("full params updates both params", func(t *testing.T) {
		account := seedAccount(t, db, 100.0)
		params := storage.UpdateAccountParams{
			Name:             new("UpdatedAccount"),
			ManualAdjustment: new(50.0),
		}

		updatedAccount, err := db.UpdateAccount(account.Id, params)
		require.NoError(t, err)
		require.Equal(t, *params.Name, updatedAccount.Name)
		require.Equal(t, *params.ManualAdjustment, updatedAccount.ManualAdjustment)
		assert.Equal(t, account.OpeningBalance+*params.ManualAdjustment, updatedAccount.Balance)
	})

	t.Run("only name change", func(t *testing.T) {
		account := seedAccount(t, db, 100.0)
		params := storage.UpdateAccountParams{
			Name: new("UpdatedAccount"),
		}

		updatedAccount, err := db.UpdateAccount(account.Id, params)
		require.NoError(t, err)
		require.Equal(t, 0.0, updatedAccount.ManualAdjustment)
		require.Equal(t, *params.Name, updatedAccount.Name)
		assert.Equal(
			t,
			account.OpeningBalance+updatedAccount.ManualAdjustment,
			updatedAccount.Balance,
		)
	})

	t.Run("wrong account id return not found", func(t *testing.T) {
		_, err := db.UpdateAccount(uuid.NewString(), storage.UpdateAccountParams{})
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestDeleteAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("existing account", func(t *testing.T) {
		account := seedAccount(t, db, 100.0)
		err := db.DeleteAccount(account.Id)
		require.NoError(t, err)
	})

	t.Run("non existing account", func(t *testing.T) {
		err := db.DeleteAccount(uuid.NewString())
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})

	t.Run("account with cashflow transaction", func(t *testing.T) {
		account := seedAccount(t, db, 1000.0)
		category := seedCategory(t, db, "income")
		_ = seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				amount:          200.0,
				accountId:       account.Id,
				categoryId:      category.Id,
				transactionType: "income",
			},
		)
		err := db.DeleteAccount(account.Id)
		require.ErrorIs(t, err, storage.ErrAccountHasTransactions)
	})

	t.Run("account with transfer transaction", func(t *testing.T) {
		account1 := seedAccount(t, db, 1000.0)
		account2 := seedAccount(t, db, 1000.0)
		_ = seedTransferTransaction(
			t,
			db,
			seedTransferTransactionParams{
				amount:        200.0,
				fromAccountId: account1.Id,
				toAccountId:   account2.Id,
			},
		)
		err := db.DeleteAccount(account1.Id)
		require.ErrorIs(t, err, storage.ErrAccountHasTransactions)
	})

	t.Run("double delete account", func(t *testing.T) {
		account := seedAccount(t, db, 100.0)
		err := db.DeleteAccount(account.Id)
		require.NoError(t, err)
		err = db.DeleteAccount(account.Id)
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestGetAccount(t *testing.T) {
	db := sqlite.NewTestDB(t)

	account := seedAccount(t, db, 100.0)

	cases := map[string]struct {
		id          string
		respError   bool
		expectedErr error
	}{
		"random non exist uuid": {
			id:          uuid.NewString(),
			respError:   true,
			expectedErr: storage.ErrAccountNotFound,
		},
		"non uuid string": {
			id:          "some id",
			respError:   true,
			expectedErr: storage.ErrAccountNotFound,
		},
		"existing account id": {
			id:        account.Id,
			respError: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fetched, err := db.GetAccount(tc.id)

			if tc.respError {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.id, fetched.Id)
			require.Equal(t, account.OpeningBalance+account.ManualAdjustment, fetched.Balance)
		})
	}

	t.Run("account with transactions returns correct balance", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		account := seedAccount(t, db, 400.0)
		account2 := seedAccount(t, db, 300.0)
		category := seedCategory(t, db, "income")
		transaction := seedCashflowTransaction(t, db, seedCashflowTransactionParams{
			amount:          50.0,
			accountId:       account.Id,
			categoryId:      category.Id,
			transactionType: "income",
		})
		transaction2 := seedTransferTransaction(t, db, seedTransferTransactionParams{
			amount:        150.0,
			fromAccountId: account2.Id,
			toAccountId:   account.Id,
		})

		fetched, err := db.GetAccount(account.Id)
		require.NoError(t, err)
		require.Equal(t, account.Id, fetched.Id)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment+transaction.Amount+transaction2.Amount,
			fetched.Balance,
		)
	})
}

func TestGetAccounts(t *testing.T) {
	t.Run("empty accounts in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		accounts, err := db.GetAccounts()
		require.NoError(t, err)
		require.Equal(t, 0, len(accounts))
	})

	t.Run("existing accounts in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		accounts := seedAccounts(t, db, 4)
		fetched, err := db.GetAccounts()
		require.NoError(t, err)
		require.Equal(t, len(accounts), len(fetched))
	})

	t.Run("accounts with transactions returns correct balances", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		account1 := seedAccount(t, db, 100.0)
		incomeCategory := seedCategory(t, db, "income")
		transaction1 := seedCashflowTransaction(t, db, seedCashflowTransactionParams{
			amount:          50.0,
			accountId:       account1.Id,
			categoryId:      incomeCategory.Id,
			transactionType: "income",
		})
		account2 := seedAccount(t, db, 200.0)
		expenseCategory := seedCategory(t, db, "expense")
		transaction2 := seedCashflowTransaction(t, db, seedCashflowTransactionParams{
			amount:          100.0,
			accountId:       account2.Id,
			categoryId:      expenseCategory.Id,
			transactionType: "expense",
		})
		transaction3 := seedTransferTransaction(t, db, seedTransferTransactionParams{
			amount:        100.0,
			fromAccountId: account2.Id,
			toAccountId:   account1.Id,
		})

		fetched, err := db.GetAccounts()
		require.NoError(t, err)
		require.Equal(t, 2, len(fetched))

		for _, account := range fetched {
			if account.Id == account1.Id {
				assert.Equal(
					t,
					account1.OpeningBalance+account1.ManualAdjustment+transaction1.Amount+transaction3.Amount,
					account.Balance,
				)
			}

			if account.Id == account2.Id {
				assert.Equal(
					t,
					account2.OpeningBalance+account2.ManualAdjustment-transaction2.Amount-transaction3.Amount,
					account.Balance,
				)
			}
		}
	})
}

func TestGetAccountBalances(t *testing.T) {
	t.Run("empty db returns empty list", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		balances, err := db.GetAccountBalances()
		require.NoError(t, err)
		assert.Empty(t, balances)
	})

	t.Run("accounts without transactions returns opening + adjustment", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		account := seedAccount(t, db, 100.0)
		balances, err := db.GetAccountBalances()
		require.NoError(t, err)
		require.Len(t, balances, 1)

		assert.Equal(t, account.Id, balances[0].Id)
		assert.Equal(t, account.Name, balances[0].Name)
		assert.Equal(t, account.OpeningBalance+account.ManualAdjustment, balances[0].Balance)
	})

	t.Run("income only account returns opening + adjustment + income", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		account, category := seedAccountAndCategory(t, db, "income")
		transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
			Type:        "income",
			Amount:      50.0,
			Description: "Income Transaction",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
			AccountId:   &account.Id,
			CategoryId:  &category.Id,
		})
		require.NoError(t, err)
		balances, err := db.GetAccountBalances()
		require.NoError(t, err)
		require.Len(t, balances, 1)

		assert.Equal(t, account.Id, balances[0].Id)
		assert.Equal(t, account.Name, balances[0].Name)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment+transaction.Amount,
			balances[0].Balance,
		)
	})

	t.Run("expense only account returns opening + adjustment + expense", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		account, category := seedAccountAndCategory(t, db, "expense")
		transaction, err := db.CreateTransaction(storage.CreateTransactionParams{
			Type:        "expense",
			Amount:      50.0,
			Description: "Expense Transaction",
			OccurredAt:  *testutil.GetTimeFromStr(t, "2024-06-01T00:00:00Z"),
			AccountId:   &account.Id,
			CategoryId:  &category.Id,
		})
		require.NoError(t, err)
		balances, err := db.GetAccountBalances()
		require.NoError(t, err)
		require.Len(t, balances, 1)

		assert.Equal(t, account.Id, balances[0].Id)
		assert.Equal(t, account.Name, balances[0].Name)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment-transaction.Amount,
			balances[0].Balance,
		)
	})

	t.Run(
		"income and expense account returns opening + adjustment + income - expense",
		func(t *testing.T) {
			db := sqlite.NewTestDB(t)
			account, incomeCategory := seedAccountAndCategory(t, db, "income")
			expenseCategory := seedCategory(t, db, "expense")
			transaction1 := seedCashflowTransaction(
				t,
				db,
				seedCashflowTransactionParams{
					amount:          100.0,
					accountId:       account.Id,
					categoryId:      incomeCategory.Id,
					transactionType: "income",
				},
			)
			transaction2 := seedCashflowTransaction(
				t,
				db,
				seedCashflowTransactionParams{
					amount:          100.0,
					accountId:       account.Id,
					categoryId:      expenseCategory.Id,
					transactionType: "expense",
				},
			)
			balances, err := db.GetAccountBalances()
			require.NoError(t, err)
			require.Len(t, balances, 1)

			assert.Equal(t, account.Id, balances[0].Id)
			assert.Equal(t, account.Name, balances[0].Name)
			assert.Equal(
				t,
				account.OpeningBalance+account.ManualAdjustment+transaction1.Amount-transaction2.Amount,
				balances[0].Balance,
			)
		},
	)

	t.Run("multiple accounts with different transactions", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		account1 := seedAccount(t, db, 2000.0)
		account1, err := db.UpdateAccount(account1.Id, storage.UpdateAccountParams{
			Name:             new("UpdatedAccount"),
			ManualAdjustment: new(554.0),
		})
		require.NoError(t, err)
		account2 := seedAccount(t, db, 3300.0)
		incomeCategory := seedCategory(t, db, "income")
		expenseCategory := seedCategory(t, db, "expense")

		transaction1 := seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				amount:          100.0,
				accountId:       account1.Id,
				categoryId:      incomeCategory.Id,
				transactionType: "income",
			},
		)
		transaction2 := seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				amount:          100.0,
				accountId:       account1.Id,
				categoryId:      expenseCategory.Id,
				transactionType: "expense",
			},
		)
		transaction3 := seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				amount:          200.0,
				accountId:       account1.Id,
				categoryId:      expenseCategory.Id,
				transactionType: "expense",
			},
		)
		transaction4 := seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				amount:          300.0,
				accountId:       account1.Id,
				categoryId:      incomeCategory.Id,
				transactionType: "income",
			},
		)
		transaction5 := seedTransferTransaction(
			t, db, seedTransferTransactionParams{
				amount:        500.0,
				fromAccountId: account1.Id,
				toAccountId:   account2.Id,
			},
		)
		acc2transaction1 := seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				amount:          2500.0,
				accountId:       account2.Id,
				categoryId:      incomeCategory.Id,
				transactionType: "income",
			},
		)
		acc2transaction2 := seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				amount:          500.0,
				accountId:       account2.Id,
				categoryId:      expenseCategory.Id,
				transactionType: "expense",
			},
		)
		balances, err := db.GetAccountBalances()
		require.NoError(t, err)
		require.Len(t, balances, 2)

		for _, balance := range balances {
			if balance.Id == account1.Id {
				assert.Equal(
					t,
					account1.OpeningBalance+account1.ManualAdjustment+transaction1.Amount-transaction2.Amount-transaction3.Amount+transaction4.Amount-transaction5.Amount,
					balance.Balance,
				)
			}

			if balance.Id == account2.Id {
				assert.Equal(
					t,
					account2.OpeningBalance+account2.ManualAdjustment+acc2transaction1.Amount-acc2transaction2.Amount+transaction5.Amount,
					balance.Balance,
				)
			}
		}
	})
}
