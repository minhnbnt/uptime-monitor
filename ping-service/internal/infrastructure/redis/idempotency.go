package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type OffsetStore struct {
	client        *redis.Client
	staleDuration time.Duration
}

func NewOffsetStore(client *redis.Client, staleDuration time.Duration) *OffsetStore {
	return &OffsetStore{
		client:        client,
		staleDuration: staleDuration,
	}
}

func offsetKey(id uint) string {
	return fmt.Sprintf("offset:ping:endpointID:%d", id)
}

func (s *OffsetStore) GetOffset(ctx context.Context, id uint) (string, error) {

	offset, err := s.client.Get(ctx, offsetKey(id)).Result()
	if err != nil {
		return "", err
	}

	return offset, nil
}

func (s *OffsetStore) SetOffset(ctx context.Context, id uint, offset string) error {
	return s.client.Set(ctx, offsetKey(id), offset, s.staleDuration).Err()
}

func parseOffset(offset string) (ms, seq uint64, err error) {
	_, err = fmt.Sscanf(offset, "%d-%d", &ms, &seq)
	return
}

func (s *OffsetStore) IsNewer(a, b string) (bool, error) {

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
