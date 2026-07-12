package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoginRateLimiter(t *testing.T) {
	t.Run("TestLoginRateLimiter", func(t *testing.T) {
		rl := NewLoginRateLimiter(3, time.Hour)

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
