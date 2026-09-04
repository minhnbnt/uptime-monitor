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

func TestTruncateDayIn(t *testing.T) {
	saigon := time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)

	// 2026-09-04T01:00:00Z is still Sep 4 in UTC but already 08:00 in Saigon.
	got := TruncateDayIn(time.Date(2026, 9, 4, 1, 0, 0, 0, time.UTC), saigon)
	want := time.Date(2026, 9, 4, 0, 0, 0, 0, saigon)
	if !got.Equal(want) {
		t.Errorf("TruncateDayIn = %v, want %v", got, want)
	}

	// 2026-09-04T16:30:00Z is still Sep 4 23:30 in Saigon, so it floors to Sep 4.
	// (17:00Z would already be Sep 5 in Saigon while still Sep 4 in UTC.)
	got = TruncateDayIn(time.Date(2026, 9, 4, 16, 30, 0, 0, time.UTC), saigon)
	want = time.Date(2026, 9, 4, 0, 0, 0, 0, saigon)
	if !got.Equal(want) {
		t.Errorf("TruncateDayIn = %v, want %v", got, want)
	}

	// nil loc falls back to UTC, same as TruncateDay.
	got = TruncateDayIn(time.Date(2026, 9, 4, 16, 30, 0, 0, time.UTC), nil)
	if !got.Equal(TruncateDay(time.Date(2026, 9, 4, 16, 30, 0, 0, time.UTC))) {
		t.Errorf("TruncateDayIn(nil) = %v, want UTC truncation", got)
	}
}

func TestLast30DaysIn(t *testing.T) {
	saigon := time.FixedZone("Asia/Ho_Chi_Minh", 7*3600)

	got := Last30DaysIn(saigon)
	if len(got) != 30 {
		t.Fatalf("len = %d, want 30", len(got))
	}
	for _, d := range got {
		lt := d.In(saigon)
		if lt.Hour() != 0 || lt.Minute() != 0 || lt.Second() != 0 {
			t.Errorf("date %v is not midnight in tz", d)
		}
		if d.Location() != saigon {
			t.Errorf("date %v does not carry the tz location", d)
		}
	}
}
