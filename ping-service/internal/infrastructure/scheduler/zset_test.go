package scheduler

import (
	"strings"
	"testing"
)

func TestSchedulerShardKey(t *testing.T) {
	t.Run("single shard returns prefix", func(t *testing.T) {
		key := schedulerShardKey(1, 42)
		if key != schedulerQueuePrefix {
			t.Errorf("got %q, want %q", key, schedulerQueuePrefix)
		}
	})

	t.Run("zero or negative treated as single", func(t *testing.T) {
		if k := schedulerShardKey(0, 1); k != schedulerQueuePrefix {
			t.Errorf("shardCount=0 got %q, want %q", k, schedulerQueuePrefix)
		}
		if k := schedulerShardKey(-1, 1); k != schedulerQueuePrefix {
			t.Errorf("shardCount=-1 got %q, want %q", k, schedulerQueuePrefix)
		}
	})

	t.Run("sharded key has prefix and numeric suffix", func(t *testing.T) {
		key := schedulerShardKey(8, 42)
		if !strings.HasPrefix(key, schedulerQueuePrefix+":") {
			t.Errorf("key %q does not start with %q:", key, schedulerQueuePrefix+":")
		}
		suffix := key[len(schedulerQueuePrefix)+1:]
		if suffix == "" {
			t.Error("empty shard ID suffix")
		}
	})

	t.Run("shard ID is within range", func(t *testing.T) {
		const shards = 8
		for id := uint(0); id < 100; id++ {
			key := schedulerShardKey(shards, id)
			suffix := key[len(schedulerQueuePrefix)+1:]
			shardID := 0
			for _, c := range suffix {
				shardID = shardID*10 + int(c-'0')
			}
			if shardID < 0 || shardID >= shards {
				t.Errorf("endpoint %d routed to shard %d, expected [0,%d)", id, shardID, shards)
			}
		}
	})

	t.Run("deterministic for same endpoint", func(t *testing.T) {
		k1 := schedulerShardKey(8, 42)
		k2 := schedulerShardKey(8, 42)
		if k1 != k2 {
			t.Errorf("not deterministic: %q vs %q", k1, k2)
		}
	})

	t.Run("different endpoints may map to different shards", func(t *testing.T) {
		shards := make(map[string]bool)
		for id := uint(1); id <= 100; id++ {
			shards[schedulerShardKey(8, id)] = true
		}
		if len(shards) < 2 {
			t.Error("all 100 endpoints mapped to the same shard — unexpected")
		}
	})
}

func TestGetScheduledTask(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		task, err := getScheduledTask("42", "1000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if task.EndpointID != 42 {
			t.Errorf("EndpointID = %d, want 42", task.EndpointID)
		}
		if task.Score != 1000 {
			t.Errorf("Score = %d, want 1000", task.Score)
		}
	})

	t.Run("invalid member type", func(t *testing.T) {
		_, err := getScheduledTask(42, "1000")
		if err == nil {
			t.Fatal("expected error for int member")
		}
	})

	t.Run("invalid score type", func(t *testing.T) {
		_, err := getScheduledTask("42", 1000)
		if err == nil {
			t.Fatal("expected error for int score")
		}
	})

	t.Run("non-numeric member string", func(t *testing.T) {
		_, err := getScheduledTask("abc", "1000")
		if err == nil {
			t.Fatal("expected error for non-numeric member")
		}
	})

	t.Run("non-numeric score string", func(t *testing.T) {
		_, err := getScheduledTask("42", "not-a-number")
		if err == nil {
			t.Fatal("expected error for non-numeric score")
		}
	})
}

func TestClaimDueTasksZeroLimit(t *testing.T) {
	t.Run("zero limit returns nil due", func(t *testing.T) {
		r := &ZSetScheduleRepository{}
		due, next, hasNext, err := r.ClaimDueTasks(t.Context(), 0)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if due != nil {
			t.Errorf("due = %v, want nil", due)
		}
		if hasNext {
			t.Error("hasNext should be false")
		}
		if next != (ScheduledTask{}) {
			t.Errorf("next = %v, want zero value", next)
		}
	})
}
