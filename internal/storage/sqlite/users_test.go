package sqlite_test

import (
	"testing"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		_, err := db.CreateUser(storage.CreateUserParams{
			Email:        "user1@example.com",
			PasswordHash: "hashedpassword1",
		})
		require.NoError(t, err)
	})

	t.Run("non duplicate user ids", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		user1 := seedUser(t, db, "user1@example.com", "hashedpassword1")
		user2 := seedUser(t, db, "user2@example.com", "hashedpassword2")
		require.NotEqual(t, user1.Id, user2.Id)
	})

	t.Run("duplicate email", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		seedUser(t, db, "user1@example.com", "hashedpassword1")
		_, err := db.CreateUser(storage.CreateUserParams{
			Email:        "user1@example.com",
			PasswordHash: "hashedpassword2",
		})
		require.ErrorIs(t, err, storage.ErrUserAlreadyExists)
	})
}

func TestGetUserByEmail(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		seedUser(t, db, "user1@example.com", "hashedpassword1")
		user, err := db.GetUserByEmail("user1@example.com")
		require.NoError(t, err)
		require.Equal(t, "user1@example.com", user.Email)
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
		seededUser := seedUser(t, db, "user1@example.com", "hashedpassword1")
		user, err := db.GetUserByID(seededUser.Id)
		require.NoError(t, err)
		require.Equal(t, seededUser.Id, user.Id)
	})

	t.Run("not found", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		_, err := db.GetUserByID(uuid.NewString())
		require.ErrorIs(t, err, storage.ErrUserNotFound)
	})
}
