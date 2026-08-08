package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis/scheduler"
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
	sv := &domain.Server{
		ID:        1,
		Namespace: "default",
		Kind:      "Pod",
		ObjectID:  "web-app",
		Interval:  30 * time.Second,
	}

	t.Run("empty due list calls handler with empty tasks", func(t *testing.T) {
		var handlerCalled bool
		s := &ZsetLoopService{
			logger:           logger.NewMockLogger(),
			schedulerStorage: nil,
			serverProvider: &mockServerProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Server, error) {
					return make(map[uint]*domain.Server), nil
				},
			},
		}
		err := s.runIteration(t.Context(), nil, func(_ context.Context, _ []PingTask) {
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
		var gotServers []*dto.Server

		s := &ZsetLoopService{
			logger: logger.NewMockLogger(),
			serverProvider: &mockServerProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Server, error) {
					return map[uint]*domain.Server{1: sv}, nil
				},
			},
		}

		due := []scheduler.ScheduledTask{
			{ServerID: 1},
		}

		err := s.runIteration(t.Context(), due, func(_ context.Context, tasks []PingTask) {
			for _, t := range tasks {
				gotServers = append(gotServers, t.Server)
			}
		})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if len(gotServers) != 1 {
			t.Errorf("got %d servers, want 1", len(gotServers))
		}
		if gotServers[0].ID != 1 {
			t.Errorf("got server %d, want 1", gotServers[0].ID)
		}
	})

	t.Run("provider error", func(t *testing.T) {
		wantErr := errors.New("provider error")
		s := &ZsetLoopService{
			logger: logger.NewMockLogger(),
			serverProvider: &mockServerProvider{
				getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Server, error) {
					return nil, wantErr
				},
			},
		}

		err := s.runIteration(t.Context(), []scheduler.ScheduledTask{{ServerID: 1}}, func(_ context.Context, _ []PingTask) {})
		if err != wantErr {
			t.Errorf("got %v, want %v", err, wantErr)
		}
	})
}
