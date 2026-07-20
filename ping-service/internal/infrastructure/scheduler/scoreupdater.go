package scheduler

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
)

type ScoreUpdater struct {
	client     *redis.Client
	shardCount int
}

func NewScoreUpdater(client *redis.Client, shardCount int) *ScoreUpdater {

	if shardCount < 1 {
		shardCount = 1
	}

	return &ScoreUpdater{client: client, shardCount: shardCount}
}

func RegisterScoreUpdater(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ScoreUpdater, error) {
		cfg := do.MustInvoke[*config.Config](i)
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return NewScoreUpdater(wrapper.GetClient(), cfg.Redis.SchedulerShards), nil
	})
}

func (u *ScoreUpdater) Update(ctx context.Context, endpointID uint, nextScore int64) error {
	return u.UpdateBatch(ctx, map[uint]int64{endpointID: nextScore})
}

func (u *ScoreUpdater) UpdateBatch(ctx context.Context, items map[uint]int64) error {

	if len(items) == 0 {
		return nil
	}

	scores := make(map[string][]redis.Z)
	for id, score := range items {

		key := schedulerShardKey(u.shardCount, id)

		score := redis.Z{Score: float64(score), Member: fmt.Sprint(id)}
		scores[key] = append(scores[key], score)
	}

	pipe := u.client.Pipeline()

	for key, scores := range scores {
		pipe.ZAdd(ctx, key, scores...)
	}

	_, err := pipe.Exec(ctx)

	return err
}
