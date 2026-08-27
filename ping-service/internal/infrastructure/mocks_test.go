package infrastructure

import (
	"context"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type mockStatusStore struct {
	getStatusFn func(ctx context.Context, endpointID uint) (domain.ServerStatus, error)
	setStatusFn func(ctx context.Context, endpointID uint, status domain.ServerStatus) error
}

func (m *mockStatusStore) GetStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error) {
	return m.getStatusFn(ctx, endpointID)
}

func (m *mockStatusStore) SetStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error {
	if m.setStatusFn == nil {
		return nil
	}
	return m.setStatusFn(ctx, endpointID, status)
}

type mockEventRecorder struct {
	recordEventFn   func(ctx context.Context, endpointID uint, status domain.ServerStatus) error
	recordEventAtFn func(ctx context.Context, endpointID uint, status domain.ServerStatus, recordedAt time.Time) error
}

func (m *mockEventRecorder) RecordEvent(ctx context.Context, endpointID uint, status domain.ServerStatus) error {
	if m.recordEventFn == nil {
		return nil
	}
	return m.recordEventFn(ctx, endpointID, status)
}

func (m *mockEventRecorder) RecordEventAt(ctx context.Context, endpointID uint, status domain.ServerStatus, recordedAt time.Time) error {
	if m.recordEventAtFn != nil {
		return m.recordEventAtFn(ctx, endpointID, status, recordedAt)
	}
	if m.recordEventFn != nil {
		return m.recordEventFn(ctx, endpointID, status)
	}
	return nil
}

var _ StatusStore = (*mockStatusStore)(nil)
var _ EventRecorder = (*mockEventRecorder)(nil)
