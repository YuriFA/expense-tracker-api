package handlers

import (
	"time"
)

func endOfDay(date *time.Time) *time.Time {
	if date == nil {
		return nil
	}

	toDate := (*date).AddDate(0, 0, 1).Add(-time.Nanosecond)
	return &toDate
}
