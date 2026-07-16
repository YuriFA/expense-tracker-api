package auth

import (
	"crypto/rand"
	"encoding/hex"
)

// sessionTokenBytes is the entropy size for a session token (256 bits).
const sessionTokenBytes = 32

func GenerateSessionToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
