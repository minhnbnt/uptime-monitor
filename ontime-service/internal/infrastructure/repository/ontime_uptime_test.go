package repository

import (
	"math"
	"testing"
	"time"
)

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("got %f, want %f", got, want)
	}
}

func TestUptimePercent(t *testing.T) {

	d := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("full window", func(t *testing.T) {
		row := UptimeRow{
			OnlineSeconds: 12 * 3600,
			ObservedFrom:  d,
			ObservedTo:    d.Add(24 * time.Hour),
		}
		if got := row.UptimePercent(); got != 50 {
			t.Errorf("UptimePercent() = %f, want 50", got)
		}
	})

	t.Run("shrunk partial window", func(t *testing.T) {
		row := UptimeRow{
			OnlineSeconds: 12 * 3600,
			ObservedFrom:  d.Add(6 * time.Hour),
			ObservedTo:    d.Add(24 * time.Hour),
		}
		assertFloat(t, row.UptimePercent(), 100.0*12.0/18.0)
	})

	t.Run("zero online seconds", func(t *testing.T) {
		row := UptimeRow{
			OnlineSeconds: 0,
			ObservedFrom:  d,
			ObservedTo:    d.Add(24 * time.Hour),
		}
		if got := row.UptimePercent(); got != 0 {
			t.Errorf("UptimePercent() = %f, want 0", got)
		}
	})

	t.Run("zero-width window guards division by zero", func(t *testing.T) {
		row := UptimeRow{
			OnlineSeconds: 3600,
			ObservedFrom:  d,
			ObservedTo:    d,
		}
		got := row.UptimePercent()
		if got != 0 {
			t.Errorf("UptimePercent() = %f, want 0 (no NaN)", got)
		}
	})
}
