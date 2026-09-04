package utils

import (
	"fmt"
	"time"

	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
)

func TruncateDay(t time.Time) time.Time {
	return TruncateDayIn(t, time.UTC)
}

// TruncateDayIn floors t to midnight in loc. Conversion is instant-based,
// so inputs carrying any location (UTC, DB session zone, user tz) map to
// the same calendar day in loc.
func TruncateDayIn(t time.Time, loc *time.Location) time.Time {

	if loc == nil {
		loc = time.UTC
	}

	lt := t.In(loc)

	return time.Date(lt.Year(), lt.Month(), lt.Day(), 0, 0, 0, 0, loc)
}

func Last30Days() []time.Time {
	return Last30DaysIn(time.UTC)
}

func Last30DaysIn(loc *time.Location) []time.Time {

	until := TruncateDayIn(time.Now(), loc)
	since := until.AddDate(0, 0, -29)

	return BuildDateRangeIn(since, until, loc)
}

func BuildDateRange(from, to time.Time) []time.Time {
	return BuildDateRangeIn(from, to, time.UTC)
}

func BuildDateRangeIn(from, to time.Time, loc *time.Location) []time.Time {

	start := TruncateDayIn(from, loc)
	end := TruncateDayIn(to, loc)

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
