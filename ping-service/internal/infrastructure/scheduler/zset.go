package scheduler

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/utils"
)

const (
	schedulerQueuePrefix = "scheduler:queue"
	claimLock            = 10 * time.Second
)

func shardKey(shardID uint) string {
	return fmt.Sprintf("%s:%d", schedulerQueuePrefix, shardID)
}

type ScheduledTask struct {
	EndpointID uint
	Score      int64 // next execution time in UnixMilliseconds
}

type ZSetScheduleRepository struct {
	client       *redis.Client
	scoreUpdater *ScoreUpdater
	claimer      *ZSetTaskClaimer
	shardCount   int
}

func NewZSetScheduleRepository(
	client *redis.Client,
	scoreUpdater *ScoreUpdater,
	claimer *ZSetTaskClaimer,
	shardCount int,
) *ZSetScheduleRepository {

	if shardCount < 1 {
		shardCount = 1
	}

	return &ZSetScheduleRepository{
		client:       client,
		shardCount:   shardCount,
		scoreUpdater: scoreUpdater,
		claimer:      claimer,
	}
}

func (r *ZSetScheduleRepository) ClaimDueTasksForShard(
	ctx context.Context, shardID uint, limit int64,
) (due []ScheduledTask, next ScheduledTask, hasNext bool, err error) {
	return r.claimer.ClaimDueTasksForShard(ctx, shardID, limit)
}

func RegisterZSetScheduleRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetScheduleRepository, error) {

		cfg := do.MustInvoke[*config.Config](i)
		scoreUpdater := do.MustInvoke[*ScoreUpdater](i)
		claimer := do.MustInvoke[*ZSetTaskClaimer](i)
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)

		return NewZSetScheduleRepository(
			wrapper.GetClient(),
			scoreUpdater,
			claimer,
			cfg.Redis.SchedulerShards,
		), nil
	})
}

func (r *ZSetScheduleRepository) Register(ctx context.Context, endpoint *domain.Endpoint) error {
	return r.RegisterBatch(ctx, []domain.Endpoint{*endpoint})
}

func (r *ZSetScheduleRepository) RegisterBatch(ctx context.Context, endpoints []domain.Endpoint) error {

	if len(endpoints) == 0 {
		return nil
	}

	scoreMap := make(map[uint]int64, len(endpoints))
	for _, endpoint := range endpoints {

		score, err := utils.NextExecutionTime(endpoint.ID, endpoint.Interval)
		if err != nil {
			return fmt.Errorf("failed to calculate next execution time: %w", err)
		}

		scoreMap[endpoint.ID] = score.UnixMilli()
	}

	return r.scoreUpdater.UpdateBatch(ctx, scoreMap)
}

func (r *ZSetScheduleRepository) Unregister(ctx context.Context, endpointID uint) error {

	zsetKey, err := schedulerShardKey(r.shardCount, endpointID)
	if err != nil {
		return fmt.Errorf("failed to get shard key: %w", err)
	}

	cmd := r.client.ZRem(ctx, zsetKey, fmt.Sprint(endpointID))

	return cmd.Err()
}

func (r *ZSetScheduleRepository) MoveIfWrongShard(
	ctx context.Context, shardID uint, due []ScheduledTask,
) ([]ScheduledTask, error) {

	filtered := make([]ScheduledTask, 0, len(due))
	claimedKey := shardKey(shardID)

	for _, task := range due {

		correctKey, err := schedulerShardKey(r.shardCount, task.EndpointID)
		if err != nil {
			return nil, fmt.Errorf("failed to get shard key: %w", err)
		}

		if correctKey == claimedKey {
			filtered = append(filtered, task)
			continue
		}

		_, err = r.client.TxPipelined(ctx, func(pipe redis.Pipeliner) error {

			pipe.ZAdd(ctx, correctKey, redis.Z{
				Member: fmt.Sprint(task.EndpointID),
				Score:  float64(task.Score),
			})

			pipe.ZRem(ctx, claimedKey, fmt.Sprint(task.EndpointID))

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("move task %d to correct shard: %w", task.EndpointID, err)
		}
	}

	return filtered, nil
}
