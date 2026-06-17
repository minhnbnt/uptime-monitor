package scheduler

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
)

type ScoreUpdater struct {
	client *redis.Client
}

func RegisterScoreUpdater(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ScoreUpdater, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &ScoreUpdater{client: wrapper.GetClient()}, nil
	})
}

func (u *ScoreUpdater) Update(ctx context.Context, endpointID uint, nextScore int64) error {
	return u.UpdateBatch(ctx, map[uint]int64{endpointID: nextScore})
}

func (u *ScoreUpdater) UpdateBatch(ctx context.Context, items map[uint]int64) error {

	if len(items) == 0 {
		return nil
	}

	pipe := u.client.Pipeline()

	for id, score := range items {
		pipe.ZAdd(ctx, schedulerQueueKey, redis.Z{
			Score:  float64(score),
			Member: fmt.Sprint(id),
		})
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("pipeline zadd: %w", err)
	}

	return nil
}
