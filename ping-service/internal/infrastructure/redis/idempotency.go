package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
)

//revive:disable-next-line:exported
type RedisOffsetStore struct {
	client        *redis.Client
	staleDuration time.Duration
}

func NewRedisOffsetStore(client *redis.Client, staleDuration time.Duration) *RedisOffsetStore {
	return &RedisOffsetStore{
		client:        client,
		staleDuration: staleDuration,
	}
}

func RegisterRedisOffsetStore(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RedisOffsetStore, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return NewRedisOffsetStore(wrapper.GetClient(), 30*time.Minute), nil
	})
}

func (s *RedisOffsetStore) GetOffset(ctx context.Context, key string) (string, error) {

	offset, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", nil
	}

	return offset, err
}

func (s *RedisOffsetStore) SetOffset(ctx context.Context, key string, offset string) error {
	return s.client.Set(ctx, key, offset, s.staleDuration).Err()
}

func parseOffset(offset string) (ms, seq uint64, err error) {
	_, err = fmt.Sscanf(offset, "%d-%d", &ms, &seq)
	return
}

func (s *RedisOffsetStore) IsNewer(a, b string) (bool, error) {

	msA, seqA, err := parseOffset(a)
	if err != nil {
		return false, err
	}

	msB, seqB, err := parseOffset(b)
	if err != nil {
		return false, err
	}

	return msA > msB || (msA == msB && seqA > seqB), nil
}
