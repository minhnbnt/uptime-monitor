package scheduler

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
	"gorm.io/gorm"
)

var testRedisAddr string

func TestMain(m *testing.M) {
	flag.Parse()

	if !testing.Short() {
		ctx := context.Background()
		container, addr := testcontainers.StartRedisAddr(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testRedisAddr = addr
	}

	os.Exit(m.Run())
}

func newRepo(tb testing.TB) *ZSetScheduleRepository {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	client := testcontainers.NewTestRedis(tb, testRedisAddr)
	return NewZSetScheduleRepository(client)
}

func newScoreUpdater(tb testing.TB, client *redis.Client) *ScoreUpdater {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	return &ScoreUpdater{client: client}
}

func TestRegisterBatch(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	endpoints := []domain.Endpoint{
		{Model: gorm.Model{ID: 1}, Interval: 30 * time.Second},
		{Model: gorm.Model{ID: 2}, Interval: 60 * time.Second},
	}

	err := repo.RegisterBatch(ctx, endpoints)
	if err != nil {
		t.Fatalf("RegisterBatch: %v", err)
	}

	// Verify members are in the ZSET
	client := repo.client
	count, err := client.ZCard(ctx, schedulerQueueKey).Result()
	if err != nil {
		t.Fatalf("ZCard: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 members, got %d", count)
	}
}

func TestRegisterUnregister(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	endpoints := []domain.Endpoint{
		{Model: gorm.Model{ID: 10}, Interval: 30 * time.Second},
	}
	err := repo.RegisterBatch(ctx, endpoints)
	if err != nil {
		t.Fatalf("RegisterBatch: %v", err)
	}

	// Verify registered
	exists, err := repo.client.ZScore(ctx, schedulerQueueKey, "10").Result()
	if err != nil {
		t.Fatalf("ZScore after register: %v", err)
	}
	if exists <= 0 {
		t.Errorf("expected positive score, got %f", exists)
	}

	// Unregister
	err = repo.Unregister(ctx, 10)
	if err != nil {
		t.Fatalf("Unregister: %v", err)
	}

	// Verify gone
	_, err = repo.client.ZScore(ctx, schedulerQueueKey, "10").Result()
	if err != redis.Nil {
		t.Errorf("expected redis.Nil after unregister, got %v", err)
	}
}

func TestClaimDueTasks(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	now := time.Now()
	pastScore := now.Add(-time.Hour).UnixMilli()
	futureScore := now.Add(time.Hour).UnixMilli()

	// Manually insert endpoints with known scores
	client := repo.client
	client.ZAdd(ctx, schedulerQueueKey,
		redis.Z{Member: "1", Score: float64(pastScore)},
		redis.Z{Member: "2", Score: float64(pastScore)},
		redis.Z{Member: "3", Score: float64(futureScore)},
	)

	t.Run("claims due tasks and bumps scores", func(t *testing.T) {
		due, next, hasNext, err := repo.ClaimDueTasks(ctx, 10)
		if err != nil {
			t.Fatalf("ClaimDueTasks: %v", err)
		}
		if len(due) != 2 {
			t.Fatalf("expected 2 due tasks, got %d", len(due))
		}

		// Returned due tasks should have original past scores (pre-bump)
		for _, task := range due {
			if task.EndpointID == 1 || task.EndpointID == 2 {
				if task.Score != pastScore {
					t.Errorf("task %d original score = %d, want %d", task.EndpointID, task.Score, pastScore)
				}
			} else {
				t.Errorf("unexpected due endpoint: %d", task.EndpointID)
			}
			// ZSET should have the bumped (locked) score, not the original
			member := fmt.Sprint(task.EndpointID)
			zsetScore, err := client.ZScore(ctx, schedulerQueueKey, member).Result()
			if err != nil {
				t.Fatalf("ZScore for %s: %v", member, err)
			}
			if zsetScore <= float64(pastScore) {
				t.Errorf("ZSET score for %s = %f, expected > %d (bumped)", member, zsetScore, pastScore)
			}
		}

		// Should have a next task (the future one, unchanged)
		if !hasNext {
			t.Error("expected hasNext=true")
		}
		if next.EndpointID != 3 {
			t.Errorf("next.EndpointID = %d, want 3", next.EndpointID)
		}
		if next.Score != futureScore {
			t.Errorf("next.Score = %d, want %d", next.Score, futureScore)
		}
	})
}

func TestClaimDueTasksEmptyQueue(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	due, next, hasNext, err := repo.ClaimDueTasks(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimDueTasks on empty: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due, got %d", len(due))
	}
	if hasNext {
		t.Error("expected hasNext=false for empty queue")
	}
	if next != (ScheduledTask{}) {
		t.Errorf("expected zero next, got %+v", next)
	}
}

func TestClaimDueTasksNoDue(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	client := repo.client
	futureScore := time.Now().Add(time.Hour).UnixMilli()
	client.ZAdd(ctx, schedulerQueueKey, redis.Z{Member: "1", Score: float64(futureScore)})

	due, next, hasNext, err := repo.ClaimDueTasks(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimDueTasks: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due, got %d", len(due))
	}
	if !hasNext {
		t.Error("expected hasNext=true when future task exists")
	}
	if next.EndpointID != 1 {
		t.Errorf("next.EndpointID = %d, want 1", next.EndpointID)
	}
}

func TestClaimDueTasksPartialClaim(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	client := repo.client
	now := time.Now()
	pastScore := now.Add(-time.Hour).UnixMilli()

	client.ZAdd(ctx, schedulerQueueKey,
		redis.Z{Member: "1", Score: float64(pastScore)},
		redis.Z{Member: "2", Score: float64(pastScore)},
		redis.Z{Member: "3", Score: float64(pastScore)},
	)

	// Claim only 2 out of 3
	due, _, _, err := repo.ClaimDueTasks(ctx, 2)
	if err != nil {
		t.Fatalf("ClaimDueTasks: %v", err)
	}
	if len(due) != 2 {
		t.Errorf("expected 2 due tasks, got %d", len(due))
	}

	// Claim the last one
	due, _, _, err = repo.ClaimDueTasks(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimDueTasks second round: %v", err)
	}
	if len(due) != 1 {
		t.Errorf("expected 1 due task in second round, got %d", len(due))
	}
	if due[0].EndpointID != 3 {
		t.Errorf("expected endpoint 3, got %d", due[0].EndpointID)
	}
}

func TestScoreUpdaterUpdateBatch(t *testing.T) {
	testcontainers.SkipIfShort(t)
	client := testcontainers.NewTestRedis(t, testRedisAddr)
	repo := NewZSetScheduleRepository(client)
	updater := newScoreUpdater(t, client)
	ctx := context.Background()

	now := time.Now()
	pastScore := now.Add(-time.Hour).UnixMilli()

	client.ZAdd(ctx, schedulerQueueKey,
		redis.Z{Member: "1", Score: float64(pastScore)},
	)

	// Update the score to a future time
	newScore := now.Add(2 * time.Hour).UnixMilli()
	err := updater.UpdateBatch(ctx, map[uint]int64{1: newScore})
	if err != nil {
		t.Fatalf("UpdateBatch: %v", err)
	}

	// Verify score was updated
	score, err := client.ZScore(ctx, schedulerQueueKey, "1").Result()
	if err != nil {
		t.Fatalf("ZScore after update: %v", err)
	}
	if score != float64(newScore) {
		t.Errorf("score = %f, want %d", score, newScore)
	}

	// ClaimDueTasks should not return it (it's in the future)
	due, _, _, err := repo.ClaimDueTasks(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimDueTasks: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due tasks after rescheduling to future, got %d", len(due))
	}
}

func TestClaimDueTasksLockPreventsReclaim(t *testing.T) {
	repo := newRepo(t)
	ctx := context.Background()

	now := time.Now()
	pastScore := now.Add(-time.Hour).UnixMilli()

	repo.client.ZAdd(ctx, schedulerQueueKey, redis.Z{Member: "1", Score: float64(pastScore)})

	// First claim
	due, _, _, err := repo.ClaimDueTasks(ctx, 10)
	if err != nil {
		t.Fatalf("first ClaimDueTasks: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due in first claim, got %d", len(due))
	}

	// Second claim — should NOT return it (locked)
	due, _, _, err = repo.ClaimDueTasks(ctx, 10)
	if err != nil {
		t.Fatalf("second ClaimDueTasks: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due in second claim (locked), got %d: %+v", len(due), due)
	}
}
