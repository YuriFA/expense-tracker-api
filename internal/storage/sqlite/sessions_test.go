package sqlite_test

import (
	"testing"
	"time"

	"expense-tracker-api/internal/storage"
	"expense-tracker-api/internal/storage/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSession(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newFixture(t)
		session, err := f.DB.CreateSession(storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    f.User.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})

		require.NoError(t, err)
		assert.NotNil(t, session)
		assert.Equal(t, f.User.ID, session.UserID)
		assert.Equal(t, "session-id-123", session.ID)
	})

	t.Run("Non existing user", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.DB.CreateSession(storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    "",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.Error(t, err)
	})
}

func TestGetSessionByID(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newFixture(t)
		session, err := f.DB.CreateSession(storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    f.User.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		retrieved, err := f.DB.GetSessionByID(session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrieved.ID)
		assert.Equal(t, session.UserID, retrieved.UserID)
	})

	t.Run("Non existing session", func(t *testing.T) {
		db := sqlite.NewTestDB(t)
		_, err := db.GetSessionByID("non-existing-session-id")
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})
}

func TestDeleteSession(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		f := newFixture(t)
		session, err := f.DB.CreateSession(storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    f.User.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		err = f.DB.DeleteSession(session.ID)
		require.NoError(t, err)

		_, err = f.DB.GetSessionByID(session.ID)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("Non existing session", func(t *testing.T) {
		f := newFixture(t)
		err := f.DB.DeleteSession("non-existing-session-id")
		require.Error(t, err)
	})
}
