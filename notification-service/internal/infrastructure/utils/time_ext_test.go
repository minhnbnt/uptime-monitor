package utils

import (
	"testing"
	"time"
)

func TestBuildDateRange(t *testing.T) {
	t.Run("same day returns single date", func(t *testing.T) {
		day := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)
		dates := BuildDateRange(day, day)
		if len(dates) != 1 {
			t.Fatalf("got %d dates, want 1", len(dates))
		}
		if !dates[0].Equal(day) {
			t.Errorf("got %v, want %v", dates[0], day)
		}
	})

	t.Run("consecutive days", func(t *testing.T) {
		from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
		dates := BuildDateRange(from, to)
		if len(dates) != 3 {
			t.Fatalf("got %d dates, want 3", len(dates))
		}
		expected := []time.Time{
			time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC),
			time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
		}
		for i, exp := range expected {
			if !dates[i].Equal(exp) {
				t.Errorf("dates[%d] = %v, want %v", i, dates[i], exp)
			}
		}
	})

	t.Run("from after to returns empty", func(t *testing.T) {
		from := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC)
		dates := BuildDateRange(from, to)
		if len(dates) != 0 {
			t.Errorf("got %d dates, want 0", len(dates))
		}
	})

	t.Run("truncates times", func(t *testing.T) {
		from := time.Date(2026, 6, 1, 10, 30, 0, 0, time.UTC)
		to := time.Date(2026, 6, 2, 15, 45, 0, 0, time.UTC)
		dates := BuildDateRange(from, to)
		if len(dates) != 2 {
			t.Fatalf("got %d dates, want 2", len(dates))
		}
		if !dates[0].Equal(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("dates[0] = %v, want truncated day", dates[0])
		}
	})

	t.Run("across month boundary", func(t *testing.T) {
		from := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC)
		dates := BuildDateRange(from, to)
		if len(dates) != 3 {
			t.Fatalf("got %d dates, want 3", len(dates))
		}
		if !dates[1].Equal(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)) {
			t.Errorf("dates[1] = %v, want Feb 1", dates[1])
		}
	})

	t.Run("across year boundary", func(t *testing.T) {
		from := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
		to := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		dates := BuildDateRange(from, to)
		if len(dates) != 2 {
			t.Fatalf("got %d dates, want 2", len(dates))
		}
	})
}
