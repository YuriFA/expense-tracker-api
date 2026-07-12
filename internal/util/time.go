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
