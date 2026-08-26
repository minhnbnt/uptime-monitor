package scheduler

import (
	"strings"
	"testing"
)

func TestScoreUpdaterShardKeyFor(t *testing.T) {

	newUpdater := func(shardCount int) *ScoreUpdater {
		return NewScoreUpdater(nil, shardCount, shardKey)
	}

	t.Run("single shard maps to shard 0 key", func(t *testing.T) {
		key, _ := newUpdater(1).ShardKeyFor(42)
		if key != schedulerQueuePrefix+":0" {
			t.Errorf("got %q, want %q", key, schedulerQueuePrefix+":0")
		}
	})

	t.Run("zero or negative treated as single", func(t *testing.T) {
		if k, _ := newUpdater(0).ShardKeyFor(1); k != schedulerQueuePrefix+":0" {
			t.Errorf("shardCount=0 got %q, want %q", k, schedulerQueuePrefix+":0")
		}
		if k, _ := newUpdater(-1).ShardKeyFor(1); k != schedulerQueuePrefix+":0" {
			t.Errorf("shardCount=-1 got %q, want %q", k, schedulerQueuePrefix+":0")
		}
	})

	t.Run("keyFor is used verbatim", func(t *testing.T) {
		updater := NewScoreUpdater(nil, 4, func(shardID uint) string {
			return "custom:prefix:" + string(rune('0'+shardID))
		})
		key, err := updater.ShardKeyFor(7)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(key, "custom:prefix:") {
			t.Errorf("key %q does not use custom keyFor", key)
		}
	})

	t.Run("sharded key has prefix and numeric suffix", func(t *testing.T) {
		key, _ := newUpdater(8).ShardKeyFor(42)
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
		for id := range uint(100) {
			key, _ := newUpdater(shards).ShardKeyFor(id)
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
		k1, _ := newUpdater(8).ShardKeyFor(42)
		k2, _ := newUpdater(8).ShardKeyFor(42)
		if k1 != k2 {
			t.Errorf("not deterministic: %q vs %q", k1, k2)
		}
	})

	t.Run("different endpoints may map to different shards", func(t *testing.T) {
		shards := make(map[string]bool)
		for id := uint(1); id <= 100; id++ {
			k, _ := newUpdater(8).ShardKeyFor(id)
			shards[k] = true
		}
		if len(shards) < 2 {
			t.Error("all 100 endpoints mapped to the same shard — unexpected")
		}
	})
}
