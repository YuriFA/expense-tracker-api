package sqlite_test

import (
	"context"
	"testing"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/storage"
	"github.com/yurifa/expense-tracker-api/internal/storage/sqlite"
	"github.com/yurifa/expense-tracker-api/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSession(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		session, err := f.DB.CreateSession(context.Background(), storage.CreateSessionParams{
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
		t.Parallel()
		f := newFixture(t)
		_, err := f.DB.CreateSession(context.Background(), storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    "",
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.Error(t, err)
	})
}

func TestGetSessionByID(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		session, err := f.DB.CreateSession(context.Background(), storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    f.User.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		retrieved, err := f.DB.GetSessionByID(context.Background(), session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, retrieved.ID)
		assert.Equal(t, session.UserID, retrieved.UserID)
	})

	t.Run("Non existing session", func(t *testing.T) {
		t.Parallel()
		db := sqlite.NewTestDB(t)
		_, err := db.GetSessionByID(context.Background(), "non-existing-session-id")
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})
}

func TestDeleteSession(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		session, err := f.DB.CreateSession(context.Background(), storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    f.User.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		err = f.DB.DeleteSession(context.Background(), session.ID)
		require.NoError(t, err)

		_, err = f.DB.GetSessionByID(context.Background(), session.ID)
		require.ErrorIs(t, err, storage.ErrSessionNotFound)
	})

	t.Run("Non existing session", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		err := f.DB.DeleteSession(context.Background(), "non-existing-session-id")
		require.Error(t, err)
	})
}

func TestExtendSession(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		session, err := f.DB.CreateSession(context.Background(), storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    f.User.ID,
			ExpiresAt: time.Now().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		newExpiry := time.Now().Add(48 * time.Hour)
		err = f.DB.ExtendSession(context.Background(), session.ID, newExpiry)
		require.NoError(t, err)

		updated, err := f.DB.GetSessionByID(context.Background(), session.ID)
		require.NoError(t, err)
		assert.Equal(
			t,
			newExpiry.UnixMilli(),
			testutil.ParseDatetime(t, updated.ExpiresAt).UnixMilli(),
		)
	})

	t.Run("Non existing session", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)
		err := f.DB.ExtendSession(context.Background(), "non-existing-session-id", time.Now().Add(48*time.Hour))
		require.Error(t, err)
	})
}

func TestDeleteExpiredSessions(t *testing.T) {
	t.Parallel()
	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		f := newFixture(t)

		_, err := f.DB.CreateSession(context.Background(), storage.CreateSessionParams{
			SessionID: "session-id-123",
			UserID:    f.User.ID,
			ExpiresAt: time.Now().UTC().Add(-24 * time.Hour),
		})
		require.NoError(t, err)

		_, err = f.DB.CreateSession(context.Background(), storage.CreateSessionParams{
			SessionID: "session-id-124",
			UserID:    f.User.ID,
			ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
		})
		require.NoError(t, err)

		count, err := f.DB.DeleteExpiredSessions(context.Background())
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		_, err = f.DB.GetSessionByID(context.Background(), "session-id-123")
		require.ErrorIs(t, err, storage.ErrSessionNotFound)

		_, err = f.DB.GetSessionByID(context.Background(), "session-id-124")
		require.NoError(t, err)
	})
}
