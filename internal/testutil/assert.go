package testutil

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func AssertErrorIs(t *testing.T, err, target error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if !errors.Is(err, target) {
		t.Fatalf("error mismatch: want `%v`, got `%v`", target, err)
	}
}

func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func AssertEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if got != want {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func AssertNotEqual[T comparable](t *testing.T, want, got T) {
	t.Helper()
	if got == want {
		t.Fatalf("expected values to differ, but both are %v", want)
	}
}

func AssertNotNil(t *testing.T, got any) {
	t.Helper()
	if got == nil {
		t.Fatalf("expected not nil, got nil")
	}
}

func AssertEmpty[T any](t *testing.T, slice []T) {
	t.Helper()
	if len(slice) != 0 {
		t.Fatalf("expected empty, got %v", slice)
	}
}

func AssertValidUUID(t *testing.T, id string) {
	t.Helper()
	if err := uuid.Validate(id); err != nil {
		t.Fatalf("expected valid UUID, got %v", id)
	}
}

func ParseDatetime(t *testing.T, datetime string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		t.Fatalf("invalid timestamp %q: %v", datetime, err)
	}
	return parsed
}

func AssertDatetimeChanged(t *testing.T, before, after string) {
	t.Helper()
	if ParseDatetime(t, before).Equal(ParseDatetime(t, after)) {
		t.Errorf("expected timestamp to change: before=%s after=%s", before, after)
	}
}
