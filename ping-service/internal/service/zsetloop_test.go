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
	t.Run("zero duration returns immediately", func(t *testing.T) {
		sleepCtx(t.Context(), 0)
	})

	t.Run("negative duration returns immediately", func(t *testing.T) {
		sleepCtx(t.Context(), -1*time.Second)
	})

	t.Run("cancelled context returns immediately", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
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

func TestRunIteration(t *testing.T) {
	ep := &domain.Endpoint{
		Model:        gorm.Model{ID: 1},
		ServerID:     10,
		URL:          "https://example.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	}

	t.Run("empty due list calls handler with empty seq", func(t *testing.T) {
		var handlerCalled bool
		s := &ZsetLoopService{
			logger:           logger.NewMockLogger(),
			schedulerStorage: nil,
			endpointProvider: &mockEndpointProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
					return make(map[uint]*domain.Endpoint), nil
				},
			},
		}
		err := s.runIteration(t.Context(), nil, func(_ context.Context, _ iter.Seq[*PingTask]) {
			handlerCalled = true
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !handlerCalled {
			t.Error("DueHandler should be called even with empty due")
		}
	})

	t.Run("happy path", func(t *testing.T) {
		var gotTasks []*PingTask

		s := &ZsetLoopService{
			logger: logger.NewMockLogger(),
			endpointProvider: &mockEndpointProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
					return map[uint]*domain.Endpoint{1: ep}, nil
				},
			},
		}

		due := []scheduler.ScheduledTask{
			{EndpointID: 1, Score: 1000},
		}

		err := s.runIteration(t.Context(), due, func(_ context.Context, tasks iter.Seq[*PingTask]) {
			for task := range tasks {
				gotTasks = append(gotTasks, task)
			}
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(gotTasks) != 1 {
			t.Errorf("got %d tasks, want 1", len(gotTasks))
		}
		if gotTasks[0].Endpoint.ID != 1 {
			t.Errorf("got endpoint %d, want 1", gotTasks[0].Endpoint.ID)
		}
		if gotTasks[0].Score != 1000 {
			t.Errorf("got score %d, want 1000", gotTasks[0].Score)
		}
	})

	t.Run("missing endpoint in batch sets nil", func(t *testing.T) {
		var gotTasks []*PingTask
		s := &ZsetLoopService{
			logger: logger.NewMockLogger(),
			endpointProvider: &mockEndpointProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
					return map[uint]*domain.Endpoint{}, nil
				},
			},
		}

		due := []scheduler.ScheduledTask{
			{EndpointID: 1, Score: 1000},
		}

		err := s.runIteration(t.Context(), due, func(_ context.Context, tasks iter.Seq[*PingTask]) {
			for task := range tasks {
				gotTasks = append(gotTasks, task)
			}
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if gotTasks[0].Endpoint != nil {
			t.Error("expected nil endpoint for missing batch entry")
		}
		if gotTasks[0].Score != 1000 {
			t.Errorf("got score %d, want 1000", gotTasks[0].Score)
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

		err := s.runIteration(t.Context(), []scheduler.ScheduledTask{{EndpointID: 1}}, func(_ context.Context, _ iter.Seq[*PingTask]) {})
		if err != wantErr {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})
}
