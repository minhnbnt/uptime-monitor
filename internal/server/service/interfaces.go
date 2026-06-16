package service

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

type ServerRepository interface {
	Count(ctx context.Context, createdByID uint) (int64, error)
	List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
	Create(ctx context.Context, s *domain.Server) error
	GetByID(ctx context.Context, id uint) (*domain.Server, error)
	Update(ctx context.Context, s *domain.Server) error
	Delete(ctx context.Context, id uint) error
	BatchGetOntime(ctx context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error)
	BatchCreateServers(ctx context.Context, servers []domain.Server) error
}

type EndpointRepository interface {
	UpsertEndpoint(ctx context.Context, endpoint domain.Endpoint) error
	DeleteByServerID(ctx context.Context, serverID uint) error
	BatchCreateEndpoints(ctx context.Context, endpoints []domain.Endpoint) error
}

type ServerSearchRepository interface {
	Search(ctx context.Context, params dto.SearchParams, createdByID uint) ([]domain.Server, int64, error)
}

type OntimeCacheRepository interface {
	MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error)
	MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]float64) error
}
