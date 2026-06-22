package service

import (
	"context"
	"io"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
)

type UserRepository interface {
	FindByID(ctx context.Context, id uint) (*domain.User, error)
}

type MailSender interface {
	Send(to, subject string, attachment io.Reader) error
}

type NotificationConfigRepository interface {
	GetByUserID(ctx context.Context, userID uint) (*domain.NotificationConfig, error)
}

type ServerLister interface {
	List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
}

type OntimeStatsService interface {
	GetServersOntimeForDates(ctx context.Context, servers []domain.Server, dates []time.Time) (map[uint][]ontimedto.OntimeStats, error)
}
