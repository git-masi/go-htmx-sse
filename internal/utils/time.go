package utils

import (
	"time"
)

func GetWeekStartEnd(now time.Time) (time.Time, time.Time) {
	wkd := int(now.Weekday())

	if wkd == 0 {
		return now, now.AddDate(0, 0, 6)
	}

	startDate := now.AddDate(0, 0, -wkd)
	endDate := startDate.AddDate(0, 0, 6)

	return startDate, endDate
}
