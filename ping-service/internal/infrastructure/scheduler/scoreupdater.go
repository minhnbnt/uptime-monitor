package scheduler

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/utils"
)

type ScoreUpdater struct {
	client     *redis.Client
	shardCount int
	keyFor     func(shardID uint) string
}

func NewScoreUpdater(client *redis.Client, shardCount int, keyFor func(shardID uint) string) *ScoreUpdater {

	if shardCount < 1 {
		shardCount = 1
	}

	return &ScoreUpdater{client: client, shardCount: shardCount, keyFor: keyFor}
}

func RegisterScoreUpdater(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ScoreUpdater, error) {

		cfg := do.MustInvoke[*config.Config](i)
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)

		return NewScoreUpdater(wrapper.GetClient(), cfg.Redis.SchedulerShards, shardKey), nil
	})
}

func (u *ScoreUpdater) ShardKeyFor(endpointID uint) (string, error) {

	hash, err := utils.Hash(endpointID)
	if err != nil {
		return "", err
	}

	shardID := hash % uint64(u.shardCount)

	return u.keyFor(uint(shardID)), nil
}

func (u *ScoreUpdater) Update(ctx context.Context, endpointID uint, nextScore int64) error {
	return u.UpdateBatch(ctx, map[uint]int64{endpointID: nextScore})
}

func (u *ScoreUpdater) UpdateBatch(ctx context.Context, items map[uint]int64) error {

	if len(items) == 0 {
		return nil
	}

	scores := make(map[string][]redis.Z, len(items))
	for id, score := range items {

		key, err := u.ShardKeyFor(id)
		if err != nil {
			return fmt.Errorf("failed to get shard key: %w", err)
		}

		score := redis.Z{
			Member: fmt.Sprint(id),
			Score:  float64(score),
		}

		scores[key] = append(scores[key], score)
	}

	pipe := u.client.Pipeline()

	for key, scores := range scores {
		pipe.ZAdd(ctx, key, scores...)
	}

	_, err := pipe.Exec(ctx)

	return err
}

func (u *ScoreUpdater) Remove(ctx context.Context, endpointID uint) error {

	zsetKey, err := u.ShardKeyFor(endpointID)
	if err != nil {
		return fmt.Errorf("failed to get shard key: %w", err)
	}

	cmd := u.client.ZRem(ctx, zsetKey, fmt.Sprint(endpointID))

	return cmd.Err()
}
