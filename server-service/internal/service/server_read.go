package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/grpcclient"
	serverrepo "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

type StatusClient interface {
	GetCurrentStatuses(ctx context.Context, endpointIDs []uint) (map[uint]domain.ServerStatus, error)
	CountByStatus(ctx context.Context, endpointIDs []uint) (online, offline int64, err error)
}

type ServerRepository interface {
	List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
	Count(ctx context.Context, createdByID uint) (int64, error)
	GetByID(ctx context.Context, id uint) (*domain.Server, error)
}

type ServerSearchRepository interface {
	Search(ctx context.Context, params dto.SearchParams, createdByID uint) ([]domain.Server, int64, error)
}

type ServerReader struct {
	serverRepository ServerRepository
	searchRepository ServerSearchRepository
	statusClient     StatusClient
	logger           *slog.Logger
}

func NewServerReader(
	serverRepository ServerRepository,
	searchRepository ServerSearchRepository,
	statusClient StatusClient,
	logger *slog.Logger,
) *ServerReader {
	return &ServerReader{
		serverRepository: serverRepository,
		searchRepository: searchRepository,
		statusClient:     statusClient,
		logger:           logger,
	}
}

func RegisterServerReader(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerReader, error) {
		return NewServerReader(
			do.MustInvoke[*serverrepo.ServerRepository](i),
			do.MustInvoke[*serverrepo.ParadeDBSearcher](i),
			do.MustInvoke[grpcclient.StatusClient](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (r *ServerReader) ListServers(
	ctx context.Context,
	createdByID uint,
	page, perPage int,
) ([]dto.Server, int64, error) {

	limit, offset := perPage, (page-1)*perPage
	result, err := r.serverRepository.List(ctx, createdByID, limit, offset)
	if err != nil {
		r.logger.Error("failed to get servers", slog.Any("error", err))
		return nil, 0, apperrors.ErrInternal
	}

	total, err := r.serverRepository.Count(ctx, createdByID)
	if err != nil {
		r.logger.Error("failed to count servers", slog.Any("error", err))
		return nil, 0, apperrors.ErrInternal
	}

	servers := lo.Map(result, func(item domain.Server, _ int) *dto.Server {
		return new(dto.ServerFromDomain(item))
	})

	r.applyStatuses(ctx, servers)

	out := lo.Map(servers, func(s *dto.Server, _ int) dto.Server {
		return *s
	})

	return out, total, nil
}

func (r *ServerReader) GetServer(ctx context.Context, id uint) (*dto.Server, error) {

	server, err := r.serverRepository.GetByID(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		r.logger.Error("failed to get server", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(*server)
	r.applyStatuses(ctx, []*dto.Server{&result})

	return &result, nil
}

func (r *ServerReader) SearchServers(
	ctx context.Context,
	params dto.SearchParams,
	createdByID uint,
) ([]dto.Server, int64, error) {

	servers, total, err := r.searchRepository.Search(ctx, params, createdByID)
	if err != nil {
		r.logger.Error("failed to search servers", slog.Any("error", err))
		return nil, 0, apperrors.ErrInternal
	}

	mapped := lo.Map(servers, func(item domain.Server, _ int) *dto.Server {
		return new(dto.ServerFromDomain(item))
	})

	r.applyStatuses(ctx, mapped)

	out := lo.Map(mapped, func(s *dto.Server, _ int) dto.Server {
		return *s
	})

	return out, total, nil
}

func (r *ServerReader) applyStatuses(ctx context.Context, servers []*dto.Server) {

	serversMap := make(map[uint]*dto.Server, len(servers))
	for _, server := range servers {
		if server != nil && server.Endpoint != nil {
			serversMap[server.Endpoint.ID] = server
		}
	}

	endpointIDs := lo.Keys(serversMap)
	if len(endpointIDs) == 0 {
		return
	}

	statuses, err := r.statusClient.GetCurrentStatuses(ctx, endpointIDs)
	if err != nil {
		r.logger.Warn(
			"failed to get current statuses, returning without monitor_status",
			slog.Any("error", err),
		)
		return
	}

	for endpointID, status := range statuses {
		if server, ok := serversMap[endpointID]; ok {
			server.MonitorStatus = status
		}
	}
}
