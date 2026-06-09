package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

const metaCachePrefix = "scheduler:meta:"

func metaCacheKey(id uint) string {
	return fmt.Sprintf("%s%d", metaCachePrefix, id)
}

type EndpointMetaCache struct {
	client *redis.Client
}

func RegisterEndpointMetaCache(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointMetaCache, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &EndpointMetaCache{client: wrapper.GetClient()}, nil
	})
}

func (c *EndpointMetaCache) Get(ctx context.Context, id uint) (*domain.Endpoint, error) {

	data, err := c.client.HGetAll(ctx, metaCacheKey(id)).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall %d: %w", id, err)
	}
	if len(data) == 0 {
		return nil, nil
	}

	return mapToEndpoint(id, data)
}

func (c *EndpointMetaCache) MGet(ctx context.Context, ids []uint) (map[uint]*domain.Endpoint, error) {

	result := make(map[uint]*domain.Endpoint, len(ids))
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

		ep, err := mapToEndpoint(id, data)
		if err != nil {
			return nil, err
		}

		result[id] = ep
	}

	return result, nil
}

func (c *EndpointMetaCache) Set(ctx context.Context, ep *domain.Endpoint) error {

	cmd := c.client.HSet(
		ctx, metaCacheKey(ep.ID),
		"url", ep.URL,
		"method", ep.Method,
		"expected_code", fmt.Sprint(ep.ExpectedCode),
		"interval_ns", fmt.Sprint(ep.Interval),
	)

	return cmd.Err()
}

func (c *EndpointMetaCache) SetMulti(ctx context.Context, endpoints []domain.Endpoint) error {

	if len(endpoints) == 0 {
		return nil
	}

	pipe := c.client.Pipeline()

	for i := range endpoints {
		pipe.HSet(
			ctx, metaCacheKey(endpoints[i].ID),
			"url", endpoints[i].URL,
			"method", endpoints[i].Method,
			"expected_code", fmt.Sprint(endpoints[i].ExpectedCode),
			"interval_ns", fmt.Sprint(endpoints[i].Interval),
		)
	}

	_, err := pipe.Exec(ctx)

	return err
}

func mapToEndpoint(id uint, data map[string]string) (*domain.Endpoint, error) {

	intervalNs, err := strconv.ParseInt(data["interval_ns"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse interval_ns: %w", err)
	}

	expectedCode, err := strconv.Atoi(data["expected_code"])
	if err != nil {
		return nil, fmt.Errorf("parse expected_code: %w", err)
	}

	return &domain.Endpoint{
		Model:        gorm.Model{ID: id},
		URL:          data["url"],
		Method:       data["method"],
		ExpectedCode: expectedCode,
		Interval:     time.Duration(intervalNs),
	}, nil
}
