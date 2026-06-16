package sqlite_test

import (
	"errors"
	"testing"
	"time"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"
	"expense-tracker-api/internal/testutil"

	"github.com/google/uuid"
)

func TestCreateCategory(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

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
			if err != nil && !tc.respError {
				t.Fatalf("unexpected error: %v", err)
			}
			if err == nil && tc.respError {
				t.Fatalf("expected error, got nil")
			}

			if category.Name != tc.params.Name || category.Type != tc.params.Type ||
				category.Icon != tc.params.Icon ||
				category.Color != tc.params.Color ||
				category.IsDefault != tc.params.IsDefault {
				t.Errorf("unexpected category data: got %v", category)
			}

			if err := uuid.Validate(category.Id); err != nil {
				t.Errorf("expected category ID to be set, got empty string")
			}

			createdAt, err := time.Parse(time.RFC3339, category.CreatedAt)
			if err != nil {
				t.Errorf("expected created_at to be a valid timestamp, got: %v", category.CreatedAt)
			}

			updatedAt, err := time.Parse(time.RFC3339, category.UpdatedAt)
			if err != nil {
				t.Errorf("expected updated_at to be a valid timestamp, got: %v", category.UpdatedAt)
			}

			if !createdAt.Equal(updatedAt) {
				t.Errorf(
					"expected created_at === updated_at, got created_at: %v, updated_at: %v",
					category.CreatedAt,
					category.UpdatedAt,
				)
			}
		})
	}
}

func TestUpdateCategory(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Run("full params updates both params", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: false,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		time.Sleep(1100 * time.Millisecond) // Ensure updated_at will be different from created_at
		params := storage.UpdateCategoryParams{
			Name:  testutil.Ptr("UpdatedCategory"),
			Type:  testutil.Ptr("expense"),
			Icon:  testutil.Ptr("icon3"),
			Color: testutil.Ptr("red"),
		}

		updatedCategory, err := db.UpdateCategory(category.Id, params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if updatedCategory.Name != *params.Name || updatedCategory.Type != *params.Type ||
			updatedCategory.Icon != *params.Icon ||
			updatedCategory.Color != *params.Color {
			t.Errorf("unexpected category data: got %v", updatedCategory)
		}

		categoryUpdatedAt, err := time.Parse(time.RFC3339, category.UpdatedAt)
		if err != nil {
			t.Errorf(
				"expected prev updated_at to be a valid timestamp, got: %v",
				category.UpdatedAt,
			)
		}
		updatedAt, err := time.Parse(time.RFC3339, updatedCategory.UpdatedAt)
		if err != nil {
			t.Errorf(
				"expected next updated_at to be a valid timestamp, got: %v",
				updatedCategory.UpdatedAt,
			)
		}

		if categoryUpdatedAt.Equal(updatedAt) {
			t.Errorf(
				"expected updated_at to be different from test category updated_at, got updated_at: %v",
				updatedCategory.UpdatedAt,
			)
		}
	})

	t.Run("only name change", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: false,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		params := storage.UpdateCategoryParams{
			Name: testutil.Ptr("UpdatedCategory"),
		}

		updatedCategory, err := db.UpdateCategory(category.Id, params)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if updatedCategory.Name != *params.Name {
			t.Errorf("unexpected category data: got %v", updatedCategory)
		}

		if category.Type != updatedCategory.Type || category.Icon != updatedCategory.Icon ||
			category.Color != updatedCategory.Color {
			t.Errorf(
				"unexpected category data, only name must be changed: prev %v, got %v",
				category,
				updatedCategory,
			)
		}
	})

	t.Run("empty params still bumps updated_at", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		time.Sleep(1100 * time.Millisecond) // Ensure updated_at will be different from created_at

		updatedCategory, err := db.UpdateCategory(category.Id, storage.UpdateCategoryParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if category.Name != updatedCategory.Name || category.Type != updatedCategory.Type ||
			category.Icon != updatedCategory.Icon ||
			category.Color != updatedCategory.Color {
			t.Errorf(
				"unexpected category data, only name must be changed: prev %v, got %v",
				category,
				updatedCategory,
			)
		}

		categoryUpdatedAt, err := time.Parse(time.RFC3339, category.UpdatedAt)
		if err != nil {
			t.Errorf(
				"expected prev updated_at to be a valid timestamp, got: %v",
				category.UpdatedAt,
			)
		}
		updatedAt, err := time.Parse(time.RFC3339, updatedCategory.UpdatedAt)
		if err != nil {
			t.Errorf(
				"expected next updated_at to be a valid timestamp, got: %v",
				updatedCategory.UpdatedAt,
			)
		}

		if categoryUpdatedAt.Equal(updatedAt) {
			t.Errorf(
				"expected updated_at to be different from test category updated_at, got updated_at: %v",
				updatedCategory.UpdatedAt,
			)
		}
	})

	t.Run("wrong category id return not found", func(t *testing.T) {
		_, err := db.UpdateCategory(uuid.NewString(), storage.UpdateCategoryParams{})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, storage.ErrCategoryNotFound) {
			t.Fatalf("error mismatch: want `%v`, got: `%v`", storage.ErrCategoryNotFound, err)
		}
	})
}

func TestDeleteCategory(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	t.Run("existing category", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		err = db.DeleteCategory(category.Id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("non existing category", func(t *testing.T) {
		err = db.DeleteCategory(uuid.NewString())
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, storage.ErrCategoryNotFound) {
			t.Fatalf("error mismatch: want `%v`, got: `%v`", storage.ErrCategoryNotFound, err)
		}
	})

	t.Run("double delete category", func(t *testing.T) {
		category, err := db.CreateCategory(storage.CreateCategoryParams{
			Name:      "Category1",
			Type:      "income",
			Icon:      "icon2",
			Color:     "blue",
			IsDefault: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = db.DeleteCategory(category.Id)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		err = db.DeleteCategory(category.Id)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}

		if !errors.Is(err, storage.ErrCategoryNotFound) {
			t.Errorf("error mismatch: want `%v`, got: `%v`", storage.ErrCategoryNotFound, err)
		}
	})
}

func TestGetCategory(t *testing.T) {
	db, err := sqlite.New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}

	testCategory, err := db.CreateCategory(storage.CreateCategoryParams{
		Name:      "Category1",
		Type:      "income",
		Icon:      "icon2",
		Color:     "blue",
		IsDefault: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

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

			if err != nil && !tc.respError {
				t.Fatalf("unexpected error: %v", err)
			}

			if err != nil && tc.respError {
				if !errors.Is(err, tc.expectedErr) {
					t.Fatalf("error mismatch: want `%v`, got: `%v`", tc.expectedErr, err)
				}
				// We expect an error and got one, so we can return early to avoid further checks.
				return
			}

			if err == nil && tc.respError {
				t.Fatalf("expected error, got nil")
			}

			if category.Id != tc.id {
				t.Errorf("expected category ID to be %v, got %v", tc.id, category.Id)
			}
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
		db, err := sqlite.New(":memory:")
		if err != nil {
			t.Fatalf("failed to create test database: %v", err)
		}

		categories, err := db.GetCategories(storage.GetCategoriesParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(categories) != 0 {
			t.Fatalf("expected 0 categories, got %v", len(categories))
		}
	})

	t.Run("existing categories in database with no params", func(t *testing.T) {
		db, err := sqlite.New(":memory:")
		if err != nil {
			t.Fatalf("failed to create test database: %v", err)
		}

		createdCategories, err := createTestCategories(db)
		if err != nil {
			t.Fatalf("failed to create test categories: %v", err)
		}

		categories, err := db.GetCategories(storage.GetCategoriesParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(createdCategories) != len(categories) {
			t.Errorf("expected %v categories, got %v", len(createdCategories), len(categories))
		}
	})

	t.Run("existing categories in database with type param = income", func(t *testing.T) {
		db, err := sqlite.New(":memory:")
		if err != nil {
			t.Fatalf("failed to create test database: %v", err)
		}

		createdCategories, err := createTestCategories(db)
		if err != nil {
			t.Fatalf("failed to create test categories: %v", err)
		}

		categories, err := db.GetCategories(
			storage.GetCategoriesParams{Type: testutil.Ptr("income")},
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		incomeCategories := testutil.Filter(createdCategories, func(c storage.Category) bool {
			return c.Type == "income"
		})
		if len(categories) != len(incomeCategories) {
			t.Errorf("expected %v categories, got %v", len(incomeCategories), len(categories))
		}
	})
}
