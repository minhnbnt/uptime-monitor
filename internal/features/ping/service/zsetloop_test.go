package service

import (
	"context"
	"io"
	"iter"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/ping/scheduler"
)

func TestGetSleepDuration(t *testing.T) {
	now := time.Now()

	t.Run("no next task returns default sleep", func(t *testing.T) {
		d := getSleepDuration(scheduler.ScheduledTask{}, false)
		if d != defaultSleepDuration {
			t.Errorf("got %v, want %v", d, defaultSleepDuration)
		}
	})

	t.Run("past due task returns 0", func(t *testing.T) {
		next := scheduler.ScheduledTask{
			EndpointID: 1,
			Score:      now.Add(-1 * time.Minute).UnixMilli(),
		}
		d := getSleepDuration(next, true)
		if d != 0 {
			t.Errorf("got %v, want 0", d)
		}
	})

	t.Run("future task returns duration until it", func(t *testing.T) {
		future := now.Add(10 * time.Second)
		next := scheduler.ScheduledTask{
			EndpointID: 1,
			Score:      future.UnixMilli(),
		}
		d := getSleepDuration(next, true)
		if d <= 0 || d > 11*time.Second {
			t.Errorf("got %v, expected around 10s", d)
		}
	})
}

func TestSleepCtx(t *testing.T) {
	t.Run("non-positive duration returns immediately", func(t *testing.T) {
		start := time.Now()
		sleepCtx(context.Background(), 0)
		sleepCtx(context.Background(), -1*time.Second)
		if time.Since(start) > 100*time.Millisecond {
			t.Error("sleepCtx should return immediately for non-positive duration")
		}
	})

	t.Run("cancelled context returns early", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		start := time.Now()
		sleepCtx(ctx, 5*time.Second)
		if time.Since(start) > 100*time.Millisecond {
			t.Error("sleepCtx should return early on cancelled context")
		}
	})

	t.Run("positive duration sleeps", func(t *testing.T) {
		start := time.Now()
		sleepCtx(context.Background(), 10*time.Millisecond)
		if time.Since(start) < 5*time.Millisecond {
			t.Error("sleepCtx should block for at least the duration")
		}
	})
}

func TestRunIteration_EmptyDue(t *testing.T) {
	var handlerCalled bool
	var updatesReceived map[uint]int64
	svc := &LoopService{
		endpointProvider: &mockEndpointProvider{
			getBatchFn: func(_ context.Context, ids []uint) (map[uint]*domain.Endpoint, error) {
				if len(ids) != 0 {
					t.Errorf("expected empty ids, got %v", ids)
				}
				return map[uint]*domain.Endpoint{}, nil
			},
		},
		scoreUpdater: &mockScoreUpdater{
			updateBatchFn: func(_ context.Context, items map[uint]int64) error {
				updatesReceived = items
				return nil
			},
		},
	}

	err := svc.runIteration(t.Context(), []scheduler.ScheduledTask{}, func(_ context.Context, tasks iter.Seq[*domain.Endpoint]) {
		handlerCalled = true
		for range tasks {
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Error("handler was not called")
	}
	if updatesReceived != nil {
		t.Errorf("expected no updates, got %v", updatesReceived)
	}
}

func TestRunIteration_SingleItem(t *testing.T) {
	now := time.Now()
	due := []scheduler.ScheduledTask{
		{EndpointID: 1, Score: now.UnixMilli()},
	}
	var handlerEndpoints []*domain.Endpoint
	var updatesReceived map[uint]int64
	svc := &LoopService{
		endpointProvider: &mockEndpointProvider{
			getBatchFn: func(_ context.Context, ids []uint) (map[uint]*domain.Endpoint, error) {
				return map[uint]*domain.Endpoint{
					1: {Interval: 10 * time.Second},
				}, nil
			},
		},
		scoreUpdater: &mockScoreUpdater{
			updateBatchFn: func(_ context.Context, items map[uint]int64) error {
				updatesReceived = items
				return nil
			},
		},
	}

	err := svc.runIteration(t.Context(), due, func(_ context.Context, tasks iter.Seq[*domain.Endpoint]) {
		for t := range tasks {
			handlerEndpoints = append(handlerEndpoints, t)
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handlerEndpoints) != 1 {
		t.Fatalf("handler got %d endpoints, want 1", len(handlerEndpoints))
	}
	if updatesReceived[1] != due[0].Score+10000 {
		t.Errorf("update score = %d, want %d", updatesReceived[1], due[0].Score+10000)
	}
}

func TestRunIteration_MultipleItems(t *testing.T) {
	now := time.Now()
	due := []scheduler.ScheduledTask{
		{EndpointID: 1, Score: now.UnixMilli()},
		{EndpointID: 2, Score: now.UnixMilli() + 5000},
	}
	var handlerEndpoints []*domain.Endpoint
	var updatesReceived map[uint]int64
	svc := &LoopService{
		endpointProvider: &mockEndpointProvider{
			getBatchFn: func(_ context.Context, ids []uint) (map[uint]*domain.Endpoint, error) {
				return map[uint]*domain.Endpoint{
					1: {Interval: 10 * time.Second},
					2: {Interval: 30 * time.Second},
				}, nil
			},
		},
		scoreUpdater: &mockScoreUpdater{
			updateBatchFn: func(_ context.Context, items map[uint]int64) error {
				updatesReceived = items
				return nil
			},
		},
	}

	err := svc.runIteration(t.Context(), due, func(_ context.Context, tasks iter.Seq[*domain.Endpoint]) {
		for t := range tasks {
			handlerEndpoints = append(handlerEndpoints, t)
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(handlerEndpoints) != 2 {
		t.Fatalf("handler got %d endpoints, want 2", len(handlerEndpoints))
	}
	expected1 := time.Now().UnixMilli() + 10000
	expected2 := time.Now().UnixMilli() + 30000
	got1 := updatesReceived[1]
	got2 := updatesReceived[2]
	if abs(got1-expected1) > 100 {
		t.Errorf("update[1] score = %d, expected around %d (diff %d)", got1, expected1, abs(got1-expected1))
	}
	if abs(got2-expected2) > 100 {
		t.Errorf("update[2] score = %d, expected around %d (diff %d)", got2, expected2, abs(got2-expected2))
	}
}

func TestRunIteration_ProviderError(t *testing.T) {
	svc := &LoopService{
		endpointProvider: &mockEndpointProvider{
			getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
				return nil, io.ErrUnexpectedEOF
			},
		},
	}
	err := svc.runIteration(t.Context(), []scheduler.ScheduledTask{{EndpointID: 1}}, func(_ context.Context, _ iter.Seq[*domain.Endpoint]) {})
	if err == nil {
		t.Fatal("expected error")
	}
}

func abs(n int64) int64 {
	return max(n, -n)
}

func TestRunIteration_UpdaterError(t *testing.T) {
	now := time.Now()
	svc := &LoopService{
		endpointProvider: &mockEndpointProvider{
			getBatchFn: func(_ context.Context, _ []uint) (map[uint]*domain.Endpoint, error) {
				return map[uint]*domain.Endpoint{
					1: {Interval: 10 * time.Second},
				}, nil
			},
		},
		scoreUpdater: &mockScoreUpdater{
			updateBatchFn: func(_ context.Context, _ map[uint]int64) error {
				return io.ErrClosedPipe
			},
		},
	}
	err := svc.runIteration(t.Context(), []scheduler.ScheduledTask{{EndpointID: 1, Score: now.UnixMilli()}}, func(_ context.Context, _ iter.Seq[*domain.Endpoint]) {})
	if err == nil {
		t.Fatal("expected error")
	}
}
