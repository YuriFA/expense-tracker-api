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

func TestCreateCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	cases := map[string]struct {
		params    storage.CreateCategoryParams
		respError bool
	}{
		"non default category": {
			params: storage.CreateCategoryParams{
				Name:      "Category 1",
				Type:      "expense",
				Icon:      "icon1",
				Color:     "red",
				IsDefault: false,
			},
			respError: false,
		},
		"default category": {
			params: storage.CreateCategoryParams{
				Name:      "Category 2",
				Type:      "income",
				Icon:      "icon2",
				Color:     "blue",
				IsDefault: true,
			},
			respError: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			category, err := db.CreateCategory(tc.params)
			if tc.respError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			assert.Equal(t, tc.params.Name, category.Name)
			assert.Equal(t, tc.params.Icon, category.Icon)
			assert.Equal(t, tc.params.Color, category.Color)
			assert.Equal(t, tc.params.IsDefault, category.IsDefault)
			testutil.AssertValidUUID(t, category.Id)

			createdAt := testutil.ParseDatetime(t, category.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, category.UpdatedAt)
			assert.Equal(t, createdAt, updatedAt)
		})
	}
}

func TestUpdateCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("full params updates both params", func(t *testing.T) {
		category := seedCategory(t, db, "income")
		params := storage.UpdateCategoryParams{
			Name:  new("UpdatedCategory"),
			Type:  new("expense"),
			Icon:  new("icon3"),
			Color: new("red"),
		}

		updatedCategory, err := db.UpdateCategory(category.Id, params)
		require.NoError(t, err)

		require.Equal(t, *params.Name, updatedCategory.Name)
		require.Equal(t, *params.Icon, updatedCategory.Icon)
		require.Equal(t, *params.Color, updatedCategory.Color)
	})

	t.Run("only name change", func(t *testing.T) {
		category := seedCategory(t, db, "income")
		params := storage.UpdateCategoryParams{
			Name: new("UpdatedCategory"),
		}

		updatedCategory, err := db.UpdateCategory(category.Id, params)
		require.NoError(t, err)
		require.Equal(t, *params.Name, updatedCategory.Name)

		require.Equal(t, category.Type, updatedCategory.Type)
		require.Equal(t, category.Icon, updatedCategory.Icon)
		require.Equal(t, category.Color, updatedCategory.Color)
	})

	t.Run("wrong category id return not found", func(t *testing.T) {
		_, err := db.UpdateCategory(uuid.NewString(), storage.UpdateCategoryParams{})
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestDeleteCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("existing category", func(t *testing.T) {
		category := seedCategory(t, db, "income")
		err := db.DeleteCategory(category.Id)
		require.NoError(t, err)
	})

	t.Run("non existing category", func(t *testing.T) {
		err := db.DeleteCategory(uuid.NewString())
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("category with transactions", func(t *testing.T) {
		account := seedAccount(t, db, 1000.0)
		category := seedCategory(t, db, "income")
		_ = seedTransaction(
			t,
			db,
			seedTransactionParams{
				amount:          200.0,
				accountId:       account.Id,
				categoryId:      category.Id,
				transactionType: "income",
			},
		)
		err := db.DeleteCategory(category.Id)
		require.ErrorIs(t, err, storage.ErrCategoryHasTransactions)
	})

	t.Run("double delete category", func(t *testing.T) {
		category := seedCategory(t, db, "income")
		err := db.DeleteCategory(category.Id)
		require.NoError(t, err)
		err = db.DeleteCategory(category.Id)
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestGetCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	testCategory := seedCategory(t, db, "income")

	cases := map[string]struct {
		id          string
		respError   bool
		expectedErr error
	}{
		"random non exist uuid": {
			id:          uuid.NewString(),
			respError:   true,
			expectedErr: storage.ErrCategoryNotFound,
		},
		"non uuid string": {
			id:          "some id",
			respError:   true,
			expectedErr: storage.ErrCategoryNotFound,
		},
		"existing category id": {
			id:        testCategory.Id,
			respError: false,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			category, err := db.GetCategory(tc.id)

			if tc.respError {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.id, category.Id)
		})
	}
}

func TestGetCategories(t *testing.T) {
	t.Run("empty categories in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		categories, err := db.GetCategories(storage.GetCategoriesParams{})
		require.NoError(t, err)
		assert.Empty(t, categories)
	})

	t.Run("existing categories in database with no params", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		createdCategories := seedCategories(t, db, 4)
		categories, err := db.GetCategories(storage.GetCategoriesParams{})
		require.NoError(t, err)
		assert.Equal(t, len(createdCategories), len(categories))
	})

	t.Run("existing categories in database with type param = income", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdCategories := seedCategories(t, db, 4)

		categories, err := db.GetCategories(
			storage.GetCategoriesParams{Type: new("income")},
		)
		require.NoError(t, err)

		incomeCategories := testutil.Filter(createdCategories, func(c *storage.Category) bool {
			return c.Type == "income"
		})
		require.Equal(t, len(incomeCategories), len(categories))
	})
}
