package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/ping/service"
)

type captureHandlerLoopRunner struct {
	capturedHandler service.DueHandler
}

func (c *captureHandlerLoopRunner) Run(_ context.Context, dueHandler service.DueHandler) error {
	c.capturedHandler = dueHandler
	return nil
}

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
