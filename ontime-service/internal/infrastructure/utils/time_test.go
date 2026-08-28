package utils

import (
	"testing"
	"time"
)

// notifBuildDateRange mirrors notification-service's copy of BuildDateRange.
// It is the reference the ontime implementation must stay identical to.
func notifBuildDateRange(from, to time.Time) []time.Time {
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)

	dates := make([]time.Time, 0, int(end.Sub(start).Hours()/24)+1)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}
	return dates
}

func TestBuildDateRangeMatchesNotification(t *testing.T) {
	cases := []struct {
		from, to time.Time
	}{
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)},
		{time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC), time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)},
		{time.Date(2026, 12, 30, 0, 0, 0, 0, time.UTC), time.Date(2027, 1, 2, 0, 0, 0, 0, time.UTC)},
	}

	for _, c := range cases {
		got := BuildDateRange(c.from, c.to)
		want := notifBuildDateRange(c.from, c.to)
		if len(got) != len(want) {
			t.Fatalf("BuildDateRange(%v,%v) len = %d, want %d", c.from, c.to, len(got), len(want))
		}
		for i := range want {
			if !got[i].Equal(want[i]) {
				t.Errorf("BuildDateRange(%v,%v)[%d] = %v, want %v", c.from, c.to, i, got[i], want[i])
			}
		}
	}
}
