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

	t.Run("success", func(t *testing.T) {
		user := seedUser(t, db, "test@example.com")
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			UserId: user.Id,
			Name:   "Category1",
			Type:   "income",
			Icon:   "icon1",
			Color:  "blue",
		})

		assert.NoError(t, err)

		assert.Equal(t, "Category1", category.Name)
		assert.Equal(t, "icon1", category.Icon)
		assert.Equal(t, "blue", category.Color)
		testutil.AssertValidUUID(t, category.Id)

		createdAt := testutil.ParseDatetime(t, category.CreatedAt)
		updatedAt := testutil.ParseDatetime(t, category.UpdatedAt)
		assert.Equal(t, createdAt, updatedAt)
	})
}

func TestUpdateCategory(t *testing.T) {
	t.Run("full params updates both params", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.Id, "income")
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
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.Id, "income")
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
		db := sqlite.NewTestDB(t)
		_, err := db.UpdateCategory(uuid.NewString(), storage.UpdateCategoryParams{})
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestDeleteCategory(t *testing.T) {
	t.Run("existing category", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.Id, "income")
		err := db.DeleteCategory(category.Id)
		require.NoError(t, err)
	})

	t.Run("non existing category", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		err := db.DeleteCategory(uuid.NewString())
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("category with transactions", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		account := seedAccount(t, db, 100000)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.Id, "income")
		_ = seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				amount:          20000,
				accountId:       account.Id,
				categoryId:      category.Id,
				transactionType: "income",
			},
		)
		err := db.DeleteCategory(category.Id)
		require.ErrorIs(t, err, storage.ErrCategoryHasTransactions)
	})

	t.Run("double delete category", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.Id, "income")
		err := db.DeleteCategory(category.Id)
		require.NoError(t, err)
		err = db.DeleteCategory(category.Id)
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestGetCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	user := seedUser(t, db, "test@example.com")
	testCategory := seedCategory(t, db, "salary", user.Id, "income")

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

func categoryNames(categories []storage.Category) []string {
	names := make([]string, len(categories))
	for i, c := range categories {
		names[i] = c.Name
	}
	return names
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
		user := seedUser(t, db, "test@example.com")
		createdCategories := seedCategories(t, db, user.Id, 4)
		categories, err := db.GetCategories(storage.GetCategoriesParams{})
		require.NoError(t, err)

		for _, c := range createdCategories {
			require.Contains(t, categoryNames(categories), c.Name)
		}
	})

	t.Run("existing categories in database with type param = income", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		user := seedUser(t, db, "test@example.com")
		createdCategories := seedCategories(t, db, user.Id, 4)

		categories, err := db.GetCategories(
			storage.GetCategoriesParams{Type: new("income")},
		)
		require.NoError(t, err)
		for _, c := range categories {
			require.Equal(t, "income", c.Type)
		}

		incomeFromTest := testutil.Filter(createdCategories, func(c *storage.Category) bool {
			return c.Type == "income"
		})
		for _, c := range incomeFromTest {
			require.Contains(t, categoryNames(categories), c.Name)
		}
	})
}
