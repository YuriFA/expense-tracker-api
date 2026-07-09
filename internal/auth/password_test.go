package auth_test

import (
	"testing"

	"expense-tracker-api/internal/auth"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		password := "mysecretpassword"
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		require.NoError(t, auth.VerifyPassword(hash, password))
		require.Error(t, auth.VerifyPassword(hash, "wrongpassword"))
	})

	t.Run("extra long password", func(t *testing.T) {
		password := "thisisaverylongpasswordthatexceedstheusualmaximumlengthandshouldstillworkproperly"
		_, err := auth.HashPassword(password)
		require.ErrorAs(t, err, &bcrypt.ErrPasswordTooLong)
	})
}

func TestVerifyPassword(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		password := "mysecretpassword"
		hash, err := auth.HashPassword(password)
		require.NoError(t, err)
		require.NoError(t, auth.VerifyPassword(hash, password))
		require.Error(t, auth.VerifyPassword(hash, "wrongpassword"))
	})

	t.Run("invalid hash", func(t *testing.T) {
		require.Error(t, auth.VerifyPassword("invalidhash", "password"))
	})
}
