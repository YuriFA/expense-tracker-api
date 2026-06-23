package handlers

import (
	"testing"
	"time"

	"expense-tracker-api/internal/testutil"
)

func TestEndOfDay(t *testing.T) {
	cases := map[string]struct {
		date     time.Time
		expected time.Time
	}{
		"midnight": {
			date:     time.Date(2009, time.November, 10, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2009, time.November, 10, 23, 59, 59, 999999999, time.UTC),
		},
		"midday": {
			date:     time.Date(2009, time.November, 10, 12, 0, 0, 0, time.UTC),
			expected: time.Date(2009, time.November, 10, 23, 59, 59, 999999999, time.UTC),
		},
		"random time": {
			date:     time.Date(2009, time.November, 10, 15, 30, 45, 123456789, time.UTC),
			expected: time.Date(2009, time.November, 10, 23, 59, 59, 999999999, time.UTC),
		},
		"end of month": {
			date:     time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2024, 1, 31, 23, 59, 59, 999999999, time.UTC),
		},
		"leap year": {
			date:     time.Date(2020, 2, 29, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2020, 2, 29, 23, 59, 59, 999999999, time.UTC),
		},
		"non leap year": {
			date:     time.Date(2021, 2, 28, 0, 0, 0, 0, time.UTC),
			expected: time.Date(2021, 2, 28, 23, 59, 59, 999999999, time.UTC),
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			result := endOfDay(tc.date)
			testutil.AssertEqual(t, tc.expected, result)
		})
	}
}
