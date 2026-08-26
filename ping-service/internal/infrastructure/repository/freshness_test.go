package repository

import (
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
)

func startFreshnessStore(tb testing.TB) (*FreshnessStore, *redis.Client) {
	tb.Helper()

	testcontainers.SkipIfShort(tb)
	ctx := tb.Context()

	container, addr := testcontainers.StartRedisAddr(ctx)
	tb.Cleanup(func() { _ = container.Terminate(ctx) })

	client := testcontainers.NewTestRedis(tb, addr)
	return NewFreshnessStore(client, 1), client
}

func TestFreshnessStore(t *testing.T) {

	const key = pushFreshnessPrefix + ":0"

	t.Run("touch sets deadline in the shard key", func(t *testing.T) {

		store, storeClient := startFreshnessStore(t)

		lease := 90 * time.Second
		before := time.Now()
		if err := store.Touch(t.Context(), 7, lease); err != nil {
			t.Fatalf("Touch: %v", err)
		}

		score, err := storeClient.ZScore(t.Context(), key, "7").Result()
		if err != nil {
			t.Fatalf("ZScore: %v", err)
		}

		wantLow := float64(before.Add(lease).UnixMilli())
		if score < wantLow {
			t.Errorf("score = %f, want >= %f", score, wantLow)
		}
	})

	t.Run("remove deletes the entry", func(t *testing.T) {

		store, storeClient := startFreshnessStore(t)

		if err := store.Touch(t.Context(), 7, time.Minute); err != nil {
			t.Fatalf("Touch: %v", err)
		}
		if err := store.Remove(t.Context(), 7); err != nil {
			t.Fatalf("Remove: %v", err)
		}

		_, err := storeClient.ZScore(t.Context(), key, "7").Result()
		if !errors.Is(err, redis.Nil) {
			t.Errorf("err = %v, want redis.Nil after remove", err)
		}
	})

	t.Run("claim returns only overdue entries and locks them", func(t *testing.T) {

		store, storeClient := startFreshnessStore(t)
		ctx := t.Context()

		now := time.Now()
		past := now.Add(-time.Minute).UnixMilli()
		future := now.Add(time.Minute).UnixMilli()

		err := storeClient.ZAdd(ctx, key,
			redis.Z{Member: "9", Score: float64(past)},
			redis.Z{Member: "8", Score: float64(future)},
		).Err()
		if err != nil {
			t.Fatalf("seed ZAdd: %v", err)
		}

		due, next, hasNext, err := store.ClaimOverdue(ctx, 0, 10)
		if err != nil {
			t.Fatalf("ClaimOverdue: %v", err)
		}
		if len(due) != 1 || due[0].EndpointID != 9 {
			t.Fatalf("due = %+v, want only endpoint 9", due)
		}
		if !hasNext || next.EndpointID != 8 {
			t.Errorf("next = %+v hasNext = %v, want endpoint 8 true", next, hasNext)
		}

		bumped, err := storeClient.ZScore(ctx, key, "9").Result()
		if err != nil {
			t.Fatalf("ZScore after claim: %v", err)
		}
		wantLow := float64(now.UnixMilli())
		wantHigh := float64(now.Add(11 * time.Second).UnixMilli())
		if bumped < wantLow || bumped > wantHigh {
			t.Errorf("bumped score = %f, want in [%f, %f]", bumped, wantLow, wantHigh)
		}

		due, _, _, err = store.ClaimOverdue(ctx, 0, 10)
		if err != nil {
			t.Fatalf("second ClaimOverdue: %v", err)
		}
		if len(due) != 0 {
			t.Errorf("second claim must be locked, got %+v", due)
		}
	})
}
