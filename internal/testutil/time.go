package testutil

import (
	"testing"
	"time"
)

func ParseDatetime(t *testing.T, datetime string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		t.Fatalf("invalid timestamp %q: %v", datetime, err)
	}
	return parsed
}

func GetTimeFromStr(t *testing.T, timeStr string) *time.Time {
	t.Helper()
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}
	return &parsedTime
}

func AssertTimeEqual(t *testing.T, expected, actual time.Time) {
	t.Helper()
	if !expected.Equal(actual) {
		t.Errorf("time mismatch:\nexpected: %v\nactual:   %v", expected, actual)
	}
}
