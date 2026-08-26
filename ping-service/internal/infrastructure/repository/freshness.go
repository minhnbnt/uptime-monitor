package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	scheduler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

const pushFreshnessPrefix = "push:freshness"

func pushFreshnessKeyFor(shardID uint) string {
	return fmt.Sprintf("%s:%d", pushFreshnessPrefix, shardID)
}

// FreshnessStore tracks how fresh each server's last recorded event is.
// Member = server ID, score = stale deadline in UnixMilliseconds.
type FreshnessStore struct {
	updater *scheduler.ScoreUpdater
	claimer *scheduler.ZSetTaskClaimer
}

func NewFreshnessStore(client *redis.Client, shardCount int) *FreshnessStore {
	return &FreshnessStore{
		updater: scheduler.NewScoreUpdater(client, shardCount, pushFreshnessKeyFor),
		claimer: scheduler.NewZSetTaskClaimer(client, pushFreshnessKeyFor),
	}
}

func RegisterFreshnessStore(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*FreshnessStore, error) {

		cfg := do.MustInvoke[*config.Config](i)
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)

		return NewFreshnessStore(wrapper.GetClient(), cfg.Redis.SchedulerShards), nil
	})
}

// Touch pushes back the stale deadline of a server by lease.
func (f *FreshnessStore) Touch(ctx context.Context, endpointID uint, lease time.Duration) error {
	return f.updater.Update(ctx, endpointID, time.Now().Add(lease).UnixMilli())
}

func (f *FreshnessStore) Remove(ctx context.Context, endpointID uint) error {
	return f.updater.Remove(ctx, endpointID)
}

func (f *FreshnessStore) ClaimOverdue(
	ctx context.Context, shardID uint, limit int64,
) (due []scheduler.ScheduledTask, next scheduler.ScheduledTask, hasNext bool, err error) {
	return f.claimer.ClaimDueTasksForShard(ctx, shardID, limit)
}
