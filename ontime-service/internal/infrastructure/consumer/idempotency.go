package consumer

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

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

func offsetKey(id uint) string {
	return fmt.Sprintf("offset:ontime:serverID:%d", id)
}

func (s *RedisOffsetStore) GetOffset(ctx context.Context, id uint) (string, error) {

	key := offsetKey(id)
	offset, err := s.client.Get(ctx, key).Result()

	if err != nil {
		return "", err
	}

	return offset, nil
}

func (s *RedisOffsetStore) SetOffset(ctx context.Context, id uint, offset string) error {
	return s.client.Set(ctx, offsetKey(id), offset, s.staleDuration).Err()
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

	if msA > msB {
		return true, nil
	}

	if msA < msB {
		return false, nil
	}

	return seqA > seqB, nil
}
