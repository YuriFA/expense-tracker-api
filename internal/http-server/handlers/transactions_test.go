package handlers_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateTransaction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		transactionType := "income"
		occurredAt := time.Now()
		category, account := seedCommonCategoryAndAccount(t, db, transactionType)

		req := newJSONRequest(t, http.MethodPost, "/api/transactions", map[string]any{
			"type":        transactionType,
			"amount":      1000.0,
			"description": "Salary for June",
			"occurredAt":  occurredAt,
			"accountId":   account.Id,
			"categoryId":  category.Id,
		})
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, "income", response.Type)
		assert.Equal(t, 1000.0, response.Amount)
		assert.Equal(t, "Salary for June", response.Description)
		testutil.AssertTimeEqual(t, occurredAt, testutil.ParseDatetime(t, response.OccurredAt))
		assert.Equal(t, account.Id, response.AccountId)
		assert.Equal(t, category.Id, response.CategoryId)
	})

	t.Run("ValidationFail", func(t *testing.T) {
		router, db := setupTestEnv(t)

		occurredAt := time.Now()
		category, account := seedCommonCategoryAndAccount(t, db, "income")

		cases := map[string]struct {
			body        map[string]any
			wantField   string
			wantMessage string
			errorsLen   int
		}{
			"missing type": {
				body: map[string]any{
					"amount":      1000.0,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.Id,
					"categoryId":  category.Id,
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
					"accountId":   account.Id,
					"categoryId":  category.Id,
				},
				wantField:   "amount",
				wantMessage: "amount is required",
				errorsLen:   1,
			},
			"missing occurredAt": {
				body: map[string]any{
					"type":        "income",
					"amount":      1000.0,
					"description": "Salary for June",
					"accountId":   account.Id,
					"categoryId":  category.Id,
				},
				wantField:   "occurredAt",
				wantMessage: "occurredAt is required",
				errorsLen:   1,
			},
			"missing accountId": {
				body: map[string]any{
					"type":        "income",
					"amount":      1000.0,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"categoryId":  category.Id,
				},
				wantField:   "accountId",
				wantMessage: "accountId is required",
				errorsLen:   1,
			},
			"missing categoryId": {
				body: map[string]any{
					"type":        "income",
					"amount":      1000.0,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.Id,
				},
				wantField:   "categoryId",
				wantMessage: "categoryId is required",
				errorsLen:   1,
			},
			"invalid accountId": {
				body: map[string]any{
					"type":        "income",
					"amount":      1000.0,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   "invalid-id",
					"categoryId":  category.Id,
				},
				wantField:   "accountId",
				wantMessage: "accountId must be a valid UUID",
				errorsLen:   1,
			},
			"invalid categoryId": {
				body: map[string]any{
					"type":        "income",
					"amount":      1000.0,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.Id,
					"categoryId":  "invalid-id",
				},
				wantField:   "categoryId",
				wantMessage: "categoryId must be a valid UUID",
				errorsLen:   1,
			},
			"empty body": {
				body:        map[string]any{},
				wantField:   "type",
				wantMessage: "type is required",
				errorsLen:   5,
			},
			"wrong type": {
				body: map[string]any{
					"type":        "outcome",
					"amount":      1000.0,
					"description": "Salary for June",
					"occurredAt":  occurredAt,
					"accountId":   account.Id,
					"categoryId":  category.Id,
				},
				wantField:   "type",
				wantMessage: "type must be either 'income' or 'expense' or 'transfer'",
				errorsLen:   1,
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				req := newJSONRequest(t, http.MethodPost, "/api/transactions", tc.body)
				w := performRequest(t, router, req)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response handlers.ValidationErrorResponse
				parseBody(t, w, &response)
				assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
				assert.Equal(t, "validation failed", response.Message)
				assert.Equal(t, tc.errorsLen, len(response.Errors))
				assert.Equal(t, tc.wantField, response.Errors[0].Field)
				assert.Equal(t, tc.wantMessage, response.Errors[0].Message)
			})
		}
	})
}

func TestUpdateTransaction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCommonTransaction(t, db, "income")
		nextAccount := seedAccount(t, db, "Bank", 2000.0)
		nextCategory := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Groceries",
			Type:  "income",
			Icon:  "shopping-cart",
			Color: "blue",
		})

		params := map[string]any{
			"type":        "income",
			"amount":      500.0,
			"description": "Some expense",
			"occurredAt":  time.Now(),
			"accountId":   nextAccount.Id,
			"categoryId":  nextCategory.Id,
		}
		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.Id,
			params,
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, params["type"], response.Type)
		assert.Equal(t, params["amount"], response.Amount)
		assert.Equal(t, params["description"], response.Description)
		testutil.AssertTimeEqual(
			t,
			params["occurredAt"].(time.Time),
			testutil.ParseDatetime(t, response.OccurredAt),
		)
		assert.Equal(t, params["accountId"], response.AccountId)
		assert.Equal(t, params["categoryId"], response.CategoryId)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCommonTransaction(t, db, "income")

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.Id,
			map[string]any{
				"description": "Updated Transaction",
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, existing.Amount, response.Amount)
		assert.Equal(t, "Updated Transaction", response.Description)
		assert.Equal(t, existing.OccurredAt, response.OccurredAt)
		assert.Equal(t, existing.AccountId, response.AccountId)
		assert.Equal(t, existing.CategoryId, response.CategoryId)
	})

	t.Run("NoFields", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCommonTransaction(t, db, "income")

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.Id,
			map[string]any{},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
		assert.Equal(t, "no fields to update", response.Message)
	})

	t.Run("NonExistAccount", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCommonTransaction(t, db, "income")

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.Id,
			map[string]any{
				"description": "Updated Transaction",
				"accountId":   uuid.NewString(),
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeAccountNotFound, response.Code)
		assert.Equal(t, "account not found", response.Message)
	})

	t.Run("NonExistCategory", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCommonTransaction(t, db, "income")

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.Id,
			map[string]any{
				"description": "Updated Transaction",
				"categoryId":  uuid.NewString(),
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})

	t.Run("CategoryTypeMismatch", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCommonTransaction(t, db, "income")

		nextCategory := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Groceries",
			Type:  "expense",
			Icon:  "shopping-cart",
			Color: "blue",
		})

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/transactions/"+existing.Id,
			map[string]any{
				"description": "Updated Transaction",
				"categoryId":  nextCategory.Id,
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeCategoryTypeMismatch, response.Code)
		assert.Equal(t, "transaction type does not match category type", response.Message)
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/transactions/"+uuid.NewString(),
			map[string]any{
				"type": "income",
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeTransactionNotFound, response.Code)
		assert.Equal(t, "transaction not found", response.Message)
	})
}

func TestDeleteTransaction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCommonTransaction(t, db, "income")

		req := httptest.NewRequest(http.MethodDelete, "/api/transactions/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, 0, w.Body.Len())
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodDelete, "/api/transactions/"+uuid.NewString(), nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeTransactionNotFound, response.Code)
		assert.Equal(t, "transaction not found", response.Message)
	})
}

func TestGetTransaction(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCommonTransaction(t, db, "income")

		req := httptest.NewRequest(http.MethodGet, "/api/transactions/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, existing.Id, response.Id)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, existing.Amount, response.Amount)
		assert.Equal(t, existing.Description, response.Description)
		assert.Equal(t, existing.OccurredAt, response.OccurredAt)
		assert.Equal(t, existing.AccountId, response.AccountId)
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodGet, "/api/transactions/"+uuid.NewString(), nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeTransactionNotFound, response.Code)
		assert.Equal(t, "transaction not found", response.Message)
	})
}

func TestListTransactions(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		seeded1 := seedCommonTransaction(t, db, "income")
		seeded2 := seedCommonTransaction(t, db, "expense")
		seeded3 := seedCommonTransaction(t, db, "income")
		seeded4 := seedCommonTransaction(t, db, "expense")
		seededTransactions := []*storage.Transaction{seeded1, seeded2, seeded3, seeded4}

		req := httptest.NewRequest(http.MethodGet, "/api/transactions", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Transaction
		parseBody(t, w, &response)
		assert.Equal(t, len(seededTransactions), len(response))

		transactionsMap := make(map[string]*storage.Transaction)
		for _, trx := range seededTransactions {
			transactionsMap[trx.Id] = trx
		}
		for _, trx := range response {
			transaction, exists := transactionsMap[trx.Id]
			assert.Equal(t, true, exists)
			assert.Equal(t, *transaction, trx)
		}
	})

	t.Run("SpecificParamsWithoutDateRange", func(t *testing.T) {
		router, db := setupTestEnv(t)

		seeded1 := seedTransactionAt(
			t,
			db,
			"income",
			time.Date(2024, 5, 10, 12, 0, 0, 0, time.UTC),
			800.0,
		)
		seeded2 := seedTransactionAt(
			t,
			db,
			"expense",
			time.Date(2024, 5, 15, 12, 14, 30, 0, time.UTC),
			200.0,
		)
		seeded3 := seedTransactionAt(
			t,
			db,
			"income",
			time.Date(2024, 4, 1, 12, 0, 0, 0, time.UTC),
			10000.0,
		)
		seeded4 := seedTransactionAt(
			t,
			db,
			"expense",
			time.Date(2024, 5, 15, 12, 12, 30, 0, time.UTC),
			3000.0,
		)
		// seededTransactions := []*storage.Transaction{seeded1, seeded2, seeded3, seeded4}

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
				params:   map[string]string{"accountId": seeded1.AccountId},
				expected: []*storage.Transaction{seeded1},
			},
			{
				params:   map[string]string{"categoryId": seeded2.CategoryId},
				expected: []*storage.Transaction{seeded2},
			},
			{
				params:   map[string]string{"type": "income", "accountId": seeded3.AccountId},
				expected: []*storage.Transaction{seeded3},
			},
			{
				params:   map[string]string{"type": "expense", "categoryId": seeded4.CategoryId},
				expected: []*storage.Transaction{seeded4},
			},
			{
				params: map[string]string{
					"type":       "income",
					"accountId":  seeded1.AccountId,
					"categoryId": seeded1.CategoryId,
				},
				expected: []*storage.Transaction{seeded1},
			},
			{
				params: map[string]string{
					"type":       "expense",
					"accountId":  seeded2.AccountId,
					"categoryId": seeded2.CategoryId,
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
		}

		for _, tc := range cases {
			params := url.Values{}
			for key, value := range tc.params {
				params.Add(key, value)
			}
			paramsEncoded := params.Encode()

			t.Run(fmt.Sprintf("Params: %v", params), func(t *testing.T) {
				req := httptest.NewRequest(
					http.MethodGet,
					"/api/transactions"+"?"+paramsEncoded,
					nil,
				)
				w := performRequest(t, router, req)

				assert.Equal(t, http.StatusOK, w.Code)
				var response []storage.Transaction
				parseBody(t, w, &response)
				assert.Equal(t, len(tc.expected), len(response))
				for i, trx := range response {
					assert.Equal(t, tc.expected[i].Id, trx.Id)
				}
			})
		}
	})

	t.Run("DateRange", func(t *testing.T) {
		router, db := setupTestEnv(t)

		beforeRange := seedTransactionAt(t, db, "income",
			time.Date(2024, 5, 15, 12, 0, 0, 0, time.UTC), 100.0)
		onUpperBoundary := seedTransactionAt(
			t,
			db,
			"expense",
			time.Date(2024, 6, 30, 23, 59, 0, 0, time.UTC),
			100.0,
		)
		justAfterUpper := seedTransactionAt(
			t,
			db,
			"income",
			time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			100.0,
		)
		onLowerBoundary := seedTransactionAt(t, db, "income",
			time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), 100.0)
		inMiddle := seedTransactionAt(t, db, "expense",
			time.Date(2024, 6, 15, 14, 30, 0, 0, time.UTC), 100.0)
		afterRange := seedTransactionAt(t, db, "income",
			time.Date(2024, 7, 15, 12, 0, 0, 0, time.UTC), 100.0)

		req := httptest.NewRequest(http.MethodGet,
			"/api/transactions?fromDate=2024-06-01&toDate=2024-06-30", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Transaction
		parseBody(t, w, &response)

		var ids []string
		for _, trx := range response {
			ids = append(ids, trx.Id)
		}

		assert.NotContains(t, ids, beforeRange.Id)
		assert.NotContains(t, ids, justAfterUpper.Id)
		assert.NotContains(t, ids, afterRange.Id)
		assert.Contains(t, ids, onLowerBoundary.Id)
		assert.Contains(t, ids, inMiddle.Id)
		assert.Contains(t, ids, onUpperBoundary.Id)
	})
}
