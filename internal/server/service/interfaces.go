package service

import (
	"context"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

type OntimeCacheRepository interface {
	MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error)
	MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]float64) error
}

type NotificationConfigRepository interface {
	GetByUserID(ctx context.Context, userID uint) (*domain.NotificationConfig, error)
	Upsert(ctx context.Context, cfg *domain.NotificationConfig) error
}

type DigestStarter interface {
	StartDigest(ctx context.Context, userID uint) error
	UpsertSchedule(ctx context.Context, userID uint, fromDate, toDate time.Time, digestTime string) error
	DeleteSchedule(ctx context.Context, userID uint) error
}
