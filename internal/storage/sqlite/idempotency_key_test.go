package sqlite_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/storage"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func defaultIdempotencyKeyParams(userID string) storage.CreateIdempotencyKeyParams {
	return storage.CreateIdempotencyKeyParams{
		IdempotencyKey: "key-1",
		UserID:         userID,
		RequestHash:    hexHash(`{"x":1}`),
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}
}

func hexHash(body string) string {
	h := sha256.Sum256([]byte(body))
	return hex.EncodeToString(h[:])
}

func TestCreateIdempotencyKey(t *testing.T) {
	t.Run("success returns pending status", func(t *testing.T) {
		f := newFixture(t)
		params := defaultIdempotencyKeyParams(f.User.ID)

		ik, err := f.DB.CreateIdempotencyKey(params)
		require.NoError(t, err)
		assert.NotEmpty(t, ik.ID)
		assert.Equal(t, params.IdempotencyKey, ik.IdempotencyKey)
		assert.Equal(t, params.UserID, ik.UserID)
		assert.Equal(t, params.RequestHash, ik.RequestHash)
		assert.Equal(t, "pending", ik.Status)
		assert.Nil(t, ik.ResponseStatus)
		assert.Nil(t, ik.ResponseHeaders)
		assert.Nil(t, ik.ResponseBody)
	})

	t.Run("duplicate key for same user returns ErrIdempotencyKeyInUse", func(t *testing.T) {
		f := newFixture(t)
		params := defaultIdempotencyKeyParams(f.User.ID)

		_, err := f.DB.CreateIdempotencyKey(params)
		require.NoError(t, err)

		_, err = f.DB.CreateIdempotencyKey(params)
		require.ErrorIs(t, err, storage.ErrIdempotencyKeyInUse)
	})

	t.Run("same key for different users is allowed", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		params := defaultIdempotencyKeyParams(f.User.ID)
		params2 := defaultIdempotencyKeyParams(user2.ID)

		_, err := f.DB.CreateIdempotencyKey(params)
		require.NoError(t, err)
		_, err = f.DB.CreateIdempotencyKey(params2)
		require.NoError(t, err)
	})

	t.Run("same key same user different hash is still ErrIdempotencyKeyInUse",
		func(t *testing.T) {
			f := newFixture(t)
			params := defaultIdempotencyKeyParams(f.User.ID)
			_, _ = f.DB.CreateIdempotencyKey(params)

			params.RequestHash = hexHash(`{"x":2}`)
			_, err := f.DB.CreateIdempotencyKey(params)
			require.ErrorIs(t, err, storage.ErrIdempotencyKeyInUse,
				"UNIQUE must be on (user_id, idempotency_key), not include request_hash")
		})
}

func TestGetByUserAndKey(t *testing.T) {
	t.Run("existing key returns row", func(t *testing.T) {
		f := newFixture(t)
		params := defaultIdempotencyKeyParams(f.User.ID)
		seeded, err := f.DB.CreateIdempotencyKey(params)
		require.NoError(t, err)

		got, err := f.DB.GetByUserAndKey(f.User.ID, params.IdempotencyKey)
		require.NoError(t, err)
		assert.Equal(t, seeded.ID, got.ID)
		assert.Equal(t, params.RequestHash, got.RequestHash)
	})

	t.Run("missing key returns ErrIdempotencyKeyNotFound", func(t *testing.T) {
		f := newFixture(t)
		_, err := f.DB.GetByUserAndKey(f.User.ID, "does-not-exist")
		require.ErrorIs(t, err, storage.ErrIdempotencyKeyNotFound)
	})

	t.Run("key from another user is not visible", func(t *testing.T) {
		f := newFixture(t)
		user2 := seedUser(t, f.DB)
		params := defaultIdempotencyKeyParams(f.User.ID)
		_, _ = f.DB.CreateIdempotencyKey(params)

		_, err := f.DB.GetByUserAndKey(user2.ID, params.IdempotencyKey)
		require.ErrorIs(t, err, storage.ErrIdempotencyKeyNotFound)
	})
}

func TestUpdateIdempotencyKey(t *testing.T) {
	t.Run("pending to completed stores response", func(t *testing.T) {
		f := newFixture(t)
		params := defaultIdempotencyKeyParams(f.User.ID)
		seeded, err := f.DB.CreateIdempotencyKey(params)
		require.NoError(t, err)

		status := "completed"
		responseStatus := 201
		headers := []byte(`{"Content-Type":"application/json"}`)
		body := []byte(`{"id":"tx-1"}`)

		updated, err := f.DB.UpdateIdempotencyKey(f.User.ID, seeded.ID, storage.UpdateIdempotencyKeyParams{
			Status:          &status,
			ResponseStatus:  &responseStatus,
			ResponseHeaders: &headers,
			ResponseBody:    &body,
		})
		require.NoError(t, err)
		require.NotNil(t, updated.ResponseStatus)
		assert.Equal(t, 201, *updated.ResponseStatus)
		require.NotNil(t, updated.ResponseHeaders)
		assert.Equal(t, string(headers), *updated.ResponseHeaders)
		assert.Equal(t, body, updated.ResponseBody)
		assert.Equal(t, "completed", updated.Status)
	})

	t.Run("not found returns ErrIdempotencyKeyNotFound", func(t *testing.T) {
		f := newFixture(t)
		status := "completed"
		_, err := f.DB.UpdateIdempotencyKey(f.User.ID, uuid.NewString(), storage.UpdateIdempotencyKeyParams{
			Status: &status,
		})
		require.ErrorIs(t, err, storage.ErrIdempotencyKeyNotFound)
	})

	t.Run("update for another users key returns ErrIdempotencyKeyNotFound",
		func(t *testing.T) {
			f := newFixture(t)
			user2 := seedUser(t, f.DB)
			params := defaultIdempotencyKeyParams(f.User.ID)
			seeded, err := f.DB.CreateIdempotencyKey(params)
			require.NoError(t, err)

			status := "completed"
			_, err = f.DB.UpdateIdempotencyKey(user2.ID, seeded.ID, storage.UpdateIdempotencyKeyParams{
				Status: &status,
			})
			require.ErrorIs(t, err, storage.ErrIdempotencyKeyNotFound)
		})
}

func TestDeleteIdempotencyKey(t *testing.T) {
	t.Run("existing key is removed", func(t *testing.T) {
		f := newFixture(t)
		params := defaultIdempotencyKeyParams(f.User.ID)
		seeded, err := f.DB.CreateIdempotencyKey(params)
		require.NoError(t, err)

		require.NoError(t, f.DB.DeleteIdempotencyKey(f.User.ID, seeded.ID))
		_, err = f.DB.GetByUserAndKey(f.User.ID, params.IdempotencyKey)
		require.ErrorIs(t, err, storage.ErrIdempotencyKeyNotFound)
	})

	t.Run("non-existing id is a no-op", func(t *testing.T) {
		f := newFixture(t)
		err := f.DB.DeleteIdempotencyKey(f.User.ID, uuid.NewString())
		require.NoError(t, err)
	})
}

func TestDeleteExpiredIdempotencyKeys(t *testing.T) {
	t.Run("removes only expired rows", func(t *testing.T) {
		f := newFixture(t)

		expired := defaultIdempotencyKeyParams(f.User.ID)
		expired.IdempotencyKey = "expired"
		expired.ExpiresAt = time.Now().UTC().Add(-1 * time.Hour)
		_, err := f.DB.CreateIdempotencyKey(expired)
		require.NoError(t, err)

		live := defaultIdempotencyKeyParams(f.User.ID)
		live.IdempotencyKey = "live"
		live.ExpiresAt = time.Now().UTC().Add(1 * time.Hour)
		_, err = f.DB.CreateIdempotencyKey(live)
		require.NoError(t, err)

		count, err := f.DB.DeleteExpiredIdempotencyKeys()
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)

		_, err = f.DB.GetByUserAndKey(f.User.ID, "expired")
		require.ErrorIs(t, err, storage.ErrIdempotencyKeyNotFound)

		_, err = f.DB.GetByUserAndKey(f.User.ID, "live")
		require.NoError(t, err)
	})

	t.Run("no expired rows returns zero count", func(t *testing.T) {
		f := newFixture(t)
		live := defaultIdempotencyKeyParams(f.User.ID)
		live.ExpiresAt = time.Now().UTC().Add(1 * time.Hour)
		_, _ = f.DB.CreateIdempotencyKey(live)

		count, err := f.DB.DeleteExpiredIdempotencyKeys()
		require.NoError(t, err)
		assert.Equal(t, int64(0), count)
	})
}
