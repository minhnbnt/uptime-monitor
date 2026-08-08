package scheduler

import (
	"context"
	"errors"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

const (
	domainCachePrefix = "scheduler:domain:"
	domainCacheTTL    = metaCacheTTL
)

func domainCacheKey(key dto.K8sObjectKey) string {
	return fmt.Sprintf("%s%s:%s:%s", domainCachePrefix, key.Namespace, key.Kind, key.ObjectID)
}

type DomainCache struct {
	client *redis.Client
}

func NewDomainCache(client *redis.Client) *DomainCache {
	return &DomainCache{client: client}
}

func RegisterDomainCache(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*DomainCache, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return NewDomainCache(wrapper.GetClient()), nil
	})
}

func (c *DomainCache) Get(ctx context.Context, key dto.K8sObjectKey) (domain string, ok bool, err error) {

	val, err := c.client.Get(ctx, domainCacheKey(key)).Result()
	if errors.Is(err, redis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}

	return val, true, nil
}

func (c *DomainCache) Set(ctx context.Context, key dto.K8sObjectKey, domain string) error {

	if domain == "" {
		return nil
	}

	return c.client.Set(ctx, domainCacheKey(key), domain, domainCacheTTL).Err()
}

func (c *DomainCache) Delete(ctx context.Context, key dto.K8sObjectKey) error {
	return c.client.Del(ctx, domainCacheKey(key)).Err()
}
