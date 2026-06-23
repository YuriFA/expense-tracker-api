package handlers

import (
	"time"
)

func endOfDay(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(),
		23, 59, 59, 999999999, date.Location())
}
