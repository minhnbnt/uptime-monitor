package redis

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type mockServerEventHandler struct {
	onCreateFn func(ctx context.Context, server domain.Server) error
	onUpdateFn func(ctx context.Context, server domain.Server) error
	onDeleteFn func(ctx context.Context, id uint) error
}

func (m *mockServerEventHandler) OnCreate(ctx context.Context, server domain.Server) error {
	if m.onCreateFn == nil {
		return nil
	}
	return m.onCreateFn(ctx, server)
}

func (m *mockServerEventHandler) OnUpdate(ctx context.Context, server domain.Server) error {
	if m.onUpdateFn == nil {
		return nil
	}
	return m.onUpdateFn(ctx, server)
}

func (m *mockServerEventHandler) OnDelete(ctx context.Context, id uint) error {
	if m.onDeleteFn == nil {
		return nil
	}
	return m.onDeleteFn(ctx, id)
}

var _ ServerEventHandler = (*mockServerEventHandler)(nil)
