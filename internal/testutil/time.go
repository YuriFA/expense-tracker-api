package testutil

import (
	"testing"
	"time"
)

func GetTimeFromStr(t *testing.T, timeStr string) *time.Time {
	parsedTime, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}
	return &parsedTime
}
