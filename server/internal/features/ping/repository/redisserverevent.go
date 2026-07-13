package repository

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
	statusKey  = "endpoint:status"
	defaultTTL = 7 * 24 * time.Hour
)

type RedisServerEventRepository struct {
	client *redis.Client
}

func NewRedisServerEventRepository(client *redis.Client) *RedisServerEventRepository {
	return &RedisServerEventRepository{client: client}
}

func newRedisServerEventRepository(i do.Injector) (*RedisServerEventRepository, error) {
	wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
	return NewRedisServerEventRepository(wrapper.GetClient()), nil
}

func RegisterRedisServerEventRepository(i do.Injector) {
	do.Provide(i, newRedisServerEventRepository)
}

func (r *RedisServerEventRepository) GetStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error) {

	val, err := r.client.HGet(ctx, statusKey, fmt.Sprint(endpointID)).Result()
	if err != nil && err != redis.Nil {
		return "", err
	}

	return domain.ServerStatus(val), nil
}

func (r *RedisServerEventRepository) SetStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error {

	pipe := r.client.Pipeline()

	pipe.HSet(ctx, statusKey, fmt.Sprint(endpointID), string(status))
	pipe.HExpire(ctx, statusKey, defaultTTL, fmt.Sprint(endpointID))

	_, err := pipe.Exec(ctx)

	return err
}

func (r *RedisServerEventRepository) DeleteStatus(ctx context.Context, endpointID uint) error {
	return r.client.HDel(ctx, statusKey, fmt.Sprint(endpointID)).Err()
}
