package repository

import (
	"context"
	"errors"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
)

type ServerHttpConfigRepository struct {
	db *gorm.DB
}

func NewServerHttpConfigRepository(db *gorm.DB) *ServerHttpConfigRepository {
	return &ServerHttpConfigRepository{db: db}
}

func RegisterServerHttpConfigRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerHttpConfigRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return NewServerHttpConfigRepository(dbWrapper.GetDB()), nil
	})
}

func (r *ServerHttpConfigRepository) Upsert(ctx context.Context, cfg *domain.ServerHttpConfig) error {

	clauses := clause.OnConflict{
		Columns:   []clause.Column{{Name: "server_id"}},
		UpdateAll: true,
	}

	result := r.db.WithContext(ctx).Clauses(clauses).Create(cfg)

	return result.Error
}

func (r *ServerHttpConfigRepository) GetByServerID(ctx context.Context, serverID uint) (*domain.ServerHttpConfig, error) {

	cfg := new(domain.ServerHttpConfig)
	err := r.db.WithContext(ctx).
		Where("server_id = ?", serverID).
		First(cfg).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrNotFound
	}

	return cfg, err
}

func (r *ServerHttpConfigRepository) GetByServerIDs(ctx context.Context, serverIDs []uint) (map[uint]domain.ServerHttpConfig, error) {

	cfgs, err := gorm.G[domain.ServerHttpConfig](r.db).
		Where("server_id IN ?", serverIDs).
		Find(ctx)

	if err != nil {
		return nil, err
	}

	results := lo.SliceToMap(cfgs, func(cfg domain.ServerHttpConfig) (uint, domain.ServerHttpConfig) {
		return cfg.ServerID, cfg
	})

	return results, nil
}

func (r *ServerHttpConfigRepository) DeleteByServerID(ctx context.Context, serverID uint) error {

	rowAffected, err := gorm.G[domain.ServerHttpConfig](r.db).
		Where("server_id = ?", serverID).
		Delete(ctx)

	if err != nil {
		return err
	}

	if rowAffected == 0 {
		return apperrors.ErrNotFound
	}

	return nil
}
