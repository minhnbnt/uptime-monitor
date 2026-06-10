package monitor

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

const (
	statusKeyPrefix = "endpoint:"
	statusKeySuffix = ":status"
	defaultTTL      = 7 * 24 * time.Hour
)

type RedisServerEventRepository struct {
	client *redis.Client
}

func newRedisServerEventRepository(i do.Injector) (*RedisServerEventRepository, error) {
	wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
	return &RedisServerEventRepository{client: wrapper.GetClient()}, nil
}

func RegisterRedisServerEventRepository(i do.Injector) {
	do.Provide(i, newRedisServerEventRepository)
}

func statusKey(endpointID uint) string {
	return fmt.Sprintf("%s%d%s", statusKeyPrefix, endpointID, statusKeySuffix)
}

func (r *RedisServerEventRepository) GetStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error) {

	val, err := r.client.Get(ctx, statusKey(endpointID)).Result()
	if err != nil && err != redis.Nil {
		return "", err
	}

	return domain.ServerStatus(val), nil
}

func (r *RedisServerEventRepository) SetStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error {
	return r.client.Set(ctx, statusKey(endpointID), string(status), defaultTTL).Err()
}

func (r *RedisServerEventRepository) DeleteStatus(ctx context.Context, endpointID uint) error {
	return r.client.Del(ctx, statusKey(endpointID)).Err()
}
