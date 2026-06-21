package handlers

import (
	"time"
)

func parseDateRange(fromStr, toStr *string) (from, to *string, err error) {
	var fromDateRFC3339 *string
	if fromStr != nil {
		fromDate, err := time.Parse("2006-01-02", *fromStr)
		if err != nil {
			return nil, nil, err
		}
		formattedDate := fromDate.Format(time.RFC3339)
		fromDateRFC3339 = &formattedDate
	}
	var toDateRFC3339 *string
	if toStr != nil {
		toDate, err := time.Parse("2006-01-02", *toStr)
		if err != nil {
			return nil, nil, err
		}
		toDate = toDate.AddDate(0, 0, 1).Add(-time.Nanosecond)
		formattedDate := toDate.Format(time.RFC3339)
		toDateRFC3339 = &formattedDate
	}

	return fromDateRFC3339, toDateRFC3339, nil
}
