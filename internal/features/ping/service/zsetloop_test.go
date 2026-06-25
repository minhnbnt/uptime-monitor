package service

import (
	"context"
	"testing"
	"time"

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
