package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/logger"
)

func re(serverID uint, anchorTime time.Time, status string, t time.Time) ontimerepo.ServerEvent {
	return ontimerepo.ServerEvent{
		ServerID: serverID,
		Event:    ontimerepo.Event{AnchorTime: anchorTime, Status: status, Time: t, Src: "test"},
	}
}

func reEvent(anchorTime time.Time, status string, t time.Time) ontimerepo.Event {
	return ontimerepo.Event{AnchorTime: anchorTime, Status: status, Time: t, Src: "test"}
}

func TestCalculateRangeOntime(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := base.Add(time.Hour)

	tests := []struct {
		name   string
		events []ontimerepo.Event
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
			events: []ontimerepo.Event{
				reEvent(base, "ON", base),
			},
			from: base,
			to:   end,
			want: 100,
		},
		{
			name: "single OFF covering full range",
			events: []ontimerepo.Event{
				reEvent(base, "OFF", base),
			},
			from: base,
			to:   end,
			want: 0,
		},
		{
			name: "ON then OFF at 30min",
			events: []ontimerepo.Event{
				reEvent(base, "ON", base),
				reEvent(base, "OFF", base.Add(30*time.Minute)),
			},
			from: base,
			to:   end,
			want: 50,
		},
		{
			name: "OFF then ON at 30min",
			events: []ontimerepo.Event{
				reEvent(base, "OFF", base),
				reEvent(base, "ON", base.Add(30*time.Minute)),
			},
			from: base,
			to:   end,
			want: 50,
		},
		{
			name: "ON→OFF→ON segments",
			events: []ontimerepo.Event{
				reEvent(base, "ON", base),
				reEvent(base, "OFF", base.Add(20*time.Minute)),
				reEvent(base, "ON", base.Add(40*time.Minute)),
			},
			from: base,
			to:   end,
			want: 100 * 40.0 / 60.0,
		},
		{
			name: "start OFF, ON at 15min, OFF at 45min",
			events: []ontimerepo.Event{
				reEvent(base, "OFF", base),
				reEvent(base, "ON", base.Add(15*time.Minute)),
				reEvent(base, "OFF", base.Add(45*time.Minute)),
			},
			from: base,
			to:   end,
			want: 100 * 30.0 / 60.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := OntimeCalculator{}.CalculateOntime(tt.events, tt.from, tt.to)
			diff := res.Uptime - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 1e-9 {
				t.Errorf("CalculateRangeOntime() = %v, want %v", res.Uptime, tt.want)
			}
			// Every case here carries known status (ON/OFF boundary rows), so
			// HasData must be true and Partial false (known at the boundary).
			if tt.name != "no events" && !res.HasData {
				t.Errorf("HasData = false, want true for %q", tt.name)
			}
		})
	}
}

func TestCalculateRangeOntime_NoData(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	res := OntimeCalculator{}.CalculateOntime(nil, base, base.Add(time.Hour))
	if res.HasData {
		t.Error("HasData = true, want false")
	}
	if res.Uptime != 0 {
		t.Errorf("Uptime = %v, want 0 (no data, not 0%% uptime)", res.Uptime)
	}
	if res.TotalSeconds != 3600 {
		t.Errorf("TotalSeconds = %v, want 3600", res.TotalSeconds)
	}
}

func TestMergeIntervals_NoDataVsZero(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	iv := func(i int, uptime float64, hasData bool) dto.IntervalResult {
		return dto.IntervalResult{
			From:    base.Add(time.Duration(i) * time.Hour).Format(time.RFC3339),
			To:      base.Add(time.Duration(i+1) * time.Hour).Format(time.RFC3339),
			Uptime:  uptime,
			HasData: hasData,
		}
	}

	t.Run("no-data adjacent to 0% is not merged", func(t *testing.T) {
		in := []dto.IntervalResult{
			iv(0, 0, true),  // 0% uptime with data
			iv(1, 0, false), // no data
			iv(2, 0, false), // no data
		}
		got := mergeIntervals(in)
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2 (0%% bucket kept separate from no-data)", len(got))
		}
		if !got[0].HasData || got[1].HasData {
			t.Errorf("wrong HasData split: %+v", got)
		}
	})

	t.Run("adjacent 0% buckets merge", func(t *testing.T) {
		in := []dto.IntervalResult{
			iv(0, 0, true),
			iv(1, 0, true),
		}
		if got := mergeIntervals(in); len(got) != 1 {
			t.Errorf("len = %d, want 1 (0%% buckets merge)", len(got))
		}
	})
}

func TestCalculateIntervals(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	events := []ontimerepo.Event{
		reEvent(base, "ON", base),
		reEvent(base, "OFF", base.Add(30*time.Minute)),
	}

	intervals := OntimeCalculator{}.CalculateIntervals(events, utils.SplitIntervals(base, base.Add(time.Hour), 15*time.Minute))

	if len(intervals) != 4 {
		t.Fatalf("len(intervals) = %d, want 4", len(intervals))
	}

	// Every interval is anchored by the ON boundary event, so each carries data.
	for i, iv := range intervals {
		if !iv.HasData {
			t.Errorf("interval %d HasData = false, want true", i)
		}
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

	raw := func(status string, tm time.Time) ontimerepo.Event {
		return ontimerepo.Event{AnchorTime: base, Status: status, Time: tm, Src: "test"}
	}

	t.Run("sets start status from first event", func(t *testing.T) {
		events := []ontimerepo.Event{raw("ON", base)}
		tl := OntimeCalculator{}.newTimeline(events, base, end)
		if tl.StartStatus != "ON" {
			t.Errorf("StartStatus = %q, want ON", tl.StartStatus)
		}
	})

	t.Run("deduplicates events", func(t *testing.T) {
		events := []ontimerepo.Event{
			raw("ON", base),
			raw("ON", base),
			raw("OFF", base.Add(30*time.Minute)),
		}
		tl := OntimeCalculator{}.newTimeline(events, base, end)
		if len(tl.Events) != 2 {
			t.Errorf("len(Events) = %d, want 2", len(tl.Events))
		}
	})

	t.Run("empty events", func(t *testing.T) {
		tl := OntimeCalculator{}.newTimeline(nil, base, end)
		if tl.StartStatus != "" {
			t.Errorf("StartStatus = %q, want empty", tl.StartStatus)
		}
		if len(tl.Events) != 0 {
			t.Errorf("len(Events) = %d, want 0", len(tl.Events))
		}
	})
}

func TestOntimeRangeService_CalculateUptime(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := base.Add(time.Hour)

	t.Run("success", func(t *testing.T) {
		svc := &OntimeRangeService{
			repo: &mockRangeRepo{
				batchGetOntimeRangeFn: func(_ context.Context, _ []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.ServerEvent, error) {
					return []ontimerepo.ServerEvent{
						re(1, base, "ON", base),
					}, nil
				},
			},
			ownerRepo: &mockOwnerRepo{
				getByServerIDFn: func(_ context.Context, _ uint) (*domain.ServerOwner, error) {
					return &domain.ServerOwner{ServerID: 1, UserID: uuid.UUID{}}, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		result, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{ServerID: 1, UserID: uuid.UUID{}, From: base, To: end, Resolution: 15 * time.Minute})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ServerID != 1 {
			t.Errorf("ServerID = %d, want 1", result.ServerID)
		}
		if result.Uptime != 100 {
			t.Errorf("Uptime = %f, want 100", result.Uptime)
		}
		if len(result.Intervals) != 1 {
			t.Errorf("len(Intervals) = %d, want 1 (merged all 100%% intervals)", len(result.Intervals))
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := &OntimeRangeService{
			repo: &mockRangeRepo{
				batchGetOntimeRangeFn: func(_ context.Context, _ []ontimerepo.BatchGetOntimeRangeRequest) ([]ontimerepo.ServerEvent, error) {
					return nil, errors.New("db error")
				},
			},
			ownerRepo: &mockOwnerRepo{
				getByServerIDFn: func(_ context.Context, _ uint) (*domain.ServerOwner, error) {
					return &domain.ServerOwner{ServerID: 1, UserID: uuid.UUID{}}, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		_, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{ServerID: 1, UserID: uuid.UUID{}, From: base, To: end, Resolution: 15 * time.Minute})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("forbidden when not owner", func(t *testing.T) {
		svc := &OntimeRangeService{
			repo: &mockRangeRepo{},
			ownerRepo: &mockOwnerRepo{
				getByServerIDFn: func(_ context.Context, _ uint) (*domain.ServerOwner, error) {
					return &domain.ServerOwner{ServerID: 1, UserID: uuid.New()}, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		_, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{ServerID: 1, UserID: uuid.UUID{}, From: base, To: end, Resolution: 15 * time.Minute})
		if !errors.Is(err, apperrors.ErrForbidden) {
			t.Fatalf("err = %v, want ErrForbidden", err)
		}
	})

	t.Run("not found when no owner record", func(t *testing.T) {
		svc := &OntimeRangeService{
			repo: &mockRangeRepo{},
			ownerRepo: &mockOwnerRepo{
				getByServerIDFn: func(_ context.Context, _ uint) (*domain.ServerOwner, error) {
					return nil, apperrors.ErrNotFound
				},
			},
			logger: logger.NewMockLogger(),
		}

		_, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{ServerID: 1, UserID: uuid.UUID{}, From: base, To: end, Resolution: 15 * time.Minute})
		if !errors.Is(err, apperrors.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})
}
