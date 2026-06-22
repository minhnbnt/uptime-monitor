package utils

import "time"

func TruncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func Last30Days() []time.Time {

	until := TruncateDay(time.Now())
	since := until.AddDate(0, 0, -29)

	return BuildDateRange(since, until)
}

func BuildDateRange(from, to time.Time) []time.Time {

	start := TruncateDay(from)
	end := TruncateDay(to)

	dates := make([]time.Time, 0, int(end.Sub(start).Hours()/24)+1)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}

	return dates
}
