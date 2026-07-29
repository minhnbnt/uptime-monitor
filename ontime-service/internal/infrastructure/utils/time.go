package utils

import (
	"fmt"
	"time"
)

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

func ParseResolution(s string) (time.Duration, error) {

	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("invalid resolution")
	}

	if d < time.Minute {
		return 0, fmt.Errorf("resolution must be at least 1 minute")
	}

	return d, nil
}

func SplitIntervals(from, to time.Time, resolution time.Duration) [][2]time.Time {

	intervals := make([][2]time.Time, 0)
	for start := from; start.Before(to); start = start.Add(resolution) {

		end := start.Add(resolution)
		if end.After(to) {
			end = to
		}

		intervals = append(intervals, [2]time.Time{start, end})
	}

	return intervals
}
