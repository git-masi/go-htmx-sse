package utils

import "time"

func ParseDBDate(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05.999999-07:00", s)
}
