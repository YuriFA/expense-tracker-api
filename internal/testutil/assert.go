package testutil

import (
	"testing"

	"github.com/google/uuid"
)

func AssertValidUUID(t *testing.T, id string) {
	t.Helper()
	if err := uuid.Validate(id); err != nil {
		t.Errorf("expected valid UUID, got %v", id)
	}
}
