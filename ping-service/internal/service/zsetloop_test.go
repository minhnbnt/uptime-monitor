package service

import (
	"context"
	"errors"
	"iter"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

func TestSleepCtx(t *testing.T) {
	t.Run("zero duration returns immediately", func(_ *testing.T) {
		sleepCtx(context.Background(), 0)
	})

	t.Run("negative duration returns immediately", func(_ *testing.T) {
		sleepCtx(context.Background(), -1*time.Second)
	})

	t.Run("cancelled context returns immediately", func(_ *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		sleepCtx(ctx, time.Hour)
	})
}

func TestGetSleepDuration(t *testing.T) {
	t.Run("no next task uses default", func(t *testing.T) {
		d := getSleepDuration(scheduler.ScheduledTask{}, false)
		if d != defaultSleepDuration {
			t.Errorf("got %v, want %v", d, defaultSleepDuration)
		}
	})

	t.Run("past due task returns 0", func(t *testing.T) {
		past := time.Now().Add(-time.Hour).UnixMilli()
		d := getSleepDuration(scheduler.ScheduledTask{Score: past}, true)
		if d != 0 {
			t.Errorf("got %v, want 0", d)
		}
	})

	t.Run("future task returns positive duration", func(t *testing.T) {
		future := time.Now().Add(time.Hour).UnixMilli()
		d := getSleepDuration(scheduler.ScheduledTask{Score: future}, true)
		if d <= 0 || d > 2*time.Hour {
			t.Errorf("got %v, want ~1h", d)
		}
	})
}

func TestCalculateNextScore(t *testing.T) {
	t.Run("future score stays unchanged", func(t *testing.T) {
		score := time.Now().Add(time.Hour).UnixMilli()
		got := calculateNextScore(score, 30*time.Second)
		if got != score {
			t.Errorf("got %d, want %d", got, score)
		}
	})

	t.Run("past score catches up", func(t *testing.T) {
		score := int64(0)
		interval := 30 * time.Second
		got := calculateNextScore(score, interval)
		if got <= 0 {
			t.Errorf("got %d, want positive", got)
		}
		if got%(interval.Milliseconds()) != 0 {
			t.Errorf("got %d, want multiple of %dms interval", got, interval.Milliseconds())
		}
	})
}

func TestRunIteration(t *testing.T) {
	ep := &domain.Endpoint{
		Model:        gorm.Model{ID: 1},
		ServerID:     10,
		URL:          "https://example.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	}

	t.Run("empty due list calls GetBatch and handler with empty seq", func(t *testing.T) {
		var getBatchCalled bool
		var handlerCalled bool
		s := &ZsetLoopService{
			logger:           logger.NewMockLogger(),
			schedulerStorage: nil,
			scoreUpdater:     &mockScoreUpdater{},
			endpointProvider: &mockEndpointProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
					getBatchCalled = true
					return make(map[uint]*domain.Endpoint), nil
				},
			},
		}
		err := s.runIteration(context.Background(), nil, func(_ context.Context, _ iter.Seq[*domain.Endpoint]) {
			handlerCalled = true
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !getBatchCalled {
			t.Error("GetBatch should be called even with empty due")
		}
		if !handlerCalled {
			t.Error("DueHandler should be called even with empty due")
		}
	})

	t.Run("happy path", func(t *testing.T) {
		var gotHandlerEndpoints []*domain.Endpoint
		var gotUpdateItems map[uint]int64

		s := &ZsetLoopService{
			logger: logger.NewMockLogger(),
			scoreUpdater: &mockScoreUpdater{
				updateBatchFn: func(_ context.Context, batchItems map[uint]int64) error {
					gotUpdateItems = batchItems
					return nil
				},
			},
			endpointProvider: &mockEndpointProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
					return map[uint]*domain.Endpoint{1: ep}, nil
				},
			},
		}

		due := []scheduler.ScheduledTask{
			{EndpointID: 1, Score: 1000},
		}

		err := s.runIteration(context.Background(), due, func(_ context.Context, endpoints iter.Seq[*domain.Endpoint]) {
			for ep := range endpoints {
				gotHandlerEndpoints = append(gotHandlerEndpoints, ep)
			}
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(gotHandlerEndpoints) != 1 {
			t.Errorf("got %d endpoints, want 1", len(gotHandlerEndpoints))
		}
		if gotHandlerEndpoints[0].ID != 1 {
			t.Errorf("got endpoint %d, want 1", gotHandlerEndpoints[0].ID)
		}
		if gotUpdateItems[1] <= 0 {
			t.Errorf("expected positive score for endpoint 1, got %d", gotUpdateItems[1])
		}
	})

	t.Run("missing endpoint in batch skips reschedule", func(t *testing.T) {
		var updateCalled bool
		s := &ZsetLoopService{
			logger: logger.NewMockLogger(),
			scoreUpdater: &mockScoreUpdater{
				updateBatchFn: func(_ context.Context, _ map[uint]int64) error {
					updateCalled = true
					return nil
				},
			},
			endpointProvider: &mockEndpointProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
					return map[uint]*domain.Endpoint{}, nil
				},
			},
		}

		due := []scheduler.ScheduledTask{
			{EndpointID: 1, Score: 1000},
		}

		err := s.runIteration(context.Background(), due, func(_ context.Context, _ iter.Seq[*domain.Endpoint]) {})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if updateCalled {
			t.Error("UpdateBatch should not be called when no endpoints found")
		}
	})

	t.Run("provider error", func(t *testing.T) {
		wantErr := errors.New("provider error")
		s := &ZsetLoopService{
			logger: logger.NewMockLogger(),
			endpointProvider: &mockEndpointProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
					return nil, wantErr
				},
			},
		}

		err := s.runIteration(context.Background(), []scheduler.ScheduledTask{{EndpointID: 1}}, func(_ context.Context, _ iter.Seq[*domain.Endpoint]) {})
		if err != wantErr {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})

	t.Run("score updater error", func(t *testing.T) {
		wantErr := errors.New("updater error")
		s := &ZsetLoopService{
			logger: logger.NewMockLogger(),
			scoreUpdater: &mockScoreUpdater{
				updateBatchFn: func(_ context.Context, _ map[uint]int64) error {
					return wantErr
				},
			},
			endpointProvider: &mockEndpointProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
					return map[uint]*domain.Endpoint{1: ep}, nil
				},
			},
		}

		err := s.runIteration(context.Background(), []scheduler.ScheduledTask{{EndpointID: 1, Score: 1000}}, func(_ context.Context, _ iter.Seq[*domain.Endpoint]) {})
		if err != wantErr {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})
}
