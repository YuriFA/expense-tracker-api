package sqlite_test

import (
	"testing"

	"github.com/yurifa/expense-tracker-api/internal/storage"
	"github.com/yurifa/expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	cases := map[string]struct {
		name           string
		openingBalance int64
		currency       string
		respError      bool
	}{
		"success": {
			name:           "Account1",
			currency:       "USD",
			openingBalance: 1000,
			respError:      false,
		},
		"negative opening balance": {
			name:           "Account2",
			currency:       "USD",
			openingBalance: -10000,
			respError:      false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			params := defaultAccountParams(f.User.ID)
			params.Name = tc.name
			params.Currency = tc.currency
			params.OpeningBalance = tc.openingBalance
			account, err := f.DB.CreateAccount(params)
			if tc.respError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tc.name, account.Name)
			require.Equal(t, tc.openingBalance, account.OpeningBalance)

			testutil.AssertValidUUID(t, account.ID)
			assert.Equal(t, int64(0), account.ManualAdjustment)
			assert.Equal(t, account.OpeningBalance+account.ManualAdjustment, account.Balance)

			createdAt := testutil.ParseDatetime(t, account.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, account.UpdatedAt)
			assert.Equal(t, createdAt, updatedAt)
		})
	}

	t.Run("non duplicate account ids", func(t *testing.T) {
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 10000)
		account2 := seedAccount(t, f.DB, f.User.ID, 20000)
		require.NotEqual(t, account1.ID, account2.ID)
	})

	t.Run("same name for different users is allowed", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		_ = seedAccount(t, f.DB, f.User.ID, 10000)
		_, err := f.DB.CreateAccount(defaultAccountParams(user2.ID))
		require.NoError(t, err)
	})
}

func TestUpdateAccount(t *testing.T) {
	t.Run("full params updates both params", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		params := storage.UpdateAccountParams{
			Name:             new("UpdatedAccount"),
			ManualAdjustment: new(int64(500)),
		}

		updatedAccount, err := f.DB.UpdateAccount(f.User.ID, account.ID, params)
		require.NoError(t, err)
		require.Equal(t, *params.Name, updatedAccount.Name)
		require.Equal(t, *params.ManualAdjustment, updatedAccount.ManualAdjustment)
		assert.Equal(t, account.OpeningBalance+*params.ManualAdjustment, updatedAccount.Balance)
	})

	t.Run("only name change", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		params := storage.UpdateAccountParams{
			Name: new("UpdatedAccount"),
		}

		updatedAccount, err := f.DB.UpdateAccount(f.User.ID, account.ID, params)
		require.NoError(t, err)
		require.Equal(t, int64(0), updatedAccount.ManualAdjustment)
		require.Equal(t, *params.Name, updatedAccount.Name)
		assert.Equal(
			t,
			account.OpeningBalance+updatedAccount.ManualAdjustment,
			updatedAccount.Balance,
		)
	})

	t.Run("wrong account id return not found", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.DB.UpdateAccount(f.User.ID, uuid.NewString(), storage.UpdateAccountParams{})
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})

	t.Run("update account for another user returns not found", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		_, err := f.DB.UpdateAccount(user2.ID, account.ID, storage.UpdateAccountParams{})
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestDeleteAccount(t *testing.T) {
	t.Run("existing account", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		err := f.DB.DeleteAccount(f.User.ID, account.ID)
		require.NoError(t, err)
	})

	t.Run("non existing account", func(t *testing.T) {
		f := newFixture(t)
		err := f.DB.DeleteAccount(f.User.ID, uuid.NewString())
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})

	t.Run("account with cashflow transaction", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		_ = seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          20000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: "income",
			},
		)
		err := f.DB.DeleteAccount(f.User.ID, account.ID)
		require.ErrorIs(t, err, storage.ErrAccountHasTransactions)
	})

	t.Run("account with transfer transaction", func(t *testing.T) {
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 100000)
		account2 := seedAccount(t, f.DB, f.User.ID, 100000)
		_ = seedTransferTransaction(
			t,
			f.DB,
			seedTransferTransactionParams{
				userID:        f.User.ID,
				amount:        20000,
				fromAccountID: account1.ID,
				toAccountID:   account2.ID,
			},
		)
		err := f.DB.DeleteAccount(f.User.ID, account1.ID)
		require.ErrorIs(t, err, storage.ErrAccountHasTransactions)
	})

	t.Run("double delete account", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		err := f.DB.DeleteAccount(f.User.ID, account.ID)
		require.NoError(t, err)
		err = f.DB.DeleteAccount(f.User.ID, account.ID)
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})

	t.Run("delete account for another user returns not found", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		err := f.DB.DeleteAccount(user2.ID, account.ID)
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestGetAccount(t *testing.T) {
	f := newFixture(t)
	account := seedAccount(t, f.DB, f.User.ID, 10000)

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
			id:        account.ID,
			respError: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			fetched, err := f.DB.GetAccount(f.User.ID, tc.id)

			if tc.respError {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.id, fetched.ID)
			require.Equal(t, account.OpeningBalance+account.ManualAdjustment, fetched.Balance)
		})
	}

	t.Run("account with transactions returns correct balance", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 40000)
		account2 := seedAccount(t, f.DB, f.User.ID, 30000)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		transaction := seedCashflowTransaction(t, f.DB, seedCashflowTransactionParams{
			userID:          f.User.ID,
			amount:          5000,
			accountID:       account.ID,
			categoryID:      category.ID,
			transactionType: "income",
		})
		transaction2 := seedTransferTransaction(t, f.DB, seedTransferTransactionParams{
			userID:        f.User.ID,
			amount:        15000,
			fromAccountID: account2.ID,
			toAccountID:   account.ID,
		})

		fetched, err := f.DB.GetAccount(f.User.ID, account.ID)
		require.NoError(t, err)
		require.Equal(t, account.ID, fetched.ID)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment+transaction.Amount+transaction2.Amount,
			fetched.Balance,
		)
	})

	t.Run("get account for another user returns not found", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		_, err := f.DB.GetAccount(user2.ID, account.ID)
		require.ErrorIs(t, err, storage.ErrAccountNotFound)
	})
}

func TestGetAccounts(t *testing.T) {
	t.Run("empty accounts in database", func(t *testing.T) {
		f := newFixture(t)
		accounts, err := f.DB.GetAccounts(f.User.ID)
		require.NoError(t, err)
		require.Equal(t, 0, len(accounts))
	})

	t.Run("existing accounts in database", func(t *testing.T) {
		f := newFixture(t)
		accounts := seedAccounts(t, f.DB, f.User.ID, 4)
		fetched, err := f.DB.GetAccounts(f.User.ID)
		require.NoError(t, err)
		require.Equal(t, len(accounts), len(fetched))
	})

	t.Run("accounts with transactions returns correct balances", func(t *testing.T) {
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 10000)
		incomeCategoryParams := defaultCategoryParams(f.User.ID)
		incomeCategoryParams.Type = "income"
		incomeCategoryParams.Name = "IncomeCategory"
		incomeCategory := seedCategory(t, f.DB, incomeCategoryParams)
		transaction1 := seedCashflowTransaction(t, f.DB, seedCashflowTransactionParams{
			userID:          f.User.ID,
			amount:          5000,
			accountID:       account1.ID,
			categoryID:      incomeCategory.ID,
			transactionType: "income",
		})
		account2 := seedAccount(t, f.DB, f.User.ID, 20000)
		expenseCategoryParams := defaultCategoryParams(f.User.ID)
		expenseCategoryParams.Type = "expense"
		expenseCategoryParams.Name = "ExpenseCategory"
		expenseCategory := seedCategory(t, f.DB, expenseCategoryParams)
		transaction2 := seedCashflowTransaction(t, f.DB, seedCashflowTransactionParams{
			userID:          f.User.ID,
			amount:          10000,
			accountID:       account2.ID,
			categoryID:      expenseCategory.ID,
			transactionType: "expense",
		})
		transaction3 := seedTransferTransaction(t, f.DB, seedTransferTransactionParams{
			userID:        f.User.ID,
			amount:        10000,
			fromAccountID: account2.ID,
			toAccountID:   account1.ID,
		})

		fetched, err := f.DB.GetAccounts(f.User.ID)
		require.NoError(t, err)
		require.Equal(t, 2, len(fetched))

		for _, account := range fetched {
			if account.ID == account1.ID {
				assert.Equal(
					t,
					account1.OpeningBalance+account1.ManualAdjustment+transaction1.Amount+transaction3.Amount,
					account.Balance,
				)
			}

			if account.ID == account2.ID {
				assert.Equal(
					t,
					account2.OpeningBalance+account2.ManualAdjustment-transaction2.Amount-transaction3.Amount,
					account.Balance,
				)
			}
		}
	})

	t.Run("get accounts for another user returns empty list", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		_ = seedAccount(t, f.DB, f.User.ID, 10000)
		accounts, err := f.DB.GetAccounts(user2.ID)
		require.NoError(t, err)
		assert.Empty(t, accounts)
	})
}

func TestGetAccountBalances(t *testing.T) {
	t.Run("empty db returns empty list", func(t *testing.T) {
		f := newFixture(t)
		balances, err := f.DB.GetAccountBalances(f.User.ID)
		require.NoError(t, err)
		assert.Empty(t, balances)
	})

	t.Run("accounts without transactions returns opening + adjustment", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		balances, err := f.DB.GetAccountBalances(f.User.ID)
		require.NoError(t, err)
		require.Len(t, balances, 1)

		assert.Equal(t, account.ID, balances[0].ID)
		assert.Equal(t, account.Name, balances[0].Name)
		assert.Equal(t, account.OpeningBalance+account.ManualAdjustment, balances[0].Balance)
	})

	t.Run("income only account returns opening + adjustment + income", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		categoryParams := defaultCategoryParams(f.User.ID)
		categoryParams.Type = "income"
		categoryParams.Name = "IncomeCategory"
		category := seedCategory(t, f.DB, categoryParams)
		transaction := seedCashflowTransaction(t, f.DB, seedCashflowTransactionParams{
			userID:          f.User.ID,
			amount:          5000,
			accountID:       account.ID,
			categoryID:      category.ID,
			transactionType: "income",
		})
		balances, err := f.DB.GetAccountBalances(f.User.ID)
		require.NoError(t, err)
		require.Len(t, balances, 1)

		assert.Equal(t, account.ID, balances[0].ID)
		assert.Equal(t, account.Name, balances[0].Name)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment+transaction.Amount,
			balances[0].Balance,
		)
	})

	t.Run("expense only account returns opening + adjustment + expense", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 10000)
		expenseCategoryParams := defaultCategoryParams(f.User.ID)
		expenseCategoryParams.Type = "expense"
		expenseCategoryParams.Name = "ExpenseCategory"
		expenseCategory := seedCategory(t, f.DB, expenseCategoryParams)
		transaction := seedCashflowTransaction(t, f.DB, seedCashflowTransactionParams{
			userID:          f.User.ID,
			transactionType: "expense",
			amount:          5000,
			accountID:       account.ID,
			categoryID:      expenseCategory.ID,
		})
		balances, err := f.DB.GetAccountBalances(f.User.ID)
		require.NoError(t, err)
		require.Len(t, balances, 1)

		assert.Equal(t, account.ID, balances[0].ID)
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
			f := newFixture(t)
			account := seedAccount(t, f.DB, f.User.ID, 10000)
			incomeCategoryParams := defaultCategoryParams(f.User.ID)
			incomeCategoryParams.Type = "income"
			incomeCategoryParams.Name = "IncomeCategory"
			incomeCategory := seedCategory(t, f.DB, incomeCategoryParams)
			expenseCategoryParams := defaultCategoryParams(f.User.ID)
			expenseCategoryParams.Type = "expense"
			expenseCategoryParams.Name = "ExpenseCategory"
			expenseCategory := seedCategory(t, f.DB, expenseCategoryParams)
			transaction1 := seedCashflowTransaction(
				t,
				f.DB,
				seedCashflowTransactionParams{
					userID:          f.User.ID,
					amount:          10000,
					accountID:       account.ID,
					categoryID:      incomeCategory.ID,
					transactionType: "income",
				},
			)
			transaction2 := seedCashflowTransaction(
				t,
				f.DB,
				seedCashflowTransactionParams{
					userID:          f.User.ID,
					amount:          10000,
					accountID:       account.ID,
					categoryID:      expenseCategory.ID,
					transactionType: "expense",
				},
			)
			balances, err := f.DB.GetAccountBalances(f.User.ID)
			require.NoError(t, err)
			require.Len(t, balances, 1)

			assert.Equal(t, account.ID, balances[0].ID)
			assert.Equal(t, account.Name, balances[0].Name)
			assert.Equal(
				t,
				account.OpeningBalance+account.ManualAdjustment+transaction1.Amount-transaction2.Amount,
				balances[0].Balance,
			)
		},
	)

	t.Run("multiple accounts with different transactions", func(t *testing.T) {
		f := newFixture(t)
		account1 := seedAccount(t, f.DB, f.User.ID, 200000)
		account1, err := f.DB.UpdateAccount(f.User.ID, account1.ID, storage.UpdateAccountParams{
			Name:             new("UpdatedAccount"),
			ManualAdjustment: new(int64(5540)),
		})
		require.NoError(t, err)
		account2 := seedAccount(t, f.DB, f.User.ID, 330000)
		incomeCategoryParams := defaultCategoryParams(f.User.ID)
		incomeCategoryParams.Type = "income"
		incomeCategoryParams.Name = "IncomeCategory"
		incomeCategory := seedCategory(t, f.DB, incomeCategoryParams)
		expenseCategoryParams := defaultCategoryParams(f.User.ID)
		expenseCategoryParams.Type = "expense"
		expenseCategoryParams.Name = "ExpenseCategory"
		expenseCategory := seedCategory(t, f.DB, expenseCategoryParams)

		transaction1 := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          10000,
				accountID:       account1.ID,
				categoryID:      incomeCategory.ID,
				transactionType: "income",
			},
		)
		transaction2 := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          10000,
				accountID:       account1.ID,
				categoryID:      expenseCategory.ID,
				transactionType: "expense",
			},
		)
		transaction3 := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          20000,
				accountID:       account1.ID,
				categoryID:      expenseCategory.ID,
				transactionType: "expense",
			},
		)
		transaction4 := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          30000,
				accountID:       account1.ID,
				categoryID:      incomeCategory.ID,
				transactionType: "income",
			},
		)
		transaction5 := seedTransferTransaction(
			t, f.DB, seedTransferTransactionParams{
				userID:        f.User.ID,
				amount:        50000,
				fromAccountID: account1.ID,
				toAccountID:   account2.ID,
			},
		)
		acc2transaction1 := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          250000,
				accountID:       account2.ID,
				categoryID:      incomeCategory.ID,
				transactionType: "income",
			},
		)
		acc2transaction2 := seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          50000,
				accountID:       account2.ID,
				categoryID:      expenseCategory.ID,
				transactionType: "expense",
			},
		)
		balances, err := f.DB.GetAccountBalances(f.User.ID)
		require.NoError(t, err)
		require.Len(t, balances, 2)

		for _, balance := range balances {
			if balance.ID == account1.ID {
				assert.Equal(
					t,
					account1.OpeningBalance+account1.ManualAdjustment+transaction1.Amount-transaction2.Amount-transaction3.Amount+transaction4.Amount-transaction5.Amount,
					balance.Balance,
				)
			}

			if balance.ID == account2.ID {
				assert.Equal(
					t,
					account2.OpeningBalance+account2.ManualAdjustment+acc2transaction1.Amount-acc2transaction2.Amount+transaction5.Amount,
					balance.Balance,
				)
			}
		}
	})

	t.Run("get account balances for another user returns empty list", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		_ = seedAccount(t, f.DB, f.User.ID, 10000)
		balances, err := f.DB.GetAccountBalances(user2.ID)
		require.NoError(t, err)
		assert.Empty(t, balances)
	})
}
