package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
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
				testutil.AssertError(t, err)
				return
			}

			testutil.AssertNoError(t, err)

			testutil.AssertEqual(t, category.Name, tc.params.Name)
			testutil.AssertEqual(t, category.Icon, tc.params.Icon)
			testutil.AssertEqual(t, category.Color, tc.params.Color)
			testutil.AssertEqual(t, category.IsDefault, tc.params.IsDefault)
			testutil.AssertValidUUID(t, category.Id)

			createdAt := testutil.ParseDatetime(t, category.CreatedAt)
			updatedAt := testutil.ParseDatetime(t, category.UpdatedAt)
			testutil.AssertEqual(t, createdAt, updatedAt)
		})
	}
}

func TestUpdateCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("full params updates both params", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: false,
		})
		testutil.AssertNoError(t, err)
		params := storage.UpdateCategoryParams{
			Name:  new("UpdatedCategory"),
			Type:  new("expense"),
			Icon:  new("icon3"),
			Color: new("red"),
		}

		updatedCategory, err := db.UpdateCategory(category.Id, params)
		testutil.AssertNoError(t, err)

		testutil.AssertEqual(t, updatedCategory.Name, *params.Name)
		testutil.AssertEqual(t, updatedCategory.Icon, *params.Icon)
		testutil.AssertEqual(t, updatedCategory.Color, *params.Color)
	})

	t.Run("only name change", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: false,
		})
		testutil.AssertNoError(t, err)
		params := storage.UpdateCategoryParams{
			Name: new("UpdatedCategory"),
		}

		updatedCategory, err := db.UpdateCategory(category.Id, params)
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, updatedCategory.Name, *params.Name)

		testutil.AssertEqual(t, updatedCategory.Type, category.Type)
		testutil.AssertEqual(t, updatedCategory.Icon, category.Icon)
		testutil.AssertEqual(t, updatedCategory.Color, category.Color)
	})

	t.Run("empty params still bumps updated_at", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: true,
		})
		testutil.AssertNoError(t, err)

		updatedCategory, err := db.UpdateCategory(category.Id, storage.UpdateCategoryParams{})
		testutil.AssertNoError(t, err)

		testutil.AssertEqual(t, updatedCategory.Name, category.Name)
		testutil.AssertEqual(t, updatedCategory.Type, category.Type)
		testutil.AssertEqual(t, updatedCategory.Icon, category.Icon)
		testutil.AssertEqual(t, updatedCategory.Color, category.Color)
	})

	t.Run("wrong category id return not found", func(t *testing.T) {
		_, err := db.UpdateCategory(uuid.NewString(), storage.UpdateCategoryParams{})
		testutil.AssertErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestDeleteCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	t.Run("existing category", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: true,
		})
		testutil.AssertNoError(t, err)

		err = db.DeleteCategory(category.Id)
		testutil.AssertNoError(t, err)
	})

	t.Run("non existing category", func(t *testing.T) {
		err := db.DeleteCategory(uuid.NewString())
		testutil.AssertErrorIs(t, err, storage.ErrCategoryNotFound)
	})

	t.Run("double delete category", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: true,
		})
		testutil.AssertNoError(t, err)
		err = db.DeleteCategory(category.Id)
		testutil.AssertNoError(t, err)
		err = db.DeleteCategory(category.Id)
		testutil.AssertErrorIs(t, err, storage.ErrCategoryNotFound)
	})
}

func TestGetCategory(t *testing.T) {
	db := sqlite.NewTestDB(t)

	testCategory, err := db.CreateCategory(storage.CreateCategoryParams{
		Name:      "Category1",
		Type:      "income",
		Icon:      "icon2",
		Color:     "blue",
		IsDefault: true,
	})
	testutil.AssertNoError(t, err)

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
				testutil.AssertErrorIs(t, err, tc.expectedErr)
				return
			}

			testutil.AssertNoError(t, err)
			testutil.AssertEqual(t, category.Id, tc.id)
		})
	}
}

func createTestCategories(db *sqlite.Storage) ([]storage.Category, error) {
	testCategories := []storage.CreateCategoryParams{
		{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: true,
		},
		{
			Name:      "Category2",
			Type:      "expense",
			Icon:      "icon3",
			Color:     "red",
			IsDefault: false,
		},
		{
			Name:      "Category3",
			Type:      "expense",
			Icon:      "icon4",
			Color:     "green",
			IsDefault: true,
		},
		{
			Name:      "Category4",
			Type:      "income",
			Icon:      "icon5",
			Color:     "black",
			IsDefault: true,
		},
	}

	result := []storage.Category{}

	for _, params := range testCategories {
		category, err := db.CreateCategory(params)
		if err != nil {
			return nil, err
		}

		result = append(result, *category)
	}

	return result, nil
}

func TestGetCategories(t *testing.T) {
	t.Run("empty categories in database", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		categories, err := db.GetCategories(storage.GetCategoriesParams{})
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(categories), 0)
	})

	t.Run("existing categories in database with no params", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdCategories, err := createTestCategories(db)
		testutil.AssertNoError(t, err)

		categories, err := db.GetCategories(storage.GetCategoriesParams{})
		testutil.AssertNoError(t, err)
		testutil.AssertEqual(t, len(createdCategories), len(categories))
	})

	t.Run("existing categories in database with type param = income", func(t *testing.T) {
		db := sqlite.NewTestDB(t)

		createdCategories, err := createTestCategories(db)
		testutil.AssertNoError(t, err)

		categories, err := db.GetCategories(
			storage.GetCategoriesParams{Type: new("income")},
		)
		testutil.AssertNoError(t, err)

		incomeCategories := testutil.Filter(createdCategories, func(c storage.Category) bool {
			return c.Type == "income"
		})
		testutil.AssertEqual(t, len(incomeCategories), len(categories))
	})
}
