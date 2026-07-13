package service

import (
	"context"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type NotificationConfigRepository interface {
	GetByUserID(ctx context.Context, userID uint) (*domain.NotificationConfig, error)
	Upsert(ctx context.Context, cfg *domain.NotificationConfig) error
}

type DigestStarter interface {
	StartDigest(ctx context.Context, userID uint) error
	UpsertSchedule(ctx context.Context, userID uint, fromDate, toDate time.Time, digestTime string) error
	DeleteSchedule(ctx context.Context, userID uint) error
}
