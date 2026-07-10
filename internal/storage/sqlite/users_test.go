package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

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

	// TODO: Categories for user
	// t.Run("creates default categories", func(t *testing.T) {
	// 	db := sqlite.NewTestDB(t)
	// 	user := seedUser(t, db, "user1@example.com")
	// 	categories, err := db.GetCategories()
	// 	require.NoError(t, err)
	// 	require.NotEmpty(t, categories)
	// })

	t.Run("non duplicate user ids", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user1 := seedUser(t, db, "user1@example.com")
		user2 := seedUser(t, db, "user2@example.com")
		require.NotEqual(t, user1.ID, user2.ID)
	})

	t.Run("duplicate email", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		seedUser(t, db, "user1@example.com")
		_, err := db.RegisterUser(storage.RegisterUserParams{
			Email:        "user1@example.com",
			PasswordHash: "hashedpassword2",
		})
		require.ErrorIs(t, err, storage.ErrUserAlreadyExists)
	})
}

func TestGetUserByEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		seedUser(t, db, "user1@example.com")
		user, err := db.GetUserByEmail("user1@example.com")
		require.NoError(t, err)
		require.Equal(t, "user1@example.com", user.Email)
		require.NotEmpty(t, user.PasswordHash)
	})

	t.Run("not found", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		_, err := db.GetUserByEmail("nonexistent@example.com")
		require.ErrorIs(t, err, storage.ErrUserNotFound)
	})
}

func TestGetUserByID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		seededUser := seedUser(t, db, "user1@example.com")
		user, err := db.GetUserByID(seededUser.ID)
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
