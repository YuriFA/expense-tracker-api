package sqlite_test

import (
	"testing"

	"github.com/yurifa/expense-tracker-api/internal/storage"
	"github.com/yurifa/expense-tracker-api/internal/storage/sqlite"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegisterUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user, err := db.RegisterUser(storage.RegisterUserParams{
			Email:        "user1@example.com",
			PasswordHash: "hashedpassword1",
		})
		require.NoError(t, err)
		require.NotEmpty(t, user.ID)
		require.Equal(t, "user1@example.com", user.Email)
		require.Empty(t, user.PasswordHash)
	})

	t.Run("creates default categories", func(t *testing.T) {
		f := newFixture(t)
		categories, err := f.DB.GetCategories(f.User.ID, storage.GetCategoriesParams{})
		require.NoError(t, err)
		require.NotEmpty(t, categories)
	})

	t.Run("non duplicate user ids", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		require.NotEqual(t, f.User.ID, user2.ID)
	})

	t.Run("duplicate email", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.DB.RegisterUser(storage.RegisterUserParams{
			Email:        f.User.Email,
			PasswordHash: "hashedpassword2",
		})
		require.ErrorIs(t, err, storage.ErrUserAlreadyExists)
	})
}

func TestGetUserByEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newFixture(t)
		user, err := f.DB.GetUserByEmail(f.User.Email)
		require.NoError(t, err)
		require.Equal(t, f.User.Email, user.Email)
		require.NotEmpty(t, user.PasswordHash)
	})

	t.Run("not found", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.DB.GetUserByEmail("nonexistent@example.com")
		require.ErrorIs(t, err, storage.ErrUserNotFound)
	})
}

func TestGetUserByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		f := newFixture(t)
		seededUser := seedUser(t, f.DB)
		user, err := f.DB.GetUserByID(seededUser.ID)
		require.NoError(t, err)
		require.Equal(t, seededUser.ID, user.ID)
		require.Empty(t, user.PasswordHash)
	})

	t.Run("not found", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		_, err := db.GetUserByID(uuid.NewString())
		require.ErrorIs(t, err, storage.ErrUserNotFound)
	})
}
