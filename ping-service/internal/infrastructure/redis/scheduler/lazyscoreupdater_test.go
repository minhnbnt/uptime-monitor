package scheduler

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
)

func newLazyScoreUpdater(tb testing.TB, client *redis.Client, batchSize int) *LazyScoreUpdater {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	logger := slog.New(slog.DiscardHandler)
	updater := NewScoreUpdater(client, 1)
	return NewLazyScoreUpdater(updater, batchSize, logger)
}

func waitForScore(ctx context.Context, tb testing.TB, client *redis.Client, member string, want int64) {
	tb.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		score, err := client.ZScore(ctx, shardKey(0), member).Result()
		if err == nil && int64(score) == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	tb.Fatalf("score for %s never reached %d", member, want)
}

func TestLazyScoreUpdaterFlushesFullBatch(t *testing.T) {
	testcontainers.SkipIfShort(t)
	client := testcontainers.NewTestRedis(t, testRedisAddr)
	lazy := newLazyScoreUpdater(t, client, 2)
	ctx := context.Background()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go lazy.Run(runCtx)

	if err := lazy.Update(ctx, 1, 100); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := lazy.Update(ctx, 2, 200); err != nil {
		t.Fatalf("Update: %v", err)
	}

	waitForScore(ctx, t, client, "1", 100)
	waitForScore(ctx, t, client, "2", 200)
}

func TestLazyScoreUpdaterFlushesOnTimeout(t *testing.T) {
	testcontainers.SkipIfShort(t)
	client := testcontainers.NewTestRedis(t, testRedisAddr)
	lazy := newLazyScoreUpdater(t, client, 50)
	ctx := context.Background()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	go lazy.Run(runCtx)

	if err := lazy.Update(ctx, 1, 300); err != nil {
		t.Fatalf("Update: %v", err)
	}

	waitForScore(ctx, t, client, "1", 300)
}
