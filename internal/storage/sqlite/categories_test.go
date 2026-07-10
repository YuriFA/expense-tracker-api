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
	t.Run("success", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			UserID: user.ID,
			Name:   "Category1",
			Type:   "income",
			Icon:   "icon1",
			Color:  "blue",
		})

		assert.NoError(t, err)

		assert.Equal(t, "Category1", category.Name)
		assert.Equal(t, "icon1", category.Icon)
		assert.Equal(t, "blue", category.Color)
		testutil.AssertValidUUID(t, category.ID)

		createdAt := testutil.ParseDatetime(t, category.CreatedAt)
		updatedAt := testutil.ParseDatetime(t, category.UpdatedAt)
		assert.Equal(t, createdAt, updatedAt)
	})

	t.Run("duplicate name for same user returns error", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		_ = seedCategory(t, db, "Category1", user.ID, "income")
		_, err := db.CreateCategory(storage.CreateCategoryParams{
			UserID: user.ID,
			Name:   "Category1",
			Type:   "income",
			Icon:   "icon1",
			Color:  "blue",
		})
		require.ErrorIs(t, err, storage.ErrCategoryAlreadyExists)
	})

	t.Run("same name for different users is allowed", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user1 := seedUser(t, db, "test1@example.com")
		user2 := seedUser(t, db, "test2@example.com")
		_ = seedCategory(t, db, "Category1", user1.ID, "income")
		_, err := db.CreateCategory(storage.CreateCategoryParams{
			UserID: user2.ID,
			Name:   "Category1",
			Type:   "income",
			Icon:   "icon1",
			Color:  "blue",
		})
		require.NoError(t, err)
	})
}

func TestUpdateCategory(t *testing.T) {
	t.Run("full params updates both params", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.ID, "income")
		params := storage.UpdateCategoryParams{
			Name:  new("UpdatedCategory"),
			Type:  new("expense"),
			Icon:  new("icon3"),
			Color: new("red"),
		}

		updatedCategory, err := db.UpdateCategory(user.ID, category.ID, params)
		require.NoError(t, err)

		require.Equal(t, *params.Name, updatedCategory.Name)
		require.Equal(t, *params.Icon, updatedCategory.Icon)
		require.Equal(t, *params.Color, updatedCategory.Color)
	})

	t.Run("only name change", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.ID, "income")
		params := storage.UpdateCategoryParams{
			Name: new("UpdatedCategory"),
		}

		updatedCategory, err := db.UpdateCategory(user.ID, category.ID, params)
		require.NoError(t, err)
		require.Equal(t, *params.Name, updatedCategory.Name)

		require.Equal(t, category.Type, updatedCategory.Type)
		require.Equal(t, category.Icon, updatedCategory.Icon)
		require.Equal(t, category.Color, updatedCategory.Color)
	})

	t.Run("wrong category id return not found", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		_, err := db.UpdateCategory(user.ID, uuid.NewString(), storage.UpdateCategoryParams{})
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("category for another user return not found", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		user2 := seedUser(t, db, "test2@example.com")
		category := seedCategory(t, db, "salary", user.ID, "income")
		_, err := db.UpdateCategory(user2.ID, category.ID, storage.UpdateCategoryParams{})
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestDeleteCategory(t *testing.T) {
	t.Run("existing category", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.ID, "income")
		err := db.DeleteCategory(user.ID, category.ID)
		require.NoError(t, err)
	})

	t.Run("non existing category", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		err := db.DeleteCategory(user.ID, uuid.NewString())
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("category for another user", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		user2 := seedUser(t, db, "test2@example.com")
		category := seedCategory(t, db, "salary", user.ID, "income")
		err := db.DeleteCategory(user2.ID, category.ID)
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("category with transactions", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		account := seedAccount(t, db, user.ID, 100000)
		category := seedCategory(t, db, "salary", user.ID, "income")
		_ = seedCashflowTransaction(
			t,
			db,
			seedCashflowTransactionParams{
				userID:          user.ID,
				amount:          20000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: "income",
			},
		)
		err := db.DeleteCategory(user.ID, category.ID)
		require.ErrorIs(t, err, storage.ErrCategoryHasTransactions)
	})

	t.Run("double delete category", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		category := seedCategory(t, db, "salary", user.ID, "income")
		err := db.DeleteCategory(user.ID, category.ID)
		require.NoError(t, err)
		err = db.DeleteCategory(user.ID, category.ID)
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestGetCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	user := seedUser(t, db, "test@example.com")
	user2 := seedUser(t, db, "test2@example.com")
	testCategory := seedCategory(t, db, "salary", user.ID, "income")
	testCategory2 := seedCategory(t, db, "salary", user2.ID, "income")

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
			id:        testCategory.ID,
			respError: false,
		},
		"non existing category id for another user": {
			id:          testCategory2.ID,
			respError:   true,
			expectedErr: storage.ErrCategoryNotFound,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			category, err := db.GetCategory(user.ID, tc.id)

			if tc.respError {
				require.ErrorIs(t, err, tc.expectedErr)
				return
			}

			require.NoError(t, err)
			require.Equal(t, tc.id, category.ID)
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
	t.Run("seeded categories in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		categories, err := db.GetCategories(user.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)
		assert.NotEmpty(t, categories)
	})

	t.Run("no other user categories for another user", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		user2 := seedUser(t, db, "test2@example.com")
		userCategories, err := db.GetCategories(user.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)
		user2Categories, err := db.GetCategories(user2.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)
		assert.NotContains(t, categoryNames(userCategories), categoryNames(user2Categories))
	})

	t.Run("existing categories in database with no params", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user := seedUser(t, db, "test@example.com")
		createdCategories := seedCategories(t, db, user.ID, 4)
		categories, err := db.GetCategories(user.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)

		for _, c := range createdCategories {
			require.Contains(t, categoryNames(categories), c.Name)
		}
	})

	t.Run("existing categories in database with type param = income", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		user := seedUser(t, db, "test@example.com")
		createdCategories := seedCategories(t, db, user.ID, 4)

		categories, err := db.GetCategories(
			user.ID,
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
