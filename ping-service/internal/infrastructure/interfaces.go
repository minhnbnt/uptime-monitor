package infrastructure

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type EventRecorder interface {
	RecordEvent(ctx context.Context, serverID uint, status domain.ServerStatus) error
}

type StatusStore interface {
	GetStatus(ctx context.Context, serverID uint) (domain.ServerStatus, error)
	SetStatus(ctx context.Context, serverID uint, status domain.ServerStatus) error
}
