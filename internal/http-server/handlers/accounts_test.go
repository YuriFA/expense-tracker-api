package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedAccount(
	t *testing.T,
	db *sqlite.Storage,
	name string,
	openingBalance float64,
) *storage.Account {
	account, err := db.CreateAccount(name, openingBalance)
	require.NoError(t, err)
	return account
}

func TestCreateAccount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodPost, "/api/accounts", map[string]any{
			"name":           "Wallet",
			"openingBalance": 1000.0,
		})
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Wallet", response.Name)
		assert.Equal(t, 1000.0, response.OpeningBalance)
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
					"openingBalance": 1000.0,
				},
				wantField:   "name",
				wantMessage: "name is required",
				errorsLen:   1,
			},
			"missing openingBalance": {
				body: map[string]any{
					"name": "Wallet",
				},
				wantField:   "openingBalance",
				wantMessage: "openingBalance is required",
				errorsLen:   1,
			},
			"empty body": {
				body:        map[string]any{},
				wantField:   "name",
				wantMessage: "name is required",
				errorsLen:   2,
			},
			"empty name": {
				body: map[string]any{
					"name":           "",
					"openingBalance": 1000.0,
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
				assert.Equal(t, tc.errorsLen, len(response.Errors))
				assert.Equal(t, tc.wantField, response.Errors[0].Field)
				assert.Equal(t, tc.wantMessage, response.Errors[0].Message)
			})
		}
	})
}

func TestUpdateAccount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 1000.0)

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/accounts/"+existing.Id,
			map[string]any{
				"name":             "Updated Wallet",
				"manualAdjustment": 100.0,
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Updated Wallet", response.Name)
		assert.Equal(t, 100.0, response.ManualAdjustment)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 1000.0)

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
	})

	t.Run("ShortName", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 1000.0)

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

		existing := seedAccount(t, db, "Wallet", 1000.0)

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

		existing := seedAccount(t, db, "Wallet", 1000.0)

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
}

func TestGetAccount(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedAccount(t, db, "Wallet", 1000.0)

		req := httptest.NewRequest(http.MethodGet, "/api/accounts/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Account
		parseBody(t, w, &response)
		assert.Equal(t, "Wallet", response.Name)
		assert.Equal(t, 1000.0, response.OpeningBalance)
		assert.Equal(t, 0.0, response.ManualAdjustment)
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

func TestListAccounts(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		seeded1 := seedAccount(t, db, "Wallet", 1000.0)
		seeded2 := seedAccount(t, db, "Bank", 5000.0)
		seeded3 := seedAccount(t, db, "Cash", 200.0)
		seeded4 := seedAccount(t, db, "Credit Card", 0.0)
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
