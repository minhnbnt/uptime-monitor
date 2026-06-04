package infrashtructure

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type EventSaver interface {
	Save(ctx context.Context, event *domain.ServerEvent) error
}

type StatusStore interface {
	GetStatus(ctx context.Context, endpointID uint) (domain.ServerStatus, error)
	SetStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error
}
