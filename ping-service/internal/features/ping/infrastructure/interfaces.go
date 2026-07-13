package infrastructure

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type EventSaver interface {
	Save(ctx context.Context, event *domain.ServerEvent) error
	GetLatestStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error)
}

type StatusStore interface {
	GetStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error)
	SetStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error
}

type EndpointStatusUpdater interface {
	UpdateMonitorStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error
}
