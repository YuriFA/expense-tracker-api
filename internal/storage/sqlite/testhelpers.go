package sqlite

import (
	"testing"
)

func NewTestDB(t *testing.T) *Storage {
	t.Helper()
	db, err := New(":memory:")
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	return db
}
