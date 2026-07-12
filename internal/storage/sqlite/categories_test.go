package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateCategory(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newFixture(t)
		params := defaultCategoryParams(f.User.ID)
		category, err := f.DB.CreateCategory(params)
		assert.NoError(t, err)
		testutil.AssertValidUUID(t, category.ID)
		assert.Equal(t, params.Name, category.Name)
		assert.Equal(t, params.Icon, category.Icon)
		assert.Equal(t, params.Color, category.Color)
		assert.Equal(t, params.Type, category.Type)

		createdAt := testutil.ParseDatetime(t, category.CreatedAt)
		updatedAt := testutil.ParseDatetime(t, category.UpdatedAt)
		assert.Equal(t, createdAt, updatedAt)
	})

	t.Run("duplicate name for same user returns error", func(t *testing.T) {
		f := newFixture(t)
		params := defaultCategoryParams(f.User.ID)
		_ = seedCategory(t, f.DB, params)
		_, err := f.DB.CreateCategory(params)
		require.ErrorIs(t, err, storage.ErrCategoryAlreadyExists)
	})

	t.Run("same name for different users is allowed", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		_ = seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		_, err := f.DB.CreateCategory(defaultCategoryParams(user2.ID))
		require.NoError(t, err)
	})
}

func TestUpdateCategory(t *testing.T) {
	t.Run("full params updates both params", func(t *testing.T) {
		f := newFixture(t)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		params := storage.UpdateCategoryParams{
			Name:  new("UpdatedCategory"),
			Type:  new("expense"),
			Icon:  new("icon3"),
			Color: new("red"),
		}

		updatedCategory, err := f.DB.UpdateCategory(f.User.ID, category.ID, params)
		require.NoError(t, err)

		require.Equal(t, *params.Name, updatedCategory.Name)
		require.Equal(t, *params.Icon, updatedCategory.Icon)
		require.Equal(t, *params.Color, updatedCategory.Color)
	})

	t.Run("only name change", func(t *testing.T) {
		f := newFixture(t)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		params := storage.UpdateCategoryParams{
			Name: new("UpdatedCategory"),
		}

		updatedCategory, err := f.DB.UpdateCategory(f.User.ID, category.ID, params)
		require.NoError(t, err)
		require.Equal(t, *params.Name, updatedCategory.Name)

		require.Equal(t, category.Type, updatedCategory.Type)
		require.Equal(t, category.Icon, updatedCategory.Icon)
		require.Equal(t, category.Color, updatedCategory.Color)
	})

	t.Run("wrong category id return not found", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.DB.UpdateCategory(f.User.ID, uuid.NewString(), storage.UpdateCategoryParams{})
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("category for another user return not found", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		_, err := f.DB.UpdateCategory(user2.ID, category.ID, storage.UpdateCategoryParams{})
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestDeleteCategory(t *testing.T) {
	t.Run("existing category", func(t *testing.T) {
		f := newFixture(t)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		err := f.DB.DeleteCategory(f.User.ID, category.ID)
		require.NoError(t, err)
	})

	t.Run("non existing category", func(t *testing.T) {
		f := newFixture(t)
		err := f.DB.DeleteCategory(f.User.ID, uuid.NewString())
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("category for another user", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		err := f.DB.DeleteCategory(user2.ID, category.ID)
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("category with transactions", func(t *testing.T) {
		f := newFixture(t)
		account := seedAccount(t, f.DB, f.User.ID, 100000)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		_ = seedCashflowTransaction(
			t,
			f.DB,
			seedCashflowTransactionParams{
				userID:          f.User.ID,
				amount:          20000,
				accountID:       account.ID,
				categoryID:      category.ID,
				transactionType: "income",
			},
		)
		err := f.DB.DeleteCategory(f.User.ID, category.ID)
		require.ErrorIs(t, err, storage.ErrCategoryHasTransactions)
	})

	t.Run("double delete category", func(t *testing.T) {
		f := newFixture(t)
		category := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
		err := f.DB.DeleteCategory(f.User.ID, category.ID)
		require.NoError(t, err)
		err = f.DB.DeleteCategory(f.User.ID, category.ID)
		require.ErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestGetCategory(t *testing.T) {
	f := newFixture(t)
	user2 := seedUser(t, f.DB)
	testCategory := seedCategory(t, f.DB, defaultCategoryParams(f.User.ID))
	testCategory2 := seedCategory(t, f.DB, defaultCategoryParams(user2.ID))

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
			category, err := f.DB.GetCategory(f.User.ID, tc.id)

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
		f := newFixture(t)
		categories, err := f.DB.GetCategories(f.User.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)
		assert.NotEmpty(t, categories)
	})

	t.Run("no other user categories for another user", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		userCategories, err := f.DB.GetCategories(f.User.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)
		user2Categories, err := f.DB.GetCategories(user2.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)
		assert.NotContains(t, categoryNames(userCategories), categoryNames(user2Categories))
	})

	t.Run("existing categories in database with no params", func(t *testing.T) {
		f := newFixture(t)
		createdCategories := seedCategories(t, f.DB, f.User.ID, 4)
		categories, err := f.DB.GetCategories(f.User.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)

		for _, c := range createdCategories {
			require.Contains(t, categoryNames(categories), c.Name)
		}
	})

	t.Run("existing categories in database with type param = income", func(t *testing.T) {
		f := newFixture(t)
		createdCategories := seedCategories(t, f.DB, f.User.ID, 4)
		categories, err := f.DB.GetCategories(
			f.User.ID,
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
