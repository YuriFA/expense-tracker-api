package handlers_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/http-server/httperr"
	"github.com/yurifa/expense-tracker-api/internal/storage"
	"github.com/yurifa/expense-tracker-api/internal/storage/sqlite"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		w := f.do(t, http.MethodPost, "/api/accounts", map[string]any{
			"name":           "Wallet",
			"currency":       "USD",
			"openingBalance": 100000,
		})

		assert.Equal(t, http.StatusCreated, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Wallet", response.Name)
		assert.Equal(t, "USD", response.Currency)
		assert.Equal(t, int64(100000), response.OpeningBalance)
		assert.Equal(t, int64(0), response.ManualAdjustment)
		assert.Equal(t, int64(100000), response.Balance)
	})

	t.Run("ValidationFail", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		cases := map[string]struct {
			body        map[string]any
			wantField   string
			wantMessage string
			errorsLen   int
		}{
			"missing name": {
				body: map[string]any{
					"currency":       "USD",
					"openingBalance": 100000,
				},
				wantField:   "name",
				wantMessage: "name is required",
				errorsLen:   1,
			},
			"missing openingBalance": {
				body: map[string]any{
					"name":     "Wallet",
					"currency": "USD",
				},
				wantField:   "openingBalance",
				wantMessage: "openingBalance is required",
				errorsLen:   1,
			},
			"missing currency": {
				body: map[string]any{
					"name":           "Wallet",
					"openingBalance": 100000,
				},
				wantField:   "currency",
				wantMessage: "currency is required",
				errorsLen:   1,
			},
			"empty body": {
				body:        map[string]any{},
				wantField:   "name",
				wantMessage: "name is required",
				errorsLen:   3,
			},
			"empty name": {
				body: map[string]any{
					"name":           "",
					"currency":       "USD",
					"openingBalance": 100000,
				},
				wantField:   "name",
				wantMessage: "name is required",
				errorsLen:   1,
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				w := f.do(t, http.MethodPost, "/api/accounts", tc.body)
				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response httperr.ValidationErrorResponse
				parseBody(t, w, &response)
				assert.Equal(t, httperr.ErrCodeValidationFailed, response.Code)
				assert.Equal(t, "validation failed", response.Message)
				require.Len(t, response.Errors, tc.errorsLen)
				assert.Equal(t, tc.wantField, response.Errors[0].Field)
				assert.Equal(t, tc.wantMessage, response.Errors[0].Message)
			})
		}
	})
}

func TestUpdateAccount(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))

		w := f.do(t, http.MethodPatch, "/api/accounts/"+existing.ID, map[string]any{
			"name":             "Updated Wallet",
			"manualAdjustment": 10000,
		})

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Updated Wallet", response.Name)
		assert.Equal(t, int64(10000), response.ManualAdjustment)
		assert.Equal(t, existing.OpeningBalance+10000, response.Balance)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		w := f.do(t, http.MethodPatch, "/api/accounts/"+existing.ID, map[string]any{
			"name": "Updated Wallet",
		})

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Updated Wallet", response.Name)
		assert.Equal(t, existing.OpeningBalance, response.OpeningBalance)
		assert.Equal(t, existing.ManualAdjustment, response.ManualAdjustment)
		assert.Equal(t, existing.Balance, response.Balance)
	})

	t.Run("AccountWithTransactions", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		incomeCategory := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		expenseCategory := seedDefaultExpenseCategory(t, f.DB, f.User.ID)
		incomeTransactionParams := defaultCashflowTransactionParams(
			f.User.ID,
			existing.ID,
			incomeCategory.ID,
		)
		incomeTransactionParams.Amount = 50000
		incomeTransactionParams.Type = storage.TransactionTypeIncome
		incomeTransaction := seedTransaction(t, f.DB, incomeTransactionParams)
		expenseTransactionParams := defaultCashflowTransactionParams(
			f.User.ID,
			existing.ID,
			expenseCategory.ID,
		)
		expenseTransactionParams.Amount = 20000
		expenseTransactionParams.Type = storage.TransactionTypeExpense
		expenseTransaction := seedTransaction(t, f.DB, expenseTransactionParams)

		w := f.do(t, http.MethodPatch, "/api/accounts/"+existing.ID, map[string]any{
			"name": "Updated Wallet",
		})

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Updated Wallet", response.Name)
		assert.Equal(t, existing.OpeningBalance, response.OpeningBalance)
		assert.Equal(t, existing.ManualAdjustment, response.ManualAdjustment)
		assert.Equal(
			t,
			existing.Balance+incomeTransaction.Amount-expenseTransaction.Amount,
			response.Balance,
		)
	})

	t.Run("ShortName", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		w := f.do(t, http.MethodPatch, "/api/accounts/"+existing.ID, map[string]any{
			"name": "qw",
		})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response httperr.ValidationErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeValidationFailed, response.Code)
		assert.Equal(t, "validation failed", response.Message)
		assert.Len(t, response.Errors, 1)
		assert.Equal(t, "name", response.Errors[0].Field)
		assert.Equal(
			t,
			"name must be at least 3 characters",
			response.Errors[0].Message,
		)
	})

	t.Run("NoFields", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		w := f.do(t, http.MethodPatch, "/api/accounts/"+existing.ID, map[string]any{})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeValidationFailed, response.Code)
		assert.Equal(t, "no fields to update", response.Message)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		w := f.do(t, http.MethodPatch, "/api/accounts/"+uuid.NewString(), map[string]any{
			"name": "Updated Wallet",
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})

	t.Run("Stranger account NotFound", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		f2 := newAuthFixture(t)

		existing := seedAccount(t, f2.DB, defaultAccountParams(f2.User.ID))

		w := f.do(t, http.MethodPatch, "/api/accounts/"+existing.ID, map[string]any{
			"name": "Updated Wallet",
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})
}

func TestDeleteAccount(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		w := f.do(t, http.MethodDelete, "/api/accounts/"+existing.ID, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, 0, w.Body.Len())
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		w := f.do(t, http.MethodDelete, "/api/accounts/"+uuid.NewString(), nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})

	t.Run("AccountWithTransactions", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		category := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		transactionParams := defaultCashflowTransactionParams(f.User.ID, existing.ID, category.ID)
		transactionParams.Type = storage.TransactionTypeIncome
		_ = seedTransaction(t, f.DB, transactionParams)
		w := f.do(t, http.MethodDelete, "/api/accounts/"+existing.ID, nil)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeAccountInUse, response.Code)
		assert.Equal(t, "account in use", response.Message)
	})
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		w := f.do(t, http.MethodGet, "/api/accounts/"+existing.ID, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, existing.Name, response.Name)
		assert.Equal(t, existing.OpeningBalance, response.OpeningBalance)
		assert.Equal(t, existing.ManualAdjustment, response.ManualAdjustment)
		assert.Equal(t, existing.Balance, response.Balance)
	})

	t.Run("AccountWithTransactions", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		existing := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		incomeCategory := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		expenseCategory := seedDefaultExpenseCategory(t, f.DB, f.User.ID)
		expenseTransactionParams := defaultCashflowTransactionParams(
			f.User.ID,
			existing.ID,
			expenseCategory.ID,
		)
		expenseTransactionParams.Amount = 20000
		expenseTransactionParams.Type = storage.TransactionTypeExpense
		transaction1 := seedTransaction(t, f.DB, expenseTransactionParams)
		incomeTransactionParams := defaultCashflowTransactionParams(
			f.User.ID,
			existing.ID,
			incomeCategory.ID,
		)
		incomeTransactionParams.Amount = 10000
		incomeTransactionParams.Type = storage.TransactionTypeIncome
		transaction2 := seedTransaction(t, f.DB, incomeTransactionParams)
		w := f.do(t, http.MethodGet, "/api/accounts/"+existing.ID, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, existing.Name, response.Name)
		assert.Equal(t, existing.OpeningBalance, response.OpeningBalance)
		assert.Equal(t, existing.ManualAdjustment, response.ManualAdjustment)
		assert.Equal(t, existing.Balance-transaction1.Amount+transaction2.Amount, response.Balance)
	})

	t.Run("NotFound", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		w := f.do(t, http.MethodGet, "/api/accounts/"+uuid.NewString(), nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})

	t.Run("Stranger account NotFound", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		f2 := newAuthFixture(t)
		existing := seedAccount(t, f2.DB, defaultAccountParams(f2.User.ID))

		w := f.do(t, http.MethodGet, "/api/accounts/"+existing.ID, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})
}

type seedAccountWithTransactionParams struct {
	openingBalance int64
	income         int64
	expense        int64
}

func seedAccountWithTransaction(
	t *testing.T,
	db *sqlite.Storage,
	userID string,
	params seedAccountWithTransactionParams,
) *storage.Account {
	t.Helper()

	accountParams := defaultAccountParams(userID)
	accountParams.OpeningBalance = params.openingBalance
	account := seedAccount(t, db, accountParams)
	incomeCategory := seedDefaultIncomeCategory(t, db, userID)
	expenseCategory := seedDefaultExpenseCategory(t, db, userID)
	expenseTransactionParams := defaultCashflowTransactionParams(
		userID,
		account.ID,
		expenseCategory.ID,
	)
	expenseTransactionParams.Amount = params.expense
	expenseTransactionParams.Type = storage.TransactionTypeExpense
	_ = seedTransaction(t, db, expenseTransactionParams)
	incomeTransactionParams := defaultCashflowTransactionParams(
		userID,
		account.ID,
		incomeCategory.ID,
	)
	incomeTransactionParams.Amount = params.income
	incomeTransactionParams.Type = storage.TransactionTypeIncome
	_ = seedTransaction(t, db, incomeTransactionParams)

	return account
}

func TestListAccounts(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		params := defaultAccountParams(f.User.ID)
		params.Name = "Wallet"
		seeded1 := seedAccount(t, f.DB, params)
		params.Name = "Bank"
		seeded2 := seedAccount(t, f.DB, params)
		params.Name = "Cash"
		seeded3 := seedAccount(t, f.DB, params)
		params.Name = "Credit Card"
		seeded4 := seedAccount(t, f.DB, params)
		seededAccounts := []*storage.Account{seeded1, seeded2, seeded3, seeded4}

		w := f.do(t, http.MethodGet, "/api/accounts", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Account
		parseBody(t, w, &response)
		assert.Len(t, response, len(seededAccounts))

		accountMap := make(map[string]*storage.Account)
		for _, acc := range seededAccounts {
			accountMap[acc.ID] = acc
		}
		for _, acc := range response {
			account, exists := accountMap[acc.ID]
			assert.True(t, exists)
			assert.Equal(t, *account, acc)
		}
	})

	t.Run("AccountsWithTransactions", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		seeded1 := seedAccountWithTransaction(
			t,
			f.DB,
			f.User.ID,
			seedAccountWithTransactionParams{openingBalance: 100000, income: 50000, expense: 20000},
		)
		seeded2 := seedAccountWithTransaction(
			t,
			f.DB,
			f.User.ID,
			seedAccountWithTransactionParams{openingBalance: 50000, income: 20000, expense: 10000},
		)
		seeded3 := seedAccountWithTransaction(
			t,
			f.DB,
			f.User.ID,
			seedAccountWithTransactionParams{openingBalance: 20000, income: 10000, expense: 5000},
		)
		seeded4 := seedAccountWithTransaction(
			t,
			f.DB,
			f.User.ID,
			seedAccountWithTransactionParams{openingBalance: 0, income: 5000, expense: 2500},
		)
		seededAccounts := []struct {
			*storage.Account

			expectedBalance int64
		}{
			{seeded1, seeded1.OpeningBalance + seeded1.ManualAdjustment + 50000 - 20000},
			{seeded2, seeded2.OpeningBalance + seeded2.ManualAdjustment + 20000 - 10000},
			{seeded3, seeded3.OpeningBalance + seeded3.ManualAdjustment + 10000 - 5000},
			{seeded4, seeded4.OpeningBalance + seeded4.ManualAdjustment + 5000 - 2500},
		}

		w := f.do(t, http.MethodGet, "/api/accounts", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Account
		parseBody(t, w, &response)
		assert.Len(t, response, len(seededAccounts))

		accountMap := make(map[string]struct {
			*storage.Account

			expectedBalance int64
		})
		for _, acc := range seededAccounts {
			accountMap[acc.ID] = acc
		}
		for _, acc := range response {
			account, exists := accountMap[acc.ID]
			assert.True(t, exists)
			assert.Equal(t, account.Name, acc.Name)
			assert.Equal(t, account.OpeningBalance, acc.OpeningBalance)
			assert.Equal(t, account.ManualAdjustment, acc.ManualAdjustment)
			assert.Equal(t, account.expectedBalance, acc.Balance)
		}
	})

	t.Run("NoAccounts", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)

		w := f.do(t, http.MethodGet, "/api/accounts", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Account
		parseBody(t, w, &response)
		assert.Empty(t, response)
	})

	t.Run("Stranger accounts not in user accounts", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		params := defaultAccountParams(f.User.ID)
		params.Name = "Wallet"
		seeded1 := seedAccount(t, f.DB, params)
		params.Name = "Bank"
		seeded2 := seedAccount(t, f.DB, params)
		params.Name = "Cash"
		seeded3 := seedAccount(t, f.DB, params)
		params.Name = "Credit Card"
		seeded4 := seedAccount(t, f.DB, params)
		seededAccounts := []*storage.Account{seeded1, seeded2, seeded3, seeded4}
		f2 := newAuthFixture(t)
		params = defaultAccountParams(f2.User.ID)
		params.Name = "Wallet"
		seeded1 = seedAccount(t, f2.DB, params)
		params.Name = "Bank"
		seeded2 = seedAccount(t, f2.DB, params)
		params.Name = "Cash"
		seeded3 = seedAccount(t, f2.DB, params)
		params.Name = "Credit Card"
		seeded4 = seedAccount(t, f2.DB, params)
		seededAccounts2 := []*storage.Account{seeded1, seeded2, seeded3, seeded4}

		w := f.do(t, http.MethodGet, "/api/accounts", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Account
		parseBody(t, w, &response)
		assert.Len(t, response, len(seededAccounts))

		account2Map := make(map[string]*storage.Account)
		for _, acc := range seededAccounts2 {
			account2Map[acc.ID] = acc
		}
		for _, acc := range response {
			account, exists := account2Map[acc.ID]
			assert.False(t, exists)
			assert.Equal(t, (*storage.Account)(nil), account)
		}
	})
}

func TestGetAccountBalances(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		category := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		account := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		transactionParams := defaultCashflowTransactionParams(f.User.ID, account.ID, category.ID)
		transactionParams.Type = storage.TransactionTypeIncome
		transaction := seedTransaction(t, f.DB, transactionParams)

		w := f.do(t, http.MethodGet, "/api/accounts/balances", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Len(t, response.Balances, 1)
		assert.Equal(t, account.ID, response.Balances[0].ID)
		assert.Equal(t, account.Name, response.Balances[0].Name)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment+transaction.Amount,
			response.Balances[0].Balance,
		)
	})

	t.Run("AccountWithoutTransactions", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		account := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))

		w := f.do(t, http.MethodGet, "/api/accounts/balances", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Len(t, response.Balances, 1)
		assert.Equal(t, account.ID, response.Balances[0].ID)
		assert.Equal(t, account.Name, response.Balances[0].Name)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment,
			response.Balances[0].Balance,
		)
	})

	t.Run("NoBalances", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)

		w := f.do(t, http.MethodGet, "/api/accounts/balances", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Empty(t, response.Balances)
	})

	t.Run("MultipleAccountsWithTransactions", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)

		account1Params := defaultAccountParams(f.User.ID)
		account1Params.Name = "Account1"
		account1 := seedAccount(t, f.DB, account1Params)
		account1, err := f.DB.UpdateAccount(context.Background(), f.User.ID, account1.ID, storage.UpdateAccountParams{
			Name:             new("UpdatedAccount"),
			ManualAdjustment: new(int64(5540)),
		})
		require.NoError(t, err)
		account2Params := defaultAccountParams(f.User.ID)
		account2Params.Name = "Account2"
		account2 := seedAccount(t, f.DB, account2Params)
		incomeCategory := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		expenseCategory := seedDefaultExpenseCategory(t, f.DB, f.User.ID)
		acc1transaction := seedTransaction(t, f.DB, storage.CreateTransactionParams{
			UserID:      f.User.ID,
			Type:        storage.TransactionTypeExpense,
			Amount:      10000,
			Description: "Shopping",
			OccurredAt:  time.Now(),
			AccountID:   &account1.ID,
			CategoryID:  &expenseCategory.ID,
		})
		acc1transaction2 := seedTransaction(t, f.DB, storage.CreateTransactionParams{
			UserID:      f.User.ID,
			Type:        storage.TransactionTypeIncome,
			Amount:      10000,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountID:   &account1.ID,
			CategoryID:  &incomeCategory.ID,
		})
		acc2transaction := seedTransaction(t, f.DB, storage.CreateTransactionParams{
			UserID:      f.User.ID,
			Type:        storage.TransactionTypeIncome,
			Amount:      10000,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountID:   &account2.ID,
			CategoryID:  &incomeCategory.ID,
		})
		acc2transaction2 := seedTransaction(t, f.DB, storage.CreateTransactionParams{
			UserID:        f.User.ID,
			Type:          storage.TransactionTypeTransfer,
			Amount:        5000,
			Description:   "Transfer",
			OccurredAt:    time.Now(),
			FromAccountID: &account2.ID,
			ToAccountID:   &account1.ID,
		})

		w := f.do(t, http.MethodGet, "/api/accounts/balances", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Len(t, response.Balances, 2)
		acc1ExpectedBalance := account1.OpeningBalance + account1.ManualAdjustment - acc1transaction.Amount + acc1transaction2.Amount + acc2transaction2.Amount
		acc2ExpectedBalance := account2.OpeningBalance + account2.ManualAdjustment + acc2transaction.Amount - acc2transaction2.Amount
		assert.Equal(
			t,
			acc1ExpectedBalance+acc2ExpectedBalance,
			response.NetWorth,
		)

		for _, accBalance := range response.Balances {
			if accBalance.ID == account1.ID {
				assert.Equal(t, acc1ExpectedBalance, accBalance.Balance)
			}

			if accBalance.ID == account2.ID {
				assert.Equal(t, acc2ExpectedBalance, accBalance.Balance)
			}
		}
	})

	t.Run("Stranger accounts not in user accounts balance", func(t *testing.T) {
		t.Parallel()
		f := newAuthFixture(t)
		category := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		account := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		transactionParams := defaultCashflowTransactionParams(f.User.ID, account.ID, category.ID)
		transactionParams.Type = storage.TransactionTypeIncome
		fTransaction := seedTransaction(t, f.DB, transactionParams)

		f2 := newAuthFixture(t)
		category2 := seedDefaultIncomeCategory(t, f2.DB, f2.User.ID)
		account2 := seedAccount(t, f2.DB, defaultAccountParams(f2.User.ID))
		transactionParams = defaultCashflowTransactionParams(f2.User.ID, account2.ID, category2.ID)
		transactionParams.Type = storage.TransactionTypeIncome
		_ = seedTransaction(t, f2.DB, transactionParams)

		w := f.do(t, http.MethodGet, "/api/accounts/balances", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Len(t, response.Balances, 1)
		assert.Equal(t, account.ID, response.Balances[0].ID)
		assert.Equal(t, account.Name, response.Balances[0].Name)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment+fTransaction.Amount,
			response.Balances[0].Balance,
		)
	})
}
