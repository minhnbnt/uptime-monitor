package service

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type mockServerProvider struct {
	getBatchFn func(ctx context.Context, ids []uint) (map[uint]*domain.Server, error)
}

func (m *mockServerProvider) GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Server, error) {
	return m.getBatchFn(ctx, ids)
}

type mockScoreUpdater struct {
	updateFn func(ctx context.Context, serverID uint, nextScore int64) error
}

func (m *mockScoreUpdater) Update(ctx context.Context, serverID uint, nextScore int64) error {
	if m.updateFn == nil {
		return nil
	}
	return m.updateFn(ctx, serverID, nextScore)
}

var _ serverProvider = (*mockServerProvider)(nil)
var _ scoreUpdater = (*mockScoreUpdater)(nil)
