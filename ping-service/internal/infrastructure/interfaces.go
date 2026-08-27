package infrastructure

import (
	"context"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type EventRecorder interface {
	RecordEvent(ctx context.Context, endpointID uint, status domain.ServerStatus) error
	RecordEventAt(ctx context.Context, endpointID uint, status domain.ServerStatus, recordedAt time.Time) error
}

type StatusStore interface {
	GetStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error)
	SetStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error
}
