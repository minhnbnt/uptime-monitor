package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

const (
	metaCachePrefix = "scheduler:meta:"
	metaCacheTTL    = 1 * time.Hour
)

func metaCacheKey(id uint) string {
	return fmt.Sprintf("%s%d", metaCachePrefix, id)
}

type ServerMetaCache struct {
	client *redis.Client
}

func NewServerMetaCache(client *redis.Client) *ServerMetaCache {
	return &ServerMetaCache{client: client}
}

func RegisterServerMetaCache(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerMetaCache, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return NewServerMetaCache(wrapper.GetClient()), nil
	})
}

func (c *ServerMetaCache) Get(ctx context.Context, id uint) (*domain.Server, error) {

	results, err := c.MGet(ctx, []uint{id})
	if err != nil {
		return nil, err
	}

	result, ok := results[id]
	if !ok {
		return nil, fmt.Errorf("server %d not found", id)
	}

	return result, nil
}

func (c *ServerMetaCache) MGet(ctx context.Context, ids []uint) (map[uint]*domain.Server, error) {

	result := make(map[uint]*domain.Server, len(ids))
	if len(ids) == 0 {
		return result, nil
	}

	pipe := c.client.Pipeline()
	cmds := make(map[uint]*redis.MapStringStringCmd, len(ids))

	for _, id := range ids {
		cmds[id] = pipe.HGetAll(ctx, metaCacheKey(id))
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("pipeline hgetall: %w", err)
	}

	for id, cmd := range cmds {

		data, err := cmd.Result()
		if err != nil {
			return nil, fmt.Errorf("hgetall %d: %w", id, err)
		}

		if len(data) == 0 {
			continue
		}

		sv, err := mapToServer(id, data)
		if err != nil {
			continue
		}

		result[id] = sv
	}

	return result, nil
}

func (c *ServerMetaCache) Set(ctx context.Context, sv *domain.Server) error {
	return c.SetMulti(ctx, []*domain.Server{sv})
}

func (c *ServerMetaCache) SetMulti(ctx context.Context, servers []*domain.Server) error {

	if len(servers) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	for _, sv := range servers {

		key := metaCacheKey(sv.ID)

		pipe.HSet(
			ctx, key,
			"namespace", sv.Namespace,
			"kind", sv.Kind,
			"object_id", sv.ObjectID,
			"container_name", sv.ContainerName,
			"interval_ns", fmt.Sprint(sv.Interval.Nanoseconds()),
			"timeout_ns", fmt.Sprint(sv.Timeout.Nanoseconds()),
		)

		pipe.Expire(ctx, key, metaCacheTTL)
	}

	_, err := pipe.Exec(ctx)

	return err
}

func (c *ServerMetaCache) Delete(ctx context.Context, id uint) error {
	return c.DeleteMulti(ctx, []uint{id})
}

func (c *ServerMetaCache) DeleteMulti(ctx context.Context, ids []uint) error {

	if len(ids) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	for _, id := range ids {
		pipe.Del(ctx, metaCacheKey(id))
	}

	_, err := pipe.Exec(ctx)

	return err
}

func mapToServer(id uint, data map[string]string) (*domain.Server, error) {

	intervalNs, err := strconv.ParseInt(data["interval_ns"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse interval_ns: %w", err)
	}

	timeoutNs, err := strconv.ParseInt(data["timeout_ns"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse timeout_ns: %w", err)
	}

	return &domain.Server{
		Model:         gorm.Model{ID: id},
		Namespace:     data["namespace"],
		Kind:          data["kind"],
		ObjectID:      data["object_id"],
		ContainerName: data["container_name"],
		Interval:      time.Duration(intervalNs),
		Timeout:       time.Duration(timeoutNs),
	}, nil
}
