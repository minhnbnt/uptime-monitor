package repository

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
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
