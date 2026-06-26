package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

func newSchedulerRepo(tb testing.TB) *ZSetScheduleRepository {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	return &ZSetScheduleRepository{client: testRedis}
}

func addTask(tb testing.TB, id uint, score int64) {
	tb.Helper()
	err := testRedis.ZAdd(tb.Context(), schedulerQueueKey, redis.Z{
		Score:  float64(score),
		Member: fmt.Sprint(id),
	}).Err()
	if err != nil {
		tb.Fatalf("seed ZAdd: %v", err)
	}
}

func addMetaCache(tb testing.TB, id uint) {
	tb.Helper()
	err := testRedis.HSet(tb.Context(), metaCacheKey(id), "url", "https://example.com").Err()
	if err != nil {
		tb.Fatalf("seed HSet: %v", err)
	}
}

func TestIntegration_Register(t *testing.T) {
	cleanRedis(t)

	repo := newSchedulerRepo(t)
	ep := &domain.Endpoint{
		URL:      "https://example.com",
		Method:   "GET",
		Interval: 30 * time.Second,
	}
	ep.ID = 42

	err := repo.Register(t.Context(), ep)
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}

	score, err := testRedis.ZScore(t.Context(), schedulerQueueKey, "42").Result()
	if err != nil {
		t.Fatalf("ZScore error: %v", err)
	}
	if score <= 0 {
		t.Errorf("score = %f, want > 0", score)
	}
}

func TestIntegration_Unregister(t *testing.T) {
	cleanRedis(t)

	repo := newSchedulerRepo(t)
	endpointID := uint(7)

	addTask(t, endpointID, time.Now().Add(time.Hour).UnixMilli())
	addMetaCache(t, endpointID)

	err := repo.Unregister(t.Context(), endpointID)
	if err != nil {
		t.Fatalf("Unregister error: %v", err)
	}

	exists, err := testRedis.ZScore(t.Context(), schedulerQueueKey, "7").Result()
	if err != redis.Nil {
		t.Fatalf("expected member to be removed, ZScore: %v (score=%f)", err, exists)
	}

	n, err := testRedis.Exists(t.Context(), metaCacheKey(endpointID)).Result()
	if err != nil {
		t.Fatalf("Exists error: %v", err)
	}
	if n > 0 {
		t.Error("meta cache key was not deleted")
	}
}

func TestIntegration_ClaimDueTasks_NoDue(t *testing.T) {
	cleanRedis(t)

	repo := newSchedulerRepo(t)
	futureScore := time.Now().Add(time.Hour).UnixMilli()
	addTask(t, 1, futureScore)

	due, next, hasNext, err := repo.ClaimDueTasks(t.Context(), 10)
	if err != nil {
		t.Fatalf("ClaimDueTasks error: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("due = %v, want empty", due)
	}
	if !hasNext {
		t.Fatal("expected hasNext=true")
	}
	if next.EndpointID != 1 {
		t.Errorf("next.EndpointID = %d, want 1", next.EndpointID)
	}
	if next.Score != futureScore {
		t.Errorf("next.Score = %d, want %d", next.Score, futureScore)
	}
}

func TestIntegration_ClaimDueTasks_Due(t *testing.T) {
	cleanRedis(t)

	repo := newSchedulerRepo(t)
	pastScore := time.Now().Add(-time.Minute).UnixMilli()
	addTask(t, 1, pastScore)

	due, _, hasNext, err := repo.ClaimDueTasks(t.Context(), 10)
	if err != nil {
		t.Fatalf("ClaimDueTasks error: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("due length = %d, want 1", len(due))
	}
	if due[0].EndpointID != 1 {
		t.Errorf("due[0].EndpointID = %d, want 1", due[0].EndpointID)
	}

	locked, err := testRedis.ZScore(t.Context(), schedulerQueueKey, "1").Result()
	if err != nil {
		t.Fatalf("ZScore after claim: %v", err)
	}
	now := time.Now().UnixMilli()
	if locked < float64(now) || locked > float64(now+claimLock.Milliseconds()+1000) {
		t.Errorf("locked score = %f, want between now(%d) and now+10s(%d)", locked, now, now+claimLock.Milliseconds())
	}

	if hasNext {
		t.Errorf("hasNext = true, want false (only one task)")
	}
}

func TestIntegration_ClaimDueTasks_WithLimit(t *testing.T) {
	cleanRedis(t)

	repo := newSchedulerRepo(t)
	pastScore := time.Now().Add(-time.Minute).UnixMilli()
	addTask(t, 1, pastScore)
	addTask(t, 2, pastScore)
	addTask(t, 3, pastScore)

	due, _, _, err := repo.ClaimDueTasks(t.Context(), 2)
	if err != nil {
		t.Fatalf("ClaimDueTasks error: %v", err)
	}
	if len(due) != 2 {
		t.Errorf("due length = %d, want 2", len(due))
	}
}

func TestIntegration_ClaimDueTasks_ZeroLimit(t *testing.T) {
	cleanRedis(t)

	repo := newSchedulerRepo(t)

	due, _, hasNext, err := repo.ClaimDueTasks(t.Context(), 0)
	if err != nil {
		t.Fatalf("ClaimDueTasks error: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("due = %v, want empty", due)
	}
	if hasNext {
		t.Error("expected hasNext=false")
	}
}
