package scheduler

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
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

func newRepo(tb testing.TB, shardCount ...int) *ZSetScheduleRepository {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	client := testcontainers.NewTestRedis(tb, testRedisAddr)
	sc := 1
	if len(shardCount) > 0 {
		sc = shardCount[0]
	}
	updater := NewScoreUpdater(client, sc)
	claimer := NewZSetTaskClaimer(client)
	return NewZSetScheduleRepository(client, updater, claimer, sc)
}

func newScoreUpdater(tb testing.TB, client *redis.Client, shardCount ...int) *ScoreUpdater {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	sc := 1
	if len(shardCount) > 0 {
		sc = shardCount[0]
	}
	return NewScoreUpdater(client, sc)
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
	count, err := client.ZCard(ctx, shardKey(0)).Result()
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
	exists, err := repo.client.ZScore(ctx, shardKey(0), "10").Result()
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
	_, err = repo.client.ZScore(ctx, shardKey(0), "10").Result()
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
	client.ZAdd(ctx, shardKey(0),
		redis.Z{Member: "1", Score: float64(pastScore)},
		redis.Z{Member: "2", Score: float64(pastScore)},
		redis.Z{Member: "3", Score: float64(futureScore)},
	)

	t.Run("claims due tasks and bumps scores", func(t *testing.T) {
		due, next, hasNext, err := repo.ClaimDueTasksForShard(ctx, 0, 10)
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
			zsetScore, err := client.ZScore(ctx, shardKey(0), member).Result()
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

	due, next, hasNext, err := repo.ClaimDueTasksForShard(ctx, 0, 10)
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
	client.ZAdd(ctx, shardKey(0), redis.Z{Member: "1", Score: float64(futureScore)})

	due, next, hasNext, err := repo.ClaimDueTasksForShard(ctx, 0, 10)
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

	client.ZAdd(ctx, shardKey(0),
		redis.Z{Member: "1", Score: float64(pastScore)},
		redis.Z{Member: "2", Score: float64(pastScore)},
		redis.Z{Member: "3", Score: float64(pastScore)},
	)

	// Claim only 2 out of 3
	due, _, _, err := repo.ClaimDueTasksForShard(ctx, 0, 2)
	if err != nil {
		t.Fatalf("ClaimDueTasks: %v", err)
	}
	if len(due) != 2 {
		t.Errorf("expected 2 due tasks, got %d", len(due))
	}

	// Claim the last one
	due, _, _, err = repo.ClaimDueTasksForShard(ctx, 0, 10)
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
	updater := newScoreUpdater(t, client)
	claimer := NewZSetTaskClaimer(client)
	repo := NewZSetScheduleRepository(client, updater, claimer, 1)
	ctx := context.Background()

	now := time.Now()
	pastScore := now.Add(-time.Hour).UnixMilli()

	client.ZAdd(ctx, shardKey(0),
		redis.Z{Member: "1", Score: float64(pastScore)},
	)

	// Update the score to a future time
	newScore := now.Add(2 * time.Hour).UnixMilli()
	err := updater.UpdateBatch(ctx, map[uint]int64{1: newScore})
	if err != nil {
		t.Fatalf("UpdateBatch: %v", err)
	}

	// Verify score was updated
	score, err := client.ZScore(ctx, shardKey(0), "1").Result()
	if err != nil {
		t.Fatalf("ZScore after update: %v", err)
	}
	if score != float64(newScore) {
		t.Errorf("score = %f, want %d", score, newScore)
	}

	// ClaimDueTasks should not return it (it's in the future)
	due, _, _, err := repo.ClaimDueTasksForShard(ctx, 0, 10)
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

	repo.client.ZAdd(ctx, shardKey(0), redis.Z{Member: "1", Score: float64(pastScore)})

	// First claim
	due, _, _, err := repo.ClaimDueTasksForShard(ctx, 0, 10)
	if err != nil {
		t.Fatalf("first ClaimDueTasks: %v", err)
	}
	if len(due) != 1 {
		t.Fatalf("expected 1 due in first claim, got %d", len(due))
	}

	// Second claim — should NOT return it (locked)
	due, _, _, err = repo.ClaimDueTasksForShard(ctx, 0, 10)
	if err != nil {
		t.Fatalf("second ClaimDueTasks: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due in second claim (locked), got %d: %+v", len(due), due)
	}
}

// --- Sharded integration tests ---

func newShardedRepo(tb testing.TB, shardCount int) *ZSetScheduleRepository {
	tb.Helper()
	testcontainers.SkipIfShort(tb)
	client := testcontainers.NewTestRedis(tb, testRedisAddr)
	updater := NewScoreUpdater(client, shardCount)
	claimer := NewZSetTaskClaimer(client)
	return NewZSetScheduleRepository(client, updater, claimer, shardCount)
}

func TestRegisterBatchWithSharding(t *testing.T) {
	repo := newShardedRepo(t, 3)
	ctx := context.Background()

	endpoints := []domain.Endpoint{
		{Model: gorm.Model{ID: 1}, Interval: 30 * time.Second},
		{Model: gorm.Model{ID: 2}, Interval: 60 * time.Second},
		{Model: gorm.Model{ID: 3}, Interval: 90 * time.Second},
	}

	err := repo.RegisterBatch(ctx, endpoints)
	if err != nil {
		t.Fatalf("RegisterBatch: %v", err)
	}

	for _, ep := range endpoints {
		key, _ := schedulerShardKey(3, ep.ID)
		score, err := repo.client.ZScore(ctx, key, fmt.Sprint(ep.ID)).Result()
		if err != nil {
			t.Errorf("endpoint %d not found in shard key %q: %v", ep.ID, key, err)
		}
		if score <= 0 {
			t.Errorf("expected positive score for endpoint %d, got %f", ep.ID, score)
		}
	}

	// Verify total count across shards
	total := 0
	for i := 0; i < 3; i++ {
		c, _ := repo.client.ZCard(ctx, fmt.Sprintf("%s:%d", schedulerQueuePrefix, i)).Result()
		total += int(c)
	}
	if total != len(endpoints) {
		t.Errorf("total entries across shards = %d, want %d", total, len(endpoints))
	}
}

func TestClaimDueTasksForShardIsolatesShards(t *testing.T) {
	repo := newShardedRepo(t, 3)
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-time.Hour).UnixMilli()

	// Put a due task on shard 1 and a due task on shard 2.
	repo.client.ZAdd(ctx, schedulerQueuePrefix+":1",
		redis.Z{Member: "11", Score: float64(past)})
	repo.client.ZAdd(ctx, schedulerQueuePrefix+":2",
		redis.Z{Member: "22", Score: float64(past)})

	// Claiming shard 0 must return nothing.
	due0, _, _, err := repo.ClaimDueTasksForShard(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ClaimDueTasksForShard shard 0: %v", err)
	}
	if len(due0) != 0 {
		t.Errorf("shard 0 claimed %d tasks, want 0 (isolated)", len(due0))
	}

	// Claiming shard 1 must return only shard 1's task.
	due1, _, _, err := repo.ClaimDueTasksForShard(ctx, 1, 10)
	if err != nil {
		t.Fatalf("ClaimDueTasksForShard shard 1: %v", err)
	}
	if len(due1) != 1 || due1[0].EndpointID != 11 {
		t.Errorf("shard 1 claimed %+v, want endpoint 11 only", due1)
	}

	// Shard 2 still holds its own task.
	due2, _, _, err := repo.ClaimDueTasksForShard(ctx, 2, 10)
	if err != nil {
		t.Fatalf("ClaimDueTasksForShard shard 2: %v", err)
	}
	if len(due2) != 1 || due2[0].EndpointID != 22 {
		t.Errorf("shard 2 claimed %+v, want endpoint 22 only", due2)
	}
}

func TestUnregisterWithSharding(t *testing.T) {
	repo := newShardedRepo(t, 3)
	ctx := context.Background()

	err := repo.RegisterBatch(ctx, []domain.Endpoint{
		{Model: gorm.Model{ID: 10}, Interval: 30 * time.Second},
		{Model: gorm.Model{ID: 20}, Interval: 30 * time.Second},
	})
	if err != nil {
		t.Fatalf("RegisterBatch: %v", err)
	}

	key10, _ := schedulerShardKey(3, 10)
	key20, _ := schedulerShardKey(3, 20)

	// Confirm both are in the correct shards
	if _, err := repo.client.ZScore(ctx, key10, "10").Result(); err != nil {
		t.Fatalf("endpoint 10 not found: %v", err)
	}
	if _, err := repo.client.ZScore(ctx, key20, "20").Result(); err != nil {
		t.Fatalf("endpoint 20 not found: %v", err)
	}

	// Unregister endpoint 10
	err = repo.Unregister(ctx, 10)
	if err != nil {
		t.Fatalf("Unregister: %v", err)
	}

	// Endpoint 10 should be gone from its shard
	_, err = repo.client.ZScore(ctx, key10, "10").Result()
	if err != redis.Nil {
		t.Errorf("expected redis.Nil for unregistered endpoint, got %v", err)
	}

	// Endpoint 20 should still exist in its shard
	if _, err := repo.client.ZScore(ctx, key20, "20").Result(); err != nil {
		t.Errorf("endpoint 20 should still exist: %v", err)
	}
}

func TestScoreUpdaterUpdateBatchWithSharding(t *testing.T) {
	testcontainers.SkipIfShort(t)
	client := testcontainers.NewTestRedis(t, testRedisAddr)
	updater := NewScoreUpdater(client, 3)
	ctx := context.Background()

	now := time.Now()
	past := now.Add(-time.Hour).UnixMilli()

	// Seed endpoints in their respective shards
	for _, id := range []uint{1, 2, 3} {
		key, _ := schedulerShardKey(3, id)
		client.ZAdd(ctx, key, redis.Z{Member: fmt.Sprint(id), Score: float64(past)})
	}

	newScores := map[uint]int64{
		1: now.Add(2 * time.Hour).UnixMilli(),
		2: now.Add(3 * time.Hour).UnixMilli(),
		3: now.Add(4 * time.Hour).UnixMilli(),
	}

	err := updater.UpdateBatch(ctx, newScores)
	if err != nil {
		t.Fatalf("UpdateBatch: %v", err)
	}

	for id, expectedScore := range newScores {
		key, _ := schedulerShardKey(3, id)
		score, err := client.ZScore(ctx, key, fmt.Sprint(id)).Result()
		if err != nil {
			t.Errorf("endpoint %d not found in shard %q: %v", id, key, err)
			continue
		}
		if score != float64(expectedScore) {
			t.Errorf("endpoint %d score = %f, want %d", id, score, expectedScore)
		}
	}
}

func TestSingleShardBehaviorIsUnchanged(t *testing.T) {
	// Verify that with shardCount=1, all ops use the "scheduler:queue:0" key
	repo := newRepo(t)
	ctx := context.Background()

	err := repo.RegisterBatch(ctx, []domain.Endpoint{
		{Model: gorm.Model{ID: 1}, Interval: 30 * time.Second},
	})
	if err != nil {
		t.Fatalf("RegisterBatch: %v", err)
	}

	// Must be in the shard 0 key with a future score
	score, err := repo.client.ZScore(ctx, shardKey(0), "1").Result()
	if err != nil {
		t.Errorf("endpoint 1 not in %q: %v", shardKey(0), err)
	}
	if score <= float64(time.Now().UnixMilli()) {
		t.Errorf("expected future score, got %f", score)
	}

	// ClaimDueTasks only runs on the shard 0 key
	due, _, hasNext, err := repo.ClaimDueTasksForShard(ctx, 0, 10)
	if err != nil {
		t.Fatalf("ClaimDueTasks: %v", err)
	}
	if len(due) != 0 {
		t.Errorf("expected 0 due tasks (future score), got %d", len(due))
	}
	if !hasNext {
		t.Error("expected hasNext=true (future task exists)")
	}
}
