package service

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type mockEndpointProvider struct {
	getBatchFn func(ctx context.Context, ids []uint) (map[uint]*domain.Endpoint, error)
}

func (m *mockEndpointProvider) GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Endpoint, error) {
	return m.getBatchFn(ctx, ids)
}

type mockScoreUpdater struct {
	updateFn func(ctx context.Context, endpointID uint, nextScore int64) error
}

func (m *mockScoreUpdater) Update(ctx context.Context, endpointID uint, nextScore int64) error {
	if m.updateFn == nil {
		return nil
	}
	return m.updateFn(ctx, endpointID, nextScore)
}

var _ endpointProvider = (*mockEndpointProvider)(nil)
var _ scoreUpdater = (*mockScoreUpdater)(nil)
