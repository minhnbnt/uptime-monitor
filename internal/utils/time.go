package utils

import "time"

func TruncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func Last30Days() []time.Time {
	until := TruncateDay(time.Now())
	since := until.AddDate(0, 0, -29)

	dates := make([]time.Time, 0, 30)
	for d := since; !d.After(until); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}

	return dates
}
