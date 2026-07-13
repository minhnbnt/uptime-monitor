package scheduler

import (
	"strconv"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

func newScoreUpdater(tb testing.TB) *ScoreUpdater {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	return &ScoreUpdater{client: testRedis}
}

func TestIntegration_UpdateBatch_Empty(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	u := newScoreUpdater(t)
	err := u.UpdateBatch(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = u.UpdateBatch(t.Context(), map[uint]int64{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_UpdateBatch_SingleItem(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	u := newScoreUpdater(t)
	err := u.UpdateBatch(t.Context(), map[uint]int64{42: 1000})
	if err != nil {
		t.Fatalf("UpdateBatch error: %v", err)
	}

	score, err := testRedis.ZScore(t.Context(), schedulerQueueKey, "42").Result()
	if err != nil {
		t.Fatalf("ZScore error: %v", err)
	}
	if score != 1000 {
		t.Errorf("score = %f, want 1000", score)
	}
}

func TestIntegration_UpdateBatch_MultipleItems(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	u := newScoreUpdater(t)
	items := map[uint]int64{
		1: 100,
		2: 200,
		3: 300,
	}

	err := u.UpdateBatch(t.Context(), items)
	if err != nil {
		t.Fatalf("UpdateBatch error: %v", err)
	}

	results, err := testRedis.ZRangeWithScores(t.Context(), schedulerQueueKey, 0, -1).Result()
	if err != nil {
		t.Fatalf("ZRangeWithScores error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	for _, z := range results {
		member, _ := strconv.ParseUint(z.Member.(string), 10, 64)
		id := uint(member)
		expectedScore, ok := items[id]
		if !ok {
			t.Errorf("unexpected member %d", id)
			continue
		}
		if z.Score != float64(expectedScore) {
			t.Errorf("member %d: score = %f, want %d", id, z.Score, expectedScore)
		}
	}
}

func TestIntegration_UpdateBatch_UpdatesExisting(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testRedis = testcontainers.NewTestRedis(t, testRedisAddr)

	u := newScoreUpdater(t)
	if err := u.UpdateBatch(t.Context(), map[uint]int64{1: 500}); err != nil {
		t.Fatalf("first UpdateBatch error: %v", err)
	}
	if err := u.UpdateBatch(t.Context(), map[uint]int64{1: 999}); err != nil {
		t.Fatalf("second UpdateBatch error: %v", err)
	}

	score, err := testRedis.ZScore(t.Context(), schedulerQueueKey, "1").Result()
	if err != nil {
		t.Fatalf("ZScore error: %v", err)
	}
	if score != 999 {
		t.Errorf("score = %f, want 999", score)
	}
}
