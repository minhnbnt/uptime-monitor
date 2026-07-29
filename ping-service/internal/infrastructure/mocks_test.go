package infrastructure

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type mockStatusStore struct {
	getStatusFn func(ctx context.Context, serverID uint) (domain.ServerStatus, error)
	setStatusFn func(ctx context.Context, serverID uint, status domain.ServerStatus) error
}

func (m *mockStatusStore) GetStatus(ctx context.Context, serverID uint) (domain.ServerStatus, error) {
	return m.getStatusFn(ctx, serverID)
}

func (m *mockStatusStore) SetStatus(ctx context.Context, serverID uint, status domain.ServerStatus) error {
	if m.setStatusFn == nil {
		return nil
	}
	return m.setStatusFn(ctx, serverID, status)
}

type mockEventRecorder struct {
	recordEventFn func(ctx context.Context, serverID uint, status domain.ServerStatus) error
}

func (m *mockEventRecorder) RecordEvent(ctx context.Context, serverID uint, status domain.ServerStatus) error {
	if m.recordEventFn == nil {
		return nil
	}
	return m.recordEventFn(ctx, serverID, status)
}

var _ StatusStore = (*mockStatusStore)(nil)
var _ EventRecorder = (*mockEventRecorder)(nil)
