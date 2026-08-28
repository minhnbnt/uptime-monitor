package utils

import (
	"testing"
	"time"
)

func TestTruncateDay(t *testing.T) {
	tests := []struct {
		input time.Time
		want  time.Time
	}{
		{
			input: time.Date(2026, 6, 4, 15, 30, 45, 123, time.UTC),
			want:  time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			input: time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
			want:  time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
		},
		{
			input: time.Date(2026, 12, 31, 23, 59, 59, 999, time.UTC),
			want:  time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			input: time.Date(2026, 6, 4, 15, 0, 0, 0, time.FixedZone("+07", 7*3600)),
			want:  time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		got := TruncateDay(tt.input)
		if !got.Equal(tt.want) {
			t.Errorf("TruncateDay(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestLast30Days(t *testing.T) {
	dates := Last30Days()

	if len(dates) != 30 {
		t.Fatalf("got %d dates, want 30", len(dates))
	}

	for i, d := range dates {
		truncated := TruncateDay(d)
		if !truncated.Equal(d) {
			t.Errorf("dates[%d] = %v is not truncated", i, d)
		}
	}

	for i := 1; i < len(dates); i++ {
		diff := dates[i].Sub(dates[i-1])
		if diff.Hours() != 24 {
			t.Errorf("gap between dates[%d] and dates[%d] = %v, want 24h", i-1, i, diff)
		}
	}

	today := TruncateDay(time.Now())
	if !dates[len(dates)-1].Equal(today) {
		t.Errorf("last date = %v, want today %v", dates[len(dates)-1], today)
	}
}

// ontimeBuildDateRange mirrors ontime-service's copy of BuildDateRange.
// It is the reference the notification implementation must stay identical to.
func ontimeBuildDateRange(from, to time.Time) []time.Time {
	start := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	end := time.Date(to.Year(), to.Month(), to.Day(), 0, 0, 0, 0, time.UTC)

	dates := make([]time.Time, 0, int(end.Sub(start).Hours()/24)+1)
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d)
	}
	return dates
}

func TestBuildDateRangeMatchesOntime(t *testing.T) {
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
		want := ontimeBuildDateRange(c.from, c.to)
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
