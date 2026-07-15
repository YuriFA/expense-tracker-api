package handlers_test

import (
	"net/http"
	"testing"

	"expense-tracker-api/internal/http-server/httperr"
	"expense-tracker-api/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodPost, "/api/categories", map[string]any{
			"name":  "CustomSalary",
			"type":  "income",
			"icon":  "dollar-sign",
			"color": "green",
		})

		assert.Equal(t, http.StatusCreated, w.Code)
		var response storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, "CustomSalary", response.Name)
		assert.Equal(t, "income", response.Type)
		assert.Equal(t, "dollar-sign", response.Icon)
		assert.Equal(t, "green", response.Color)
	})

	t.Run("ValidationFail", func(t *testing.T) {
		f := newAuthFixture(t)

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
				w := f.do(t, http.MethodPost, "/api/categories", tc.body)

				assert.Equal(t, http.StatusBadRequest, w.Code)
				var response httperr.ValidationErrorResponse
				parseBody(t, w, &response)
				assert.Equal(t, httperr.ErrCodeValidationFailed, response.Code)
				assert.Equal(t, "validation failed", response.Message)
				require.Equal(t, tc.errorsLen, len(response.Errors))
				assert.Equal(t, tc.wantField, response.Errors[0].Field)
				assert.Equal(t, tc.wantMessage, response.Errors[0].Message)
			})
		}
	})

	t.Run("Duplicate category by name", func(t *testing.T) {
		f := newAuthFixture(t)

		seedCategory(t, f.DB, storage.CreateCategoryParams{
			UserID: f.User.ID,
			Name:   "CustomSalary",
			Type:   "income",
			Icon:   "dollar-sign",
			Color:  "green",
		})
		w := f.do(t, http.MethodPost, "/api/categories", map[string]any{
			"name":  "CustomSalary",
			"type":  "income",
			"icon":  "dollar-sign",
			"color": "green",
		})

		assert.Equal(t, http.StatusConflict, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		require.Equal(t, httperr.ErrCodeCategoryAlreadyExists, response.Code)
	})
}

func TestUpdateCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedDefaultExpenseCategory(t, f.DB, f.User.ID)

		params := map[string]any{
			"name":  "Updated Category",
			"type":  "expense",
			"icon":  "cart",
			"color": "red",
		}
		w := f.do(t, http.MethodPatch, "/api/categories/"+existing.ID, params)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, params["name"], response.Name)
		assert.Equal(t, params["type"], response.Type)
		assert.Equal(t, params["icon"], response.Icon)
		assert.Equal(t, params["color"], response.Color)
	})

	t.Run("PartialUpdate", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedDefaultIncomeCategory(t, f.DB, f.User.ID)

		w := f.do(t, http.MethodPatch, "/api/categories/"+existing.ID, map[string]any{
			"name": "Updated Category",
		})

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, "Updated Category", response.Name)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, existing.Icon, response.Icon)
		assert.Equal(t, existing.Color, response.Color)
	})

	t.Run("NoFields", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedDefaultIncomeCategory(t, f.DB, f.User.ID)

		w := f.do(t, http.MethodPatch, "/api/categories/"+existing.ID, map[string]any{})

		assert.Equal(t, http.StatusBadRequest, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeValidationFailed, response.Code)
		assert.Equal(t, "no fields to update", response.Message)
	})

	t.Run("New name is already existing", func(t *testing.T) {
		f := newAuthFixture(t)
		category := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		category2 := seedCategory(t, f.DB, storage.CreateCategoryParams{
			UserID: f.User.ID,
			Name:   "CustomSalary",
			Type:   "income",
			Icon:   "dollar-sign",
			Color:  "green",
		})

		w := f.do(t, http.MethodPatch, "/api/categories/"+category.ID, map[string]any{
			"name": category2.Name,
		})

		assert.Equal(t, http.StatusConflict, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeCategoryAlreadyExists, response.Code)
	})

	t.Run("NotFound", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodPatch, "/api/categories/"+uuid.NewString(), map[string]any{
			"name": "Updated Salary",
		})

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})
}

func TestDeleteCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedDefaultIncomeCategory(t, f.DB, f.User.ID)

		w := f.do(t, http.MethodDelete, "/api/categories/"+existing.ID, nil)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, 0, w.Body.Len())
	})

	t.Run("NotFound", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodDelete, "/api/categories/"+uuid.NewString(), nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})

	t.Run("CategoryWithTransactions", func(t *testing.T) {
		f := newAuthFixture(t)
		account := seedAccount(t, f.DB, defaultAccountParams(f.User.ID))
		existing := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		transactionParams := defaultCashflowTransactionParams(f.User.ID, account.ID, existing.ID)
		transactionParams.Type = "income"
		_ = seedTransaction(t, f.DB, transactionParams)

		w := f.do(t, http.MethodDelete, "/api/categories/"+existing.ID, nil)

		assert.Equal(t, http.StatusConflict, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeCategoryInUse, response.Code)
		assert.Equal(t, "category in use", response.Message)
	})

	t.Run("Stranger category NotFound", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		f2 := newAuthFixture(t)

		w := f2.do(t, http.MethodDelete, "/api/categories/"+existing.ID, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})
}

func TestGetCategory(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedDefaultIncomeCategory(t, f.DB, f.User.ID)

		w := f.do(t, http.MethodGet, "/api/categories/"+existing.ID, nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response storage.Category
		parseBody(t, w, &response)
		assert.Equal(t, existing.ID, response.ID)
		assert.Equal(t, existing.Name, response.Name)
		assert.Equal(t, existing.Type, response.Type)
		assert.Equal(t, existing.Icon, response.Icon)
		assert.Equal(t, existing.Color, response.Color)
	})

	t.Run("NotFound", func(t *testing.T) {
		f := newAuthFixture(t)

		w := f.do(t, http.MethodGet, "/api/categories/"+uuid.NewString(), nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})

	t.Run("Stranger category NotFound", func(t *testing.T) {
		f := newAuthFixture(t)
		existing := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		f2 := newAuthFixture(t)

		w := f2.do(t, http.MethodGet, "/api/categories/"+existing.ID, nil)

		assert.Equal(t, http.StatusNotFound, w.Code)
		var response httperr.ErrorResponse
		parseBody(t, w, &response)
		assert.Equal(t, httperr.ErrCodeCategoryNotFound, response.Code)
		assert.Equal(t, "category not found", response.Message)
	})
}

func TestListCategories(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newAuthFixture(t)
		seeded1 := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		seeded2 := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		seeded3 := seedDefaultExpenseCategory(t, f.DB, f.User.ID)
		seeded4 := seedDefaultExpenseCategory(t, f.DB, f.User.ID)
		seededCategories := []*storage.Category{seeded1, seeded2, seeded3, seeded4}

		w := f.do(t, http.MethodGet, "/api/categories", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Category
		parseBody(t, w, &response)

		categoriesMap := make(map[string]*storage.Category)
		for _, c := range response {
			categoriesMap[c.ID] = &c
		}
		for _, c := range seededCategories {
			category, exists := categoriesMap[c.ID]
			assert.Equal(t, true, exists)
			assert.Equal(t, category, c)
		}
	})

	t.Run("OnlyExpense", func(t *testing.T) {
		f := newAuthFixture(t)

		seeded1 := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		seeded2 := seedDefaultIncomeCategory(t, f.DB, f.User.ID)
		seeded3 := seedDefaultExpenseCategory(t, f.DB, f.User.ID)
		seeded4 := seedDefaultExpenseCategory(t, f.DB, f.User.ID)
		seededCategories := []*storage.Category{seeded1, seeded2, seeded3, seeded4}

		w := f.do(t, http.MethodGet, "/api/categories?type=expense", nil)

		assert.Equal(t, http.StatusOK, w.Code)
		var response []storage.Category
		parseBody(t, w, &response)

		categoriesMap := make(map[string]*storage.Category)
		for _, c := range response {
			categoriesMap[c.ID] = &c
		}
		for _, c := range seededCategories {
			if c.Type == "expense" {
				category, exists := categoriesMap[c.ID]
				assert.Equal(t, true, exists)
				assert.Equal(t, category, c)
			}
		}
	})
}
