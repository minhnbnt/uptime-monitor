package service

import (
	"context"
	"io"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/features/ping/repository"
)

type UserRepository interface {
	FindByID(ctx context.Context, id uint) (*domain.User, error)
}

type EventRepository interface {
	GetEnrichedEventsByUser(ctx context.Context, userID uint, from, to time.Time) ([]monitorrepo.EnrichedEvent, error)
}

type MailSender interface {
	Send(to, subject string, attachment io.Reader) error
}

type NotificationConfigRepository interface {
	GetByUserID(ctx context.Context, userID uint) (*domain.NotificationConfig, error)
}
