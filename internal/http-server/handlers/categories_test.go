package handlers_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"expense-tracker-api/internal/http-server/handlers"
	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(t, http.MethodPost, "/api/categories", map[string]any{
			"name":  "Salary",
			"type":  "income",
			"icon":  "dollar-sign",
			"color": "green",
		})
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var response storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, "Salary", response.Name)
		assert.Equal(t, "income", response.Type)
		assert.Equal(t, "dollar-sign", response.Icon)
		assert.Equal(t, "green", response.Color)
		assert.Equal(t, false, response.IsDefault)
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
					"type":  "income",
					"icon":  "dollar-sign",
					"color": "green",
				},
				wantField:   "name",
				wantMessage: "name is required",
				errorsLen:   1,
			},
			"missing type": {
				body: map[string]any{
					"name":  "Salary",
					"icon":  "dollar-sign",
					"color": "green",
				},
				wantField:   "type",
				wantMessage: "type is required",
				errorsLen:   1,
			},
			"empty body": {
				body:        map[string]any{},
				wantField:   "name",
				wantMessage: "name is required",
				errorsLen:   4,
			},
			"empty name": {
				body: map[string]any{
					"name":  "",
					"type":  "income",
					"icon":  "dollar-sign",
					"color": "green",
				},
				wantField:   "name",
				wantMessage: "name is required",
				errorsLen:   1,
			},
			"wrong type": {
				body: map[string]any{
					"name":  "salary",
					"type":  "outcome",
					"icon":  "dollar-sign",
					"color": "green",
				},
				wantField:   "type",
				wantMessage: "type must be either 'income' or 'expense'",
				errorsLen:   1,
			},
		}

		for name, tc := range cases {
			t.Run(name, func(t *testing.T) {
				req := newJSONRequest(t, http.MethodPost, "/api/categories", tc.body)
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

func TestUpdateCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Salary",
			Type:  "income",
			Icon:  "dollar-sign",
			Color: "green",
		})

		params := map[string]any{
			"name":  "Updated Category",
			"type":  "expense",
			"icon":  "cart",
			"color": "red",
		}
		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/categories/"+existing.Id,
			params,
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, params["name"], response.Name)
		assert.Equal(t, params["type"], response.Type)
		assert.Equal(t, params["icon"], response.Icon)
		assert.Equal(t, params["color"], response.Color)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Salary",
			Type:  "income",
			Icon:  "dollar-sign",
			Color: "green",
		})

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/categories/"+existing.Id,
			map[string]any{
				"name": "Updated Category",
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, "Updated Category", response.Name)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, existing.Icon, response.Icon)
		assert.Equal(t, existing.Color, response.Color)
	})

	t.Run("NoFields", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Salary",
			Type:  "income",
			Icon:  "dollar-sign",
			Color: "green",
		})

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/categories/"+existing.Id,
			map[string]any{},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeValidationFailed, response.Code)
		assert.Equal(t, "no fields to update", response.Message)
	})

	t.Run("ForbiddenDefaultCategory", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCategory(t, db, storage.CreateCategoryParams{
			Name:      "Some default",
			Type:      "expense",
			Icon:      "sign-out",
			Color:     "red",
			IsDefault: true,
		})

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/categories/"+existing.Id,
			map[string]any{
				"name": "Updated Salary",
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeForbidden, response.Code)
		assert.Equal(t, "cannot update default category", response.Message)
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := newJSONRequest(
			t,
			http.MethodPatch,
			"/api/categories/"+uuid.NewString(),
			map[string]any{
				"name": "Updated Salary",
			},
		)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})
}

func TestDeleteCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Salary",
			Type:  "income",
			Icon:  "dollar-sign",
			Color: "green",
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/categories/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, 0, w.Body.Len())
	})

	t.Run("ForbiddenDefaultCategory", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCategory(t, db, storage.CreateCategoryParams{
			Name:      "Some default",
			Type:      "expense",
			Icon:      "sign-out",
			Color:     "red",
			IsDefault: true,
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/categories/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeForbidden, response.Code)
		assert.Equal(t, "cannot delete default category", response.Message)
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodDelete, "/api/categories/"+uuid.NewString(), nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})

	t.Run("CategoryWithTransactions", func(t *testing.T) {
		router, db := setupTestEnv(t)

		account := seedAccount(t, db, "Wallet", 1000.0)
		existing := seedCategory(
			t, db, storage.CreateCategoryParams{
				Name:  "Salary",
				Type:  "income",
				Icon:  "dollar-sign",
				Color: "green",
			},
		)
		_ = seedTransaction(t, db, storage.CreateTransactionParams{
			Type:        "income",
			Amount:      100.0,
			Description: "Salary",
			OccurredAt:  time.Now(),
			AccountId:   &account.Id,
			CategoryId:  &existing.Id,
		})

		req := httptest.NewRequest(http.MethodDelete, "/api/categories/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeCategoryInUse, response.Code)
		assert.Equal(t, "category in use", response.Message)
	})
}

func TestGetCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		existing := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Salary",
			Type:  "income",
			Icon:  "dollar-sign",
			Color: "green",
		})

		req := httptest.NewRequest(http.MethodGet, "/api/categories/"+existing.Id, nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, existing.Id, response.Id)
		assert.Equal(t, "Salary", response.Name)
		assert.Equal(t, "income", response.Type)
		assert.Equal(t, "dollar-sign", response.Icon)
		assert.Equal(t, "green", response.Color)
	})

	t.Run("NotFound", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodGet, "/api/categories/"+uuid.NewString(), nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response handlers.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, handlers.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})
}

func TestListCategories(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		router, db := setupTestEnv(t)

		seeded1 := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Salary",
			Type:  "income",
			Icon:  "dollar-sign",
			Color: "green",
		})
		seeded2 := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Investment",
			Type:  "income",
			Icon:  "bank",
			Color: "blue",
		})
		seeded3 := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Shopping",
			Type:  "expense",
			Icon:  "cart",
			Color: "green",
		})
		seeded4 := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Transport",
			Type:  "expense",
			Icon:  "car",
			Color: "red",
		})
		seededCategories := []*storage.Category{seeded1, seeded2, seeded3, seeded4}

		req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, len(seededCategories), len(response))

		categoriesMap := make(map[string]*storage.Category)
		for _, c := range seededCategories {
			categoriesMap[c.Id] = c
		}
		for _, c := range response {
			category, exists := categoriesMap[c.Id]
			assert.Equal(t, true, exists)
			assert.Equal(t, *category, c)
		}
	})

	t.Run("OnlyExpense", func(t *testing.T) {
		router, db := setupTestEnv(t)

		seeded1 := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Salary",
			Type:  "income",
			Icon:  "dollar-sign",
			Color: "green",
		})
		seeded2 := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Investment",
			Type:  "income",
			Icon:  "bank",
			Color: "blue",
		})
		seeded3 := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Shopping",
			Type:  "expense",
			Icon:  "cart",
			Color: "green",
		})
		seeded4 := seedCategory(t, db, storage.CreateCategoryParams{
			Name:  "Transport",
			Type:  "expense",
			Icon:  "car",
			Color: "red",
		})
		seededCategories := []*storage.Category{seeded1, seeded2, seeded3, seeded4}

		req := httptest.NewRequest(http.MethodGet, "/api/categories?type=expense", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, 2, len(response))

		categoriesMap := make(map[string]*storage.Category)
		for _, c := range seededCategories {
			if c.Type == "expense" {
				categoriesMap[c.Id] = c
			}
		}
		for _, c := range response {
			category, exists := categoriesMap[c.Id]
			assert.Equal(t, true, exists)
			assert.Equal(t, *category, c)
		}
	})

	t.Run("NoCategories", func(t *testing.T) {
		router, _ := setupTestEnv(t)

		req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
		w := performRequest(t, router, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, 0, len(response))
	})
}
