package service

import (
	"context"
	"errors"
	"testing"
	"time"

	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/logger"
)

func re(serverID uint, startStatus string, status string, t time.Time) ontimerepo.RangeEvent {
	return ontimerepo.RangeEvent{
		ServerID:    serverID,
		StartStatus: startStatus,
		Status:      status,
		Time:        t,
	}
}

func TestCalculateRangeOntime(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := base.Add(time.Hour)

	tests := []struct {
		name   string
		events []ontimerepo.RangeEvent
		from   time.Time
		to     time.Time
		want   float64
	}{
		{
			name:   "no events",
			events: nil,
			from:   base,
			to:     end,
			want:   0,
		},
		{
			name: "single ON covering full range",
			events: []ontimerepo.RangeEvent{
				re(1, "ON", "ON", base),
			},
			from: base,
			to:   end,
			want: 100,
		},
		{
			name: "single OFF covering full range",
			events: []ontimerepo.RangeEvent{
				re(1, "OFF", "OFF", base),
			},
			from: base,
			to:   end,
			want: 0,
		},
		{
			name: "ON then OFF at 30min",
			events: []ontimerepo.RangeEvent{
				re(1, "ON", "OFF", base.Add(30*time.Minute)),
			},
			from: base,
			to:   end,
			want: 50,
		},
		{
			name: "OFF then ON at 30min",
			events: []ontimerepo.RangeEvent{
				re(1, "OFF", "ON", base.Add(30*time.Minute)),
			},
			from: base,
			to:   end,
			want: 50,
		},
		{
			name: "ON→OFF→ON segments",
			events: []ontimerepo.RangeEvent{
				re(1, "ON", "OFF", base.Add(20*time.Minute)),
				re(1, "", "ON", base.Add(40*time.Minute)),
			},
			from: base,
			to:   end,
			want: 100 * 40.0 / 60.0,
		},
		{
			name: "start OFF, ON at 15min, OFF at 45min",
			events: []ontimerepo.RangeEvent{
				re(1, "OFF", "ON", base.Add(15*time.Minute)),
				re(1, "", "OFF", base.Add(45*time.Minute)),
			},
			from: base,
			to:   end,
			want: 100 * 30.0 / 60.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateRangeOntime(tt.events, tt.from, tt.to)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 1e-9 {
				t.Errorf("CalculateRangeOntime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateIntervals(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	events := []ontimerepo.RangeEvent{
		re(1, "ON", "OFF", base.Add(30*time.Minute)),
	}

	intervals := CalculateIntervals(events, base, base.Add(time.Hour), 15*time.Minute)

	if len(intervals) != 4 {
		t.Fatalf("len(intervals) = %d, want 4", len(intervals))
	}

	// First interval: ON for 15min = 100%
	if intervals[0].Uptime != 100 {
		t.Errorf("interval 0 uptime = %f, want 100", intervals[0].Uptime)
	}

	// Second interval: ON for 15min = 100%
	if intervals[1].Uptime != 100 {
		t.Errorf("interval 1 uptime = %f, want 100", intervals[1].Uptime)
	}

	// Third interval: OFF for 15min = 0%
	if intervals[2].Uptime != 0 {
		t.Errorf("interval 2 uptime = %f, want 0", intervals[2].Uptime)
	}

	// Fourth interval: OFF for 15min = 0%
	if intervals[3].Uptime != 0 {
		t.Errorf("interval 3 uptime = %f, want 0", intervals[3].Uptime)
	}
}

func TestBuildTimeline(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := base.Add(time.Hour)

	raw := func(status string, tm time.Time) ontimerepo.RawEvent {
		return ontimerepo.RawEvent{ServerID: 1, Status: status, Time: tm}
	}

	t.Run("sets start status from first event", func(t *testing.T) {
		events := []ontimerepo.RawEvent{raw("ON", base)}
		tl := OntimeCalculator{}.BuildTimeline(events, base, end)
		if tl.StartStatus != "ON" {
			t.Errorf("StartStatus = %q, want ON", tl.StartStatus)
		}
	})

	t.Run("deduplicates events", func(t *testing.T) {
		events := []ontimerepo.RawEvent{
			raw("ON", base),
			raw("ON", base),
			raw("OFF", base.Add(30*time.Minute)),
		}
		tl := OntimeCalculator{}.BuildTimeline(events, base, end)
		if len(tl.Events) != 2 {
			t.Errorf("len(Events) = %d, want 2", len(tl.Events))
		}
	})

	t.Run("empty events", func(t *testing.T) {
		tl := OntimeCalculator{}.BuildTimeline(nil, base, end)
		if tl.StartStatus != "" {
			t.Errorf("StartStatus = %q, want empty", tl.StartStatus)
		}
		if len(tl.Events) != 0 {
			t.Errorf("len(Events) = %d, want 0", len(tl.Events))
		}
	})
}

func TestConvertRangeToRawEvents(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("filters empty status, keeps synthetic start", func(t *testing.T) {
		events := []ontimerepo.RangeEvent{
			re(1, "ON", "", base.Add(10*time.Minute)),
			re(1, "", "", base.Add(20*time.Minute)),
		}
		raw := convertRangeToRawEvents(events, base)
		if len(raw) != 1 {
			t.Fatalf("len(raw) = %d, want 1 (synthetic start)", len(raw))
		}
		if raw[0].Status != "ON" || !raw[0].Time.Equal(base) {
			t.Errorf("raw[0] = {Status: %q, Time: %v}, want {ON, %v}", raw[0].Status, raw[0].Time, base)
		}
	})

	t.Run("adds synthetic event with start status", func(t *testing.T) {
		events := []ontimerepo.RangeEvent{
			re(1, "ON", "OFF", base.Add(30*time.Minute)),
		}
		raw := convertRangeToRawEvents(events, base)
		if len(raw) != 2 {
			t.Fatalf("len(raw) = %d, want 2", len(raw))
		}
		if raw[0].Status != "ON" || !raw[0].Time.Equal(base) {
			t.Errorf("raw[0] = {Status: %q, Time: %v}, want {ON, %v}", raw[0].Status, raw[0].Time, base)
		}
		if raw[1].Status != "OFF" {
			t.Errorf("raw[1].Status = %q, want OFF", raw[1].Status)
		}
	})

	t.Run("uses lowerbound event status", func(t *testing.T) {
		events := []ontimerepo.RangeEvent{
			re(1, "OFF", "ON", base.Add(-10*time.Minute)),
			re(1, "", "OFF", base.Add(30*time.Minute)),
		}
		raw := convertRangeToRawEvents(events, base)
		if len(raw) != 2 {
			t.Fatalf("len(raw) = %d, want 2 (lowerbound + real)", len(raw))
		}
		if raw[0].Status != "ON" {
			t.Errorf("raw[0].Status = %q, want ON (from lowerbound Status)", raw[0].Status)
		}
		if raw[1].Status != "OFF" {
			t.Errorf("raw[1].Status = %q, want OFF", raw[1].Status)
		}
	})

	t.Run("no synthetic when no start status", func(t *testing.T) {
		events := []ontimerepo.RangeEvent{
			re(1, "", "ON", base.Add(30*time.Minute)),
		}
		raw := convertRangeToRawEvents(events, base)
		if len(raw) != 1 {
			t.Fatalf("len(raw) = %d, want 1", len(raw))
		}
		if raw[0].Status != "ON" {
			t.Errorf("raw[0].Status = %q, want ON", raw[0].Status)
		}
	})
}

func TestOntimeRangeService_CalculateUptime(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := base.Add(time.Hour)

	t.Run("success", func(t *testing.T) {
		svc := &OntimeRangeService{
			repo: &mockRangeRepo{
				batchGetOntimeRangeFn: func(_ context.Context, _ []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.RangeEvent, error) {
					return []ontimerepo.RangeEvent{
						re(1, "ON", "ON", base),
					}, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		result, err := svc.CalculateUptime(t.Context(), 1, base, end, 15*time.Minute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ServerID != 1 {
			t.Errorf("ServerID = %d, want 1", result.ServerID)
		}
		if result.Uptime != 100 {
			t.Errorf("Uptime = %f, want 100", result.Uptime)
		}
		if len(result.Intervals) != 4 {
			t.Errorf("len(Intervals) = %d, want 4", len(result.Intervals))
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := &OntimeRangeService{
			repo: &mockRangeRepo{
				batchGetOntimeRangeFn: func(_ context.Context, _ []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.RangeEvent, error) {
					return nil, errors.New("db error")
				},
			},
			logger: logger.NewMockLogger(),
		}

		_, err := svc.CalculateUptime(t.Context(), 1, base, end, 15*time.Minute)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
