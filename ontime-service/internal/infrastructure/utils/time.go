package utils

import (
	"fmt"
	"time"

	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
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
		return 0, fmt.Errorf("%w: %q is not a valid duration (e.g. \"15m\", \"1h\")", apperrors.ErrBadRequest, s)
	}

	if d < time.Minute {
		return 0, fmt.Errorf("%w: resolution %s is below minimum (1m)", apperrors.ErrBadRequest, d)
	}

	return d, nil
}

type Interval struct {
	Start time.Time
	End   time.Time
}

func SplitIntervals(from, to time.Time, resolution time.Duration) []Interval {

	intervals := make([]Interval, 0)
	for start := from; start.Before(to); start = start.Add(resolution) {

		end := start.Add(resolution)
		if end.After(to) {
			end = to
		}

		intervals = append(intervals, Interval{start, end})
	}

	return intervals
}
