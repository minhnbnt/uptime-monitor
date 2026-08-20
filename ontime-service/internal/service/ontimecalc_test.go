package service

import (
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
)

func e(anchor, t time.Time, status domain.ServerStatus) ontimerepo.RawEvent {
	return ontimerepo.RawEvent{Day: anchor, Time: t, Status: string(status), Src: "test"}
}

func day(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func tm(y, m, d, h, mn int) time.Time {
	return time.Date(y, time.Month(m), d, h, mn, 0, 0, time.UTC)
}

func approx(a, b float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-9
}

func TestCalculateOntime_NoData(t *testing.T) {
	d := day(2026, 6, 4)
	res := OntimeCalculator{}.CalculateOntime(nil, d, d.Add(24*time.Hour))
	if res.HasData {
		t.Error("HasData = true, want false")
	}
	if res.Partial {
		t.Error("Partial = true, want false")
	}
	if res.TotalSeconds != 24*3600 {
		t.Errorf("TotalSeconds = %v, want 86400", res.TotalSeconds)
	}
}

func TestCalculateOntime_NoKnownState(t *testing.T) {
	// NULL-joined placeholder rows (empty status) count as unknown — data
	// exists nowhere, so the result is no-data, never silently 0%.
	d := day(2026, 6, 4)
	events := []ontimerepo.RawEvent{
		e(d, tm(2026, 6, 4, 6, 0), domain.ServerStatus("")),
		e(d, tm(2026, 6, 4, 12, 0), domain.ServerStatus("")),
	}
	res := OntimeCalculator{}.CalculateOntime(events, d, d.Add(24*time.Hour))
	if res.HasData {
		t.Error("HasData = true, want false")
	}
}

func TestCalculateOntime_KnownBoundary(t *testing.T) {
	d := day(2026, 6, 4)
	// An ON event exactly at the window start anchors the whole window.
	events := []ontimerepo.RawEvent{e(d, d, domain.StatusOn)}
	res := OntimeCalculator{}.CalculateOntime(events, d, d.Add(24*time.Hour))
	if !res.HasData {
		t.Fatal("HasData = false, want true")
	}
	if res.Partial {
		t.Error("Partial = true, want false (known at boundary)")
	}
	if !res.ObservedFrom.Equal(d) {
		t.Errorf("ObservedFrom = %v, want %v", res.ObservedFrom, d)
	}
	if !approx(res.Uptime, 100) {
		t.Errorf("Uptime = %v, want 100", res.Uptime)
	}
	if !approx(res.OnlineSeconds, 24*3600) {
		t.Errorf("OnlineSeconds = %v, want 86400", res.OnlineSeconds)
	}
	if !approx(res.TotalSeconds, 24*3600) {
		t.Errorf("TotalSeconds = %v, want 86400", res.TotalSeconds)
	}
}

func TestCalculateOntime_PartialNoBoundary(t *testing.T) {
	d := day(2026, 6, 4)
	// No event before the window; first known event at 06:00 → window
	// shrinks to start at 06:00, flagged Partial, ObservedFrom = 06:00.
	events := []ontimerepo.RawEvent{e(d, tm(2026, 6, 4, 6, 0), domain.StatusOn)}
	res := OntimeCalculator{}.CalculateOntime(events, d, d.Add(24*time.Hour))
	if !res.HasData {
		t.Fatal("HasData = false, want true")
	}
	if !res.Partial {
		t.Error("Partial = false, want true")
	}
	if !res.ObservedFrom.Equal(tm(2026, 6, 4, 6, 0)) {
		t.Errorf("ObservedFrom = %v, want 06:00", res.ObservedFrom)
	}
	if !approx(res.Uptime, 100) {
		t.Errorf("Uptime = %v, want 100", res.Uptime)
	}
	if !approx(res.TotalSeconds, 18*3600) {
		t.Errorf("TotalSeconds = %v, want %v (18h observed sub-window)", res.TotalSeconds, 18*3600)
	}
}

func TestCalculateOntime_PartialFullDayUptimeWithStatusChanges(t *testing.T) {
	d := day(2026, 6, 4)
	// No boundary: ON at 06:00, OFF at 12:00 → observed 06:00–24:00, 6h online.
	events := []ontimerepo.RawEvent{
		e(d, tm(2026, 6, 4, 6, 0), domain.StatusOn),
		e(d, tm(2026, 6, 4, 12, 0), domain.StatusOff),
	}
	res := OntimeCalculator{}.CalculateOntime(events, d, d.Add(24*time.Hour))
	if !res.HasData || !res.Partial {
		t.Fatalf("HasData=%v Partial=%v, want true/true", res.HasData, res.Partial)
	}
	if !approx(res.Uptime, 100*6.0/18.0) {
		t.Errorf("Uptime = %v, want %v", res.Uptime, 100*6.0/18.0)
	}
}
func TestCalculateDayOntime_PastDayComputedOverOwnDay(t *testing.T) {
	// A past day whose events live entirely inside that day must be computed
	// over its own 24h window — NOT over today's window (which would exclude
	// these events and report no-data).
	d := day(2026, 6, 4)
	now := tm(2026, 6, 10, 12, 0)
	events := []ontimerepo.RawEvent{e(d, tm(2026, 6, 4, 6, 0), domain.StatusOn)}
	res := OntimeCalculator{}.CalculateDayOntime(events, d, now)
	if !res.HasData {
		t.Fatal("HasData = false, want true — past day must compute over its own window")
	}
	if !approx(res.Uptime, 100) {
		t.Errorf("Uptime = %v, want 100", res.Uptime)
	}
}

func TestCalculateDayOntime_CurrentDayClampedToNow(t *testing.T) {
	d := day(2026, 6, 4)
	now := tm(2026, 6, 4, 12, 0)
	events := []ontimerepo.RawEvent{e(d, d, domain.StatusOn)}
	res := OntimeCalculator{}.CalculateDayOntime(events, d, now)
	if !res.HasData {
		t.Fatal("HasData = false, want true")
	}
	if !approx(res.TotalSeconds, 12*3600) {
		t.Errorf("TotalSeconds = %v, want %v (clamped to now)", res.TotalSeconds, 12*3600)
	}
	if !approx(res.Uptime, 100) {
		t.Errorf("Uptime = %v, want 100", res.Uptime)
	}
}

func TestDedupExact(t *testing.T) {
	d := day(2026, 6, 4)
	t06 := tm(2026, 6, 4, 6, 0)
	t12 := tm(2026, 6, 4, 12, 0)

	t.Run("drops same-time same-status", func(t *testing.T) {
		events := []ontimerepo.RawEvent{e(d, t06, domain.StatusOn), e(d, t06, domain.StatusOn)}
		if got := dedupExact(events); len(got) != 1 {
			t.Errorf("len = %d, want 1", len(got))
		}
	})

	t.Run("keeps same-time different-status", func(t *testing.T) {
		events := []ontimerepo.RawEvent{
			e(d, t06, domain.StatusOn), e(d, t06, domain.StatusOff),
			e(d, t12, domain.StatusOn),
		}
		if got := dedupExact(events); len(got) != 3 {
			t.Errorf("len = %d, want 3 (conflicting same-timestamp kept)", len(got))
		}
	})
}

func TestToServerStatus(t *testing.T) {

	if s, ok := domain.ToServerStatus(string(domain.StatusOn)); !ok || s != domain.StatusOn {
		t.Errorf("ToServerStatus(ON) = %v,%v", s, ok)
	}

	if s, ok := domain.ToServerStatus(string(domain.StatusOff)); !ok || s != domain.StatusOff {
		t.Errorf("ToServerStatus(OFF) = %v,%v", s, ok)
	}

	for _, raw := range []string{"", "UNKNOWN", "up"} {
		if _, ok := domain.ToServerStatus(raw); ok {
			t.Errorf("ToServerStatus(%q) = known, want unknown", raw)
		}
	}
}
