package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type mockPingService struct {
	pingFn   func(ctx context.Context, method, url string) (int, error)
	recordFn func(ctx context.Context, event *domain.ServerEvent) error
}

func (m *mockPingService) Ping(ctx context.Context, method, url string) (int, error) {
	return m.pingFn(ctx, method, url)
}

func (m *mockPingService) Record(ctx context.Context, event *domain.ServerEvent) error {
	return m.recordFn(ctx, event)
}
