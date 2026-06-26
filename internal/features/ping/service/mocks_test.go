package service

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type mockEndpointProvider struct {
	getBatchFn func(ctx context.Context, ids []uint) (map[uint]*domain.Endpoint, error)
}

func (m *mockEndpointProvider) GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Endpoint, error) {
	return m.getBatchFn(ctx, ids)
}

type mockScoreUpdater struct {
	updateBatchFn func(ctx context.Context, items map[uint]int64) error
}

func (m *mockScoreUpdater) UpdateBatch(ctx context.Context, items map[uint]int64) error {
	return m.updateBatchFn(ctx, items)
}
