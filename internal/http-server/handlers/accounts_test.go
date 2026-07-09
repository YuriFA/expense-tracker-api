package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAccount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodPost, "/api/accounts", map[string]any{
			"name":           "Wallet",
			"currency":       "USD",
			"openingBalance": 100000,
		})
		w := performRequest(t, router, req)

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
		router, _ := setupTestEnv(t)

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
				req := newJSONRequest(t, http.MethodPost, "/api/accounts", tc.body)
				w := performRequest(t, router, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response handlers.ValidationErrorResponse
				parseBody(t, w, &response)
				assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
				assert.Equal(t, "validation failed", response.Message)
				require.Equal(t, tc.errorsLen, len(response.Errors))
				assert.Equal(t, tc.wantField, response.Errors[0].Field)
				assert.Equal(t, tc.wantMessage, response.Errors[0].Message)
			})
		}
	})
}

func TestUpdateAccount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 100000)

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/accounts/"+existing.Id,
			map[string]any{
				"name":             "Updated Wallet",
				"manualAdjustment": 10000,
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Updated Wallet", response.Name)
		assert.Equal(t, int64(10000), response.ManualAdjustment)
		assert.Equal(t, existing.OpeningBalance+10000, response.Balance)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 100000)

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/accounts/"+existing.Id,
			map[string]any{
				"name": "Updated Wallet",
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Updated Wallet", response.Name)
		assert.Equal(t, existing.OpeningBalance, response.OpeningBalance)
		assert.Equal(t, existing.ManualAdjustment, response.ManualAdjustment)
		assert.Equal(t, existing.Balance, response.Balance)
	})

	t.Run("AccountWithTransactions", func(t *testing.T) {
		router, db := setupTestEnv(t)

		user := seedUser(t, db, "test@example.com")
		existing := seedAccount(t, db, "Wallet", 100000)
		incomeCategory := seedCategory(t, db, "salary", user.Id, "income")
		expenseCategory := seedCategory(t, db, "rent", user.Id, "expense")
		incomeTransaction := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "income",
			Amount:      10000,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountId:   &existing.Id,
			CategoryId:  &incomeCategory.Id,
		})
		expenseTransaction := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "expense",
			Amount:      25000,
			Description: "Groceries",
			OccurredAt:  time.Now(),
			AccountId:   &existing.Id,
			CategoryId:  &expenseCategory.Id,
		})

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/accounts/"+existing.Id,
			map[string]any{
				"name": "Updated Wallet",
			},
		)
		w := performRequest(t, router, req)

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
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 100000)

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/accounts/"+existing.Id,
			map[string]any{
				"name": "qw",
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response handlers.ValidationErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
		assert.Equal(t, "validation failed", response.Message)
		assert.Equal(t, 1, len(response.Errors))
		assert.Equal(t, "name", response.Errors[0].Field)
		assert.Equal(
			t,
			"name must be at least 3 characters",
			response.Errors[0].Message,
		)
	})

	t.Run("NoFields", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 100000)

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/accounts/"+existing.Id,
			map[string]any{},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
		assert.Equal(t, "no fields to update", response.Message)
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/accounts/"+uuid.NewString(),
			map[string]any{
				"name": "Updated Wallet",
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})
}

func TestDeleteAccount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 100000)

		req := httptest.NewRequest(http.MethodDelete, "/api/accounts/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, 0, w.Body.Len())
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodDelete, "/api/accounts/"+uuid.NewString(), nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})

	t.Run("AccountWithTransactions", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 100000)
		user := seedUser(t, db, "user@example.com")
		category := seedCategory(t, db, "salary", user.Id, "income")
		_ = seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "income",
			Amount:      10000,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountId:   &existing.Id,
			CategoryId:  &category.Id,
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/accounts/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeAccountInUse, response.Code)
		assert.Equal(t, "account in use", response.Message)
	})
}

func TestGetAccount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 100000)

		req := httptest.NewRequest(http.MethodGet, "/api/accounts/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Wallet", response.Name)
		assert.Equal(t, int64(100000), response.OpeningBalance)
		assert.Equal(t, int64(0), response.ManualAdjustment)
		assert.Equal(t, int64(100000), response.Balance)
	})

	t.Run("AccountWithTransactions", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 100000)
		user := seedUser(t, db, "user@example.com")
		incomeCategory := seedCategory(t, db, "salary", user.Id, "income")
		expenseCategory := seedCategory(t, db, "rent", user.Id, "expense")
		transaction1 := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "expense",
			Amount:      10000,
			Description: "Shopping",
			OccurredAt:  time.Now(),
			AccountId:   &existing.Id,
			CategoryId:  &expenseCategory.Id,
		})
		transaction2 := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "income",
			Amount:      10000,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountId:   &existing.Id,
			CategoryId:  &incomeCategory.Id,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/accounts/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Wallet", response.Name)
		assert.Equal(t, int64(100000), response.OpeningBalance)
		assert.Equal(t, int64(0), response.ManualAdjustment)
		assert.Equal(t, existing.Balance-transaction1.Amount+transaction2.Amount, response.Balance)
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodGet, "/api/accounts/"+uuid.NewString(), nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeAccountNotFound, response.Code)
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
	categoryName string,
	userId string,
	params seedAccountWithTransactionParams,
) *storage.Account {
	t.Helper()

	account := seedAccount(t, db, "Wallet", params.openingBalance)
	incomeCategory := seedCategory(t, db, fmt.Sprintf("income_%s", categoryName), userId, "income")
	expenseCategory := seedCategory(
		t,
		db,
		fmt.Sprintf("expense_%s", categoryName),
		userId,
		"expense",
	)
	_ = seedTransaction(t, db, storage.CreateTransactionParams{
		Type:        "expense",
		Amount:      params.expense,
		Description: "Shopping",
		OccurredAt:  time.Now(),
		AccountId:   &account.Id,
		CategoryId:  &expenseCategory.Id,
	})
	_ = seedTransaction(t, db, storage.CreateTransactionParams{
		Type:        "income",
		Amount:      params.income,
		Description: "Salary",
		OccurredAt:  time.Now(),
		AccountId:   &account.Id,
		CategoryId:  &incomeCategory.Id,
	})

	return account
}

func TestListAccounts(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		seeded1 := seedAccount(t, db, "Wallet", 100000)
		seeded2 := seedAccount(t, db, "Bank", 500000)
		seeded3 := seedAccount(t, db, "Cash", 20000)
		seeded4 := seedAccount(t, db, "Credit Card", 0)
		seededAccounts := []*storage.Account{seeded1, seeded2, seeded3, seeded4}

		req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, len(seededAccounts), len(response))

		accountMap := make(map[string]*storage.Account)
		for _, acc := range seededAccounts {
			accountMap[acc.Id] = acc
		}
		for _, acc := range response {
			account, exists := accountMap[acc.Id]
			assert.Equal(t, true, exists)
			assert.Equal(t, *account, acc)
		}
	})

	t.Run("AccountsWithTransactions", func(t *testing.T) {
		router, db := setupTestEnv(t)

		user := seedUser(t, db, "test@example.com")
		seeded1 := seedAccountWithTransaction(
			t,
			db,
			"category1",
			user.Id,
			seedAccountWithTransactionParams{openingBalance: 100000, income: 50000, expense: 20000},
		)
		seeded2 := seedAccountWithTransaction(
			t,
			db,
			"category2",
			user.Id,
			seedAccountWithTransactionParams{openingBalance: 50000, income: 20000, expense: 10000},
		)
		seeded3 := seedAccountWithTransaction(
			t,
			db,
			"category3",
			user.Id,
			seedAccountWithTransactionParams{openingBalance: 20000, income: 10000, expense: 5000},
		)
		seeded4 := seedAccountWithTransaction(
			t,
			db,
			"category4",
			user.Id,
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

		req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, len(seededAccounts), len(response))

		accountMap := make(map[string]struct {
			*storage.Account
			expectedBalance int64
		})
		for _, acc := range seededAccounts {
			accountMap[acc.Id] = acc
		}
		for _, acc := range response {
			account, exists := accountMap[acc.Id]
			assert.Equal(t, true, exists)
			assert.Equal(t, account.Name, acc.Name)
			assert.Equal(t, account.OpeningBalance, acc.OpeningBalance)
			assert.Equal(t, account.ManualAdjustment, acc.ManualAdjustment)
			assert.Equal(t, account.expectedBalance, acc.Balance)
		}
	})

	t.Run("NoAccounts", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodGet, "/api/accounts", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, 0, len(response))
	})
}

func TestGetAccountBalances(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.Id, "income")
		account := seedAccount(t, db, "Wallet", 100000)
		transaction := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "income",
			Amount:      10000,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountId:   &account.Id,
			CategoryId:  &category.Id,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/accounts/balances", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Equal(t, 1, len(response.Balances))
		assert.Equal(t, account.Id, response.Balances[0].Id)
		assert.Equal(t, account.Name, response.Balances[0].Name)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment+transaction.Amount,
			response.Balances[0].Balance,
		)
	})

	t.Run("AccountWithoutTransactions", func(t *testing.T) {
		router, db := setupTestEnv(t)

		account := seedAccount(t, db, "Account1", 100000)

		req := httptest.NewRequest(http.MethodGet, "/api/accounts/balances", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Equal(t, 1, len(response.Balances))
		assert.Equal(t, account.Id, response.Balances[0].Id)
		assert.Equal(t, account.Name, response.Balances[0].Name)
		assert.Equal(
			t,
			account.OpeningBalance+account.ManualAdjustment,
			response.Balances[0].Balance,
		)
	})

	t.Run("NoBalances", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodGet, "/api/accounts/balances", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Equal(t, 0, len(response.Balances))
	})

	t.Run("MultipleAccountsWithTransactions", func(t *testing.T) {
		router, db := setupTestEnv(t)

		account1 := seedAccount(t, db, "Account1", 100000)
		account1, err := db.UpdateAccount(account1.Id, storage.UpdateAccountParams{
			Name:             new("UpdatedAccount"),
			ManualAdjustment: new(int64(5540)),
		})
		require.NoError(t, err)
		account2 := seedAccount(t, db, "Account2", 50000)
		user := seedUser(t, db, "test@example.com")
		incomeCategory := seedCategory(t, db, "salary", user.Id, "income")
		expenseCategory := seedCategory(t, db, "rent", user.Id, "expense")
		acc1transaction := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "expense",
			Amount:      10000,
			Description: "Shopping",
			OccurredAt:  time.Now(),
			AccountId:   &account1.Id,
			CategoryId:  &expenseCategory.Id,
		})
		acc1transaction2 := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "income",
			Amount:      10000,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountId:   &account1.Id,
			CategoryId:  &incomeCategory.Id,
		})
		acc2transaction := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "income",
			Amount:      10000,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountId:   &account2.Id,
			CategoryId:  &incomeCategory.Id,
		})
		acc2transaction2 := seedTransaction(t, db, storage.CreateTransactionParams{
			Type:          "transfer",
			Amount:        5000,
			Description:   "Transfer",
			OccurredAt:    time.Now(),
			FromAccountId: &account2.Id,
			ToAccountId:   &account1.Id,
		})

		req := httptest.NewRequest(http.MethodGet, "/api/accounts/balances", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response struct {
			Balances []storage.AccountBalance `json:"balances"`
			NetWorth int64                    `json:"netWorth"`
		}
		parseBody(t, w, &response)
		assert.Equal(t, 2, len(response.Balances))
		acc1ExpectedBalance := account1.OpeningBalance + account1.ManualAdjustment - acc1transaction.Amount + acc1transaction2.Amount + acc2transaction2.Amount
		acc2ExpectedBalance := account2.OpeningBalance + account2.ManualAdjustment + acc2transaction.Amount - acc2transaction2.Amount
		assert.Equal(
			t,
			acc1ExpectedBalance+acc2ExpectedBalance,
			response.NetWorth,
		)

		for _, accBalance := range response.Balances {
			if accBalance.Id == account1.Id {
				assert.Equal(t, acc1ExpectedBalance, accBalance.Balance)
			}

			if accBalance.Id == account2.Id {
				assert.Equal(t, acc2ExpectedBalance, accBalance.Balance)
			}

		}
	})
}
