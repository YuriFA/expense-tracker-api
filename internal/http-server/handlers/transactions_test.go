package handlers_test

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTransaction(t *testing.T) {
	t.Run("Success cashflow", func(t *testing.T) {
		f := newAuthFixture(t)
		occurredAt := time.Now()
		category := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		account := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))

		w := f.do(t, http.MethodPost, "/api/transactions", map[string]any{
			"type":        "income",
			"amount":      100000,
			"description": "Salary for June",
			"occurredAt":  occurredAt,
			"accountId":   account.ID,
			"categoryId":  category.ID,
		})

		assert.Equal(t, http.StatusCreated, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, "income", response.Type)
		assert.Equal(t, int64(100000), response.Amount)
		assert.Equal(t, "Salary for June", response.Description)
		testutil.AssertTimeEqual(t, occurredAt, testutil.ParseDatetime(t, response.OccurredAt))
		assert.Equal(t, account.ID, *response.AccountID)
		assert.Equal(t, category.ID, *response.CategoryID)
		assert.Nil(t, response.FromAccountID)
		assert.Nil(t, response.ToAccountID)
	})

	t.Run("Success transfer", func(t *testing.T) {
		f := newAuthFixture(t)
		occurredAt := time.Now()
		accountParams := defaultAccountParams(f.User.ID)
		accountParams.Name = "Card"
		account := seedAccount(t, f.DB, accountParams)
		account2Params := defaultAccountParams(f.User.ID)
		account2Params.Name = "Bank"
		account2 := seedAccount(t, f.DB, account2Params)

		w := f.do(t, http.MethodPost, "/api/transactions", map[string]any{
			"type":          "transfer",
			"amount":        100000,
			"description":   "Salary for June",
			"occurredAt":    occurredAt,
			"fromAccountId": account.ID,
			"toAccountId":   account2.ID,
		})

		assert.Equal(t, http.StatusCreated, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, "transfer", response.Type)
		assert.Equal(t, int64(100000), response.Amount)
		assert.Equal(t, "Salary for June", response.Description)
		testutil.AssertTimeEqual(t, occurredAt, testutil.ParseDatetime(t, response.OccurredAt))
		assert.Equal(t, account.ID, *response.FromAccountID)
		assert.Equal(t, account2.ID, *response.ToAccountID)
		assert.Nil(t, response.AccountID)
		assert.Nil(t, response.CategoryID)
	})

	t.Run("ValidationFail", func(t *testing.T) {
		f := newAuthFixture(t)
		occurredAt := time.Now()
		category := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		accountParams := defaultAccountParams(f.User.ID)
		accountParams.Name = "Cash"
		account := seedAccount(t, f.DB, accountParams)
		account2Params := defaultAccountParams(f.User.ID)
		account2Params.Name = "Bank"
		account2 := seedAccount(t, f.DB, account2Params)

		cases := map[string]struct {
			body        map[string]any
			wantField   string
			wantMessage string
			errorsLen   int
		}{
			"cashflow without accountId": {
				body: map[string]any{
					"type":        "income",
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"categoryId":  category.ID,
				},
				wantField:   "accountId",
				wantMessage: "accountId is required",
				errorsLen:   1,
			},
			"cashflow without categoryId": {
				body: map[string]any{
					"type":        "income",
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.ID,
				},
				wantField:   "categoryId",
				wantMessage: "categoryId is required",
				errorsLen:   1,
			},
			"cashflow + fromAccountId": {
				body: map[string]any{
					"type":          "income",
					"amount":        100000,
					"description":   "Salary for June",
					"occurredAt":    occurredAt,
					"accountId":     account.ID,
					"categoryId":    category.ID,
					"fromAccountId": uuid.NewString(),
				},
				wantField:   "fromAccountId",
				wantMessage: "not allowed for income or expense transactions",
				errorsLen:   1,
			},
			"transfer without fromAccountId": {
				body: map[string]any{
					"type":        "transfer",
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"toAccountId": account2.ID,
				},
				wantField:   "fromAccountId",
				wantMessage: "fromAccountId is required",
				errorsLen:   1,
			},
			"transfer without toAccountId": {
				body: map[string]any{
					"type":          "transfer",
					"amount":        100000,
					"description":   "Salary for June",
					"occurredAt":    occurredAt,
					"fromAccountId": account.ID,
				},
				wantField:   "toAccountId",
				wantMessage: "toAccountId is required",
				errorsLen:   1,
			},
			"transfer with categoryId": {
				body: map[string]any{
					"type":          "transfer",
					"amount":        100000,
					"description":   "Salary for June",
					"fromAccountId": account.ID,
					"occurredAt":    occurredAt,
					"toAccountId":   account2.ID,
					"categoryId":    category.ID,
				},
				wantField:   "categoryId",
				wantMessage: "not allowed for transfer transactions",
				errorsLen:   1,
			},
			"missing type": {
				body: map[string]any{
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.ID,
					"categoryId":  category.ID,
				},
				wantField:   "type",
				wantMessage: "type is required",
				errorsLen:   1,
			},
			"missing amount": {
				body: map[string]any{
					"type":        "income",
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.ID,
					"categoryId":  category.ID,
				},
				wantField:   "amount",
				wantMessage: "amount is required",
				errorsLen:   1,
			},
			"missing occurredAt": {
				body: map[string]any{
					"type":        "income",
					"amount":      100000,
					"description": "Salary for June",
					"accountId":   account.ID,
					"categoryId":  category.ID,
				},
				wantField:   "occurredAt",
				wantMessage: "occurredAt is required",
				errorsLen:   1,
			},
			"missing accountId": {
				body: map[string]any{
					"type":        "income",
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"categoryId":  category.ID,
				},
				wantField:   "accountId",
				wantMessage: "accountId is required",
				errorsLen:   1,
			},
			"missing categoryId": {
				body: map[string]any{
					"type":        "income",
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.ID,
				},
				wantField:   "categoryId",
				wantMessage: "categoryId is required",
				errorsLen:   1,
			},
			"invalid accountId": {
				body: map[string]any{
					"type":        "income",
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   "invalid-id",
					"categoryId":  category.ID,
				},
				wantField:   "accountId",
				wantMessage: "accountId must be a valid UUID",
				errorsLen:   1,
			},
			"invalid categoryId": {
				body: map[string]any{
					"type":        "income",
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.ID,
					"categoryId":  "invalid-id",
				},
				wantField:   "categoryId",
				wantMessage: "categoryId must be a valid UUID",
				errorsLen:   1,
			},
			"zero amount": {
				body: map[string]any{
					"type":        "income",
					"amount":      0,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.ID,
					"categoryId":  category.ID,
				},
				wantField:   "amount",
				wantMessage: "amount is required",
				errorsLen:   1,
			},
			"negative amount": {
				body: map[string]any{
					"type":        "income",
					"amount":      -100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.ID,
					"categoryId":  category.ID,
				},
				wantField:   "amount",
				wantMessage: "amount must be greater than 0",
				errorsLen:   1,
			},
			"empty body": {
				body:        map[string]any{},
				wantField:   "type",
				wantMessage: "type is required",
				errorsLen:   3,
			},
			"wrong type": {
				body: map[string]any{
					"type":        "outcome",
					"amount":      100000,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.ID,
					"categoryId":  category.ID,
				},
				wantField:   "type",
				wantMessage: "type must be either 'income' or 'expense' or 'transfer'",
				errorsLen:   1,
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				w := f.do(t, http.MethodPost, "/api/transactions", tc.body)

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

	t.Run("NonExistAccountForCashflow", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodPost, "/api/transactions", map[string]any{
			"type":        "income",
			"amount":      100000,
			"description": "Salary for June",
			"occurredAt":  time.Now(),
			"accountId":   uuid.NewString(),
			"categoryId":  uuid.NewString(),
		})

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})

	t.Run("NonExistAccountForTransfer", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodPost, "/api/transactions", map[string]any{
			"type":          "transfer",
			"amount":        100000,
			"description":   "Transfer to bank",
			"occurredAt":    time.Now(),
			"fromAccountId": uuid.NewString(),
			"toAccountId":   uuid.NewString(),
		})

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})

	t.Run("SameAccountForTransfer", func(t *testing.T) {
		f := newAuthFixture(t)
		account := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))

		w := f.do(
			t, http.MethodPost, "/api/transactions", map[string]any{
				"type":          "transfer",
				"amount":        100000,
				"description":   "Transfer to bank",
				"occurredAt":    time.Now(),
				"fromAccountId": account.ID,
				"toAccountId":   account.ID,
			},
		)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeSameAccountTransfer, response.Code)
		assert.Equal(t, "transaction from and to accounts are the same", response.Message)
	})
}

func TestUpdateTransaction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "transport",
			transactionType: "expense",
		})
		nextAccount := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		nextCategory := seedDefaultExpenseCategory(t, f.DB, f.User.ID)

		params := map[string]any{
			"amount":      int64(50000),
			"description": "Some expense",
			"occurredAt":  time.Now(),
			"accountId":   nextAccount.ID,
			"categoryId":  nextCategory.ID,
		}
		w := f.do(t, http.MethodPatch, "/api/transactions/"+existing.ID, params)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, params["amount"], response.Amount)
		assert.Equal(t, params["description"], response.Description)
		testutil.AssertTimeEqual(
			t,
			params["occurredAt"].(time.Time),
			testutil.ParseDatetime(t, response.OccurredAt),
		)
		assert.Equal(t, params["accountId"], *response.AccountID)
		assert.Equal(t, params["categoryId"], *response.CategoryID)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
		})

		w := f.do(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.ID,
			map[string]any{"description": "Updated Transaction"},
		)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, existing.Amount, response.Amount)
		assert.Equal(t, "Updated Transaction", response.Description)
		assert.Equal(t, existing.OccurredAt, response.OccurredAt)
		assert.Equal(t, existing.AccountID, response.AccountID)
		assert.Equal(t, existing.CategoryID, response.CategoryID)
	})

	t.Run("NoFields", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
		})

		w := f.do(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.ID,
			map[string]any{},
		)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
		assert.Equal(t, "no fields to update", response.Message)
	})

	t.Run("NonExistAccount", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
		})

		w := f.do(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.ID,
			map[string]any{
				"description": "Updated Transaction",
				"accountId":   uuid.NewString(),
			},
		)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})

	t.Run("TypeParamIgnored", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
		})

		w := f.do(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.ID,
			map[string]any{
				"type":        "expense",
				"description": "Updated Transaction",
			},
		)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, "Updated Transaction", response.Description)
	})

	t.Run("NonExistCategory", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
		})

		w := f.do(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.ID,
			map[string]any{
				"description": "Updated Transaction",
				"categoryId":  uuid.NewString(),
			},
		)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})

	t.Run("CategoryTypeMismatch", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
		})

		nextCategory := seedDefaultExpenseCategory(t, f.DB, f.User.ID)

		w := f.do(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.ID,
			map[string]any{
				"description": "Updated Transaction",
				"categoryId":  nextCategory.ID,
			},
		)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeCategoryTypeMismatch, response.Code)
		assert.Equal(t, "transaction type does not match category type", response.Message)
	})

	t.Run("NotFound", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(
			t,
			http.MethodPatch,
			"/api/transactions/"+uuid.NewString(),
			map[string]any{
				"amount": 10000,
			},
		)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeTransactionNotFound, response.Code)
		assert.Equal(t, "transaction not found", response.Message)
	})

	t.Run("ShapeViolation", func(t *testing.T) {
		f := newAuthFixture(t)
		cashflowTransaction := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
		})
		transferTransaction := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "transfer",
			transactionType: "transfer",
		})

		cases := map[string]struct {
			id          string
			body        map[string]any
			wantField   string
			wantMessage string
		}{
			"cashflow with fromAccountId": {
				id: cashflowTransaction.ID,
				body: map[string]any{
					"fromAccountId": uuid.NewString(),
				},
				wantField:   "fromAccountId",
				wantMessage: "not allowed for income or expense transactions",
			},
			"cashflow with toAccountId": {
				id: cashflowTransaction.ID,
				body: map[string]any{
					"toAccountId": uuid.NewString(),
				},
				wantField:   "toAccountId",
				wantMessage: "not allowed for income or expense transactions",
			},
			"transfer with accountId": {
				id: transferTransaction.ID,
				body: map[string]any{
					"accountId": uuid.NewString(),
				},
				wantField:   "accountId",
				wantMessage: "not allowed for transfer transactions",
			},
			"transfer with categoryId": {
				id: transferTransaction.ID,
				body: map[string]any{
					"categoryId": uuid.NewString(),
				},
				wantField:   "categoryId",
				wantMessage: "not allowed for transfer transactions",
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				w := f.do(
					t,
					http.MethodPatch,
					"/api/transactions/"+tc.id,
					tc.body,
				)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response handlers.ValidationErrorResponse
				parseBody(t, w, &response)
				assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
				assert.Equal(t, "validation failed", response.Message)
				assert.Equal(t, 1, len(response.Errors))
				assert.Equal(t, tc.wantField, response.Errors[0].Field)
				assert.Equal(t, tc.wantMessage, response.Errors[0].Message)
			})
		}
	})
}

func TestDeleteTransaction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			transactionType: "income",
		})

		w := f.do(t, http.MethodDelete, "/api/transactions/"+existing.ID, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, 0, w.Body.Len())
	})

	t.Run("NotFound", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodDelete, "/api/transactions/"+uuid.NewString(), nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeTransactionNotFound, response.Code)
		assert.Equal(t, "transaction not found", response.Message)
	})

	t.Run("Stranger transaction NotFound", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			transactionType: "income",
		})
		f2 := newAuthFixture(t)

		w := f2.do(t, http.MethodDelete, "/api/transactions/"+existing.ID, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeTransactionNotFound, response.Code)
		assert.Equal(t, "transaction not found", response.Message)
	})
}

func TestGetTransaction(t *testing.T) {
	t.Run("Success for cashflow", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			transactionType: "income",
		})

		w := f.do(t, http.MethodGet, "/api/transactions/"+existing.ID, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, existing.ID, response.ID)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, existing.Amount, response.Amount)
		assert.Equal(t, existing.Description, response.Description)
		assert.Equal(t, existing.OccurredAt, response.OccurredAt)
		assert.Equal(t, existing.AccountID, response.AccountID)
		assert.Equal(t, existing.CategoryID, response.CategoryID)
		assert.Nil(t, response.FromAccountID)
		assert.Nil(t, response.ToAccountID)
	})

	t.Run("Success for transfer", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			transactionType: "transfer",
		})

		w := f.do(t, http.MethodGet, "/api/transactions/"+existing.ID, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, existing.ID, response.ID)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, existing.Amount, response.Amount)
		assert.Equal(t, existing.Description, response.Description)
		assert.Equal(t, existing.FromAccountID, response.FromAccountID)
		assert.Equal(t, existing.ToAccountID, response.ToAccountID)
		assert.Equal(t, existing.OccurredAt, response.OccurredAt)
		assert.Nil(t, response.AccountID)
		assert.Nil(t, response.CategoryID)
	})

	t.Run("NotFound", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodGet, "/api/transactions/"+uuid.NewString(), nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeTransactionNotFound, response.Code)
		assert.Equal(t, "transaction not found", response.Message)
	})

	t.Run("Stranger transaction NotFound", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			transactionType: "income",
		})
		f2 := newAuthFixture(t)

		w := f2.do(t, http.MethodGet, "/api/transactions/"+existing.ID, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeTransactionNotFound, response.Code)
		assert.Equal(t, "transaction not found", response.Message)
	})
}

func TestListTransactions(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)
		seeded1 := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
		})
		seeded2 := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "groceries",
			transactionType: "expense",
		})
		seeded3 := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "freelance",
			transactionType: "income",
		})
		seeded4 := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "transfer",
			transactionType: "transfer",
		})
		seeded5 := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "expense",
			transactionType: "expense",
		})
		seeded6 := seedCommonTransaction(t, f.DB, seedCommonTransactionParams{
			userID:          f.User.ID,
			categoryName:    "transfer",
			transactionType: "transfer",
		})
		seededTransactions := []*storage.Transaction{
			seeded1,
			seeded2,
			seeded3,
			seeded4,
			seeded5,
			seeded6,
		}

		w := f.do(t, http.MethodGet, "/api/transactions", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, len(seededTransactions), len(response))

		transactionsMap := make(map[string]*storage.Transaction)
		for _, trx := range seededTransactions {
			transactionsMap[trx.ID] = trx
		}
		for _, trx := range response {
			transaction, exists := transactionsMap[trx.ID]
			assert.Equal(t, true, exists)
			assert.Equal(t, *transaction, trx)
		}
	})

	t.Run("SpecificParamsWithoutDateRange", func(t *testing.T) {
		f := newAuthFixture(t)
		seeded1 := seedTransactionAt(
			t,
			f.DB,
			seedTransactionAtParams{
				userID:          f.User.ID,
				categoryName:    "salary",
				transactionType: "income",
				occurredAt:      time.Date(2024, 5, 10, 12, 0, 0, 0, time.UTC),
				amount:          80000,
			},
		)
		seeded2 := seedTransactionAt(
			t,
			f.DB,
			seedTransactionAtParams{
				userID:          f.User.ID,
				categoryName:    "groceries",
				transactionType: "expense",
				occurredAt:      time.Date(2024, 5, 15, 12, 14, 30, 0, time.UTC),
				amount:          20000,
			},
		)
		seeded3 := seedTransactionAt(
			t,
			f.DB,
			seedTransactionAtParams{
				userID:          f.User.ID,
				transactionType: "income",
				categoryName:    "freelance",
				occurredAt:      time.Date(2024, 4, 1, 12, 0, 0, 0, time.UTC),
				amount:          1000000,
			},
		)
		seeded4 := seedTransactionAt(
			t,
			f.DB,
			seedTransactionAtParams{
				userID:          f.User.ID,
				transactionType: "expense",
				categoryName:    "groceries2",
				occurredAt:      time.Date(2024, 5, 15, 12, 12, 30, 0, time.UTC),
				amount:          300000,
			},
		)
		seeded5 := seedTransactionAt(
			t,
			f.DB,
			seedTransactionAtParams{
				userID:          f.User.ID,
				categoryName:    "transfer2",
				transactionType: "transfer",
				occurredAt:      time.Date(2024, 2, 10, 12, 15, 30, 0, time.UTC),
				amount:          300000,
			},
		)

		cases := []struct {
			params   map[string]string
			expected []*storage.Transaction
		}{
			{
				params:   map[string]string{"type": "income"},
				expected: []*storage.Transaction{seeded1, seeded3},
			},
			{
				params:   map[string]string{"type": "expense"},
				expected: []*storage.Transaction{seeded2, seeded4},
			},
			{
				params:   map[string]string{"accountId": *seeded1.AccountID},
				expected: []*storage.Transaction{seeded1},
			},
			{
				params:   map[string]string{"categoryId": *seeded2.CategoryID},
				expected: []*storage.Transaction{seeded2},
			},
			{
				params:   map[string]string{"type": "income", "accountId": *seeded3.AccountID},
				expected: []*storage.Transaction{seeded3},
			},
			{
				params:   map[string]string{"type": "expense", "categoryId": *seeded4.CategoryID},
				expected: []*storage.Transaction{seeded4},
			},
			{
				params: map[string]string{
					"categoryId": *seeded1.CategoryID,
				},
				expected: []*storage.Transaction{seeded1},
			},
			{
				params: map[string]string{
					"type":       "income",
					"accountId":  *seeded1.AccountID,
					"categoryId": *seeded1.CategoryID,
				},
				expected: []*storage.Transaction{seeded1},
			},
			{
				params: map[string]string{
					"type":       "expense",
					"accountId":  *seeded2.AccountID,
					"categoryId": *seeded2.CategoryID,
				},
				expected: []*storage.Transaction{seeded2},
			},
			{
				params:   map[string]string{"type": "income", "limit": "1"},
				expected: []*storage.Transaction{seeded1},
			},
			{
				params:   map[string]string{"type": "expense", "limit": "1"},
				expected: []*storage.Transaction{seeded2},
			},
			{
				params:   map[string]string{"type": "income", "sort": "-amount"},
				expected: []*storage.Transaction{seeded3, seeded1},
			},
			{
				params:   map[string]string{"type": "expense", "sort": "-amount"},
				expected: []*storage.Transaction{seeded4, seeded2},
			},
			{
				params:   map[string]string{"type": "income", "sort": "amount"},
				expected: []*storage.Transaction{seeded1, seeded3},
			},
			{
				params:   map[string]string{"type": "expense", "sort": "amount"},
				expected: []*storage.Transaction{seeded2, seeded4},
			},
			{
				params:   map[string]string{"type": "income", "limit": "1", "sort": "-amount"},
				expected: []*storage.Transaction{seeded3},
			},
			{
				params:   map[string]string{"type": "income", "sort": "-occurredAt"},
				expected: []*storage.Transaction{seeded1, seeded3},
			},
			{
				params:   map[string]string{"type": "income", "sort": "occurredAt"},
				expected: []*storage.Transaction{seeded3, seeded1},
			},
			{
				params: map[string]string{
					"accountId": *seeded5.FromAccountID,
				},
				expected: []*storage.Transaction{seeded5},
			},
		}

		for _, tc := range cases {
			params := url.Values{}
			for key, value := range tc.params {
				params.Add(key, value)
			}
			paramsEncoded := params.Encode()

			t.Run(fmt.Sprintf("Params: %v", params), func(t *testing.T) {
				w := f.do(t, http.MethodGet, "/api/transactions"+"?"+paramsEncoded, nil)

				assert.Equal(t, http.StatusOK, w.Code)
				var response []storage.Transaction
				parseBody(t, w, &response)
				assert.Equal(t, len(tc.expected), len(response))
				for i, trx := range response {
					assert.Equal(t, tc.expected[i].ID, trx.ID)
				}
			})
		}
	})

	t.Run("DateRange", func(t *testing.T) {
		f := newAuthFixture(t)
		beforeRange := seedTransactionAt(t, f.DB, seedTransactionAtParams{
			userID:          f.User.ID,
			categoryName:    "salary",
			transactionType: "income",
			occurredAt:      time.Date(2024, 5, 15, 12, 0, 0, 0, time.UTC),
			amount:          10000,
		})
		onUpperBoundary := seedTransactionAt(
			t,
			f.DB,
			seedTransactionAtParams{
				userID:          f.User.ID,
				categoryName:    "groceries",
				transactionType: "expense",
				occurredAt:      time.Date(2024, 6, 30, 23, 59, 0, 0, time.UTC),
				amount:          10000,
			},
		)
		justAfterUpper := seedTransactionAt(
			t,
			f.DB,
			seedTransactionAtParams{
				userID:          f.User.ID,
				categoryName:    "salary2",
				transactionType: "income",
				occurredAt:      time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
				amount:          10000,
			},
		)
		onLowerBoundary := seedTransactionAt(t, f.DB, seedTransactionAtParams{
			userID:          f.User.ID,
			categoryName:    "salary3",
			transactionType: "income",
			occurredAt:      time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
			amount:          10000,
		})
		inMiddle := seedTransactionAt(t, f.DB, seedTransactionAtParams{
			userID:          f.User.ID,
			categoryName:    "groceries2",
			transactionType: "expense",
			occurredAt:      time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC),
			amount:          10000,
		})
		afterRange := seedTransactionAt(t, f.DB, seedTransactionAtParams{
			userID:          f.User.ID,
			categoryName:    "salary4",
			transactionType: "income",
			occurredAt:      time.Date(2024, 7, 15, 12, 0, 0, 0, time.UTC),
			amount:          10000,
		})

		w := f.do(t, http.MethodGet, "/api/transactions?fromDate=2024-06-01&toDate=2024-06-30", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Transaction
		parseBody(t, w, &response)

		var ids []string
		for _, trx := range response {
			ids = append(ids, trx.ID)
		}

		assert.NotContains(t, ids, beforeRange.ID)
		assert.NotContains(t, ids, justAfterUpper.ID)
		assert.NotContains(t, ids, afterRange.ID)
		assert.Contains(t, ids, onLowerBoundary.ID)
		assert.Contains(t, ids, inMiddle.ID)
		assert.Contains(t, ids, onUpperBoundary.ID)
	})
}
