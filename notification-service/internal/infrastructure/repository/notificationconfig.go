package repository

import (
	"context"
	"errors"

	"github.com/samber/do/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type NotificationConfigRepository struct {
	db *gorm.DB
}

func RegisterNotificationConfigRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*NotificationConfigRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &NotificationConfigRepository{db: dbWrapper.GetDB()}, nil
	})
}

func (r *NotificationConfigRepository) GetByUserID(ctx context.Context, userID uint) (*domain.NotificationConfig, error) {

	cfg, err := gorm.G[domain.NotificationConfig](r.db).Where("user_id = ?", userID).First(ctx)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (r *NotificationConfigRepository) Upsert(ctx context.Context, cfg *domain.NotificationConfig) error {

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
