package repository

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type NotificationConfigRepository struct {
	db     *gorm.DB
	logger *slog.Logger
}

func RegisterNotificationConfigRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*NotificationConfigRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &NotificationConfigRepository{
			db:     dbWrapper.GetDB(),
			logger: do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (r *NotificationConfigRepository) GetByUserID(ctx context.Context, userID uint) (*domain.NotificationConfig, error) {

	cfg, err := gorm.G[domain.NotificationConfig](r.db).Where("user_id = ?", userID).First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		r.logger.Debug("NotificationConfigRepository.GetByUserID: no config found", slog.Uint64("user_id", uint64(userID)))
		return nil, nil
	}

	if err != nil {
		r.logger.Error("NotificationConfigRepository.GetByUserID: query failed",
			slog.Uint64("user_id", uint64(userID)), slog.Any("error", err))
		return nil, err
	}

	return &cfg, nil
}

func (r *NotificationConfigRepository) Upsert(_ context.Context, cfg *domain.NotificationConfig) error {

	queryClause := clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"active",
			"from_date",
			"to_date",
			"digest_time",
		}),
	}

	result := r.db.Clauses(queryClause).Create(cfg)

	return result.Error
}
