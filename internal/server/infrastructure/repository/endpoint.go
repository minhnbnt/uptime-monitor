package repository

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
)

type EndpointRepository struct {
	db *gorm.DB
}

func RegisterEndpointRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &EndpointRepository{db: dbWrapper.GetDB()}, nil
	})
}

func (er *EndpointRepository) GetByServerID(ctx context.Context, serverID uint) (*domain.Endpoint, error) {

	endpoint, err := gorm.G[domain.Endpoint](er.db).Where("server_id = ?", serverID).First(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to get endpoint: %w", err)
	}

	return &endpoint, nil
}

func (er *EndpointRepository) UpsertEndpoint(ctx context.Context, endpoint domain.Endpoint) error {
	return gorm.G[domain.Endpoint](er.db).Create(ctx, &endpoint)
}
