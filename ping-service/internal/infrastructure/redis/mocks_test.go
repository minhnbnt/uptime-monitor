package redis

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type mockEndpointEventHandler struct {
	onCreateFn func(ctx context.Context, endpoint domain.Endpoint) error
	onUpdateFn func(ctx context.Context, endpoint domain.Endpoint) error
	onDeleteFn func(ctx context.Context, id uint) error
}

func (m *mockEndpointEventHandler) OnCreate(ctx context.Context, endpoint domain.Endpoint) error {
	if m.onCreateFn == nil {
		return nil
	}
	return m.onCreateFn(ctx, endpoint)
}

func (m *mockEndpointEventHandler) OnUpdate(ctx context.Context, endpoint domain.Endpoint) error {
	if m.onUpdateFn == nil {
		return nil
	}
	return m.onUpdateFn(ctx, endpoint)
}

func (m *mockEndpointEventHandler) OnDelete(ctx context.Context, id uint) error {
	if m.onDeleteFn == nil {
		return nil
	}
	return m.onDeleteFn(ctx, id)
}

var _ EndpointEventHandler = (*mockEndpointEventHandler)(nil)
