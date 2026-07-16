package util

import (
	"time"
)

func ParseDatetime(datetime string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		return time.Time{}, err
	}
	return parsed, nil
}

func EndOfDay(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(),
		23, 59, 59, 999999999, date.Location())
}
