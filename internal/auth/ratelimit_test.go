package auth_test

import (
	"testing"
	"time"

	"github.com/yurifa/expense-tracker-api/internal/auth"

	"github.com/stretchr/testify/require"
)

func TestLoginRateLimiter(t *testing.T) {
	t.Parallel()
	t.Run("TestLoginRateLimiter", func(t *testing.T) {
		t.Parallel()
		rl := auth.NewLoginRateLimiter(3, time.Hour)

		key := "test-key"

		require.False(t, rl.IsLocked(key), "expected key to be unlocked initially")

		// Record failures and check if the key gets locked
		for range 3 {
			rl.RecordFailure(key)
		}

		require.True(t, rl.IsLocked(key), "expected key to be locked after 3 failures")

		// Record success and check if the key gets unlocked
		rl.RecordSuccess(key)
		require.False(t, rl.IsLocked(key), "expected key to be unlocked after success")
	})
}
