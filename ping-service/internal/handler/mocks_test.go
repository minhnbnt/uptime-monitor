package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type mockPingService struct {
	pingFn   func(ctx context.Context, ep *domain.Endpoint) (bool, error)
	recordFn func(ctx context.Context, event *domain.ServerEvent) error
}

func (m *mockPingService) Ping(ctx context.Context, ep *domain.Endpoint) (bool, error) {
	if m.pingFn == nil {
		return false, nil
	}
	return m.pingFn(ctx, ep)
}

func (m *mockPingService) Record(ctx context.Context, event *domain.ServerEvent) error {
	if m.recordFn == nil {
		return nil
	}
	return m.recordFn(ctx, event)
}

var _ PingService = (*mockPingService)(nil)
