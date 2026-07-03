package service

import (
	"context"
	"errors"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type ServerService struct {
	serverRepository   ServerRepository
	searchRepository   ServerSearchRepository
	endpointRepository EndpointRepository
	logger             logger.Logger
}

func RegisterServerService(i do.Injector) {

	do.Provide(i, func(i do.Injector) (*ServerService, error) {
		return &ServerService{
			serverRepository:   do.MustInvoke[*serverrepo.ServerRepository](i),
			searchRepository:   do.MustInvoke[*serverrepo.ParadeDBSearcher](i),
			endpointRepository: do.MustInvoke[*serverrepo.EndpointRepository](i),
			logger:             do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (ss *ServerService) ListServers(ctx context.Context, createdByID uint, page, perPage int) ([]dto.Server, error) {

	limit, offset := perPage, (page-1)*perPage
	result, err := ss.serverRepository.List(ctx, createdByID, limit, offset)
	if err != nil {
		ss.logger.Error("failed to get servers", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	return lo.Map(result, func(item domain.Server, index int) dto.Server {
		return dto.ServerFromDomain(item)
	}), nil
}

func (ss *ServerService) CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error) {

	server := domain.Server{
		Name:        req.Name,
		CreatedByID: createdByID,
	}

	if err := ss.serverRepository.Create(ctx, &server); err != nil {
		ss.logger.Error("failed to create server", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(server)
	return &result, nil
}

func (ss *ServerService) GetServer(ctx context.Context, id uint) (*dto.Server, error) {

	server, err := ss.serverRepository.GetByID(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to get server", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(*server)
	return &result, nil
}

func (ss *ServerService) UpdateServer(ctx context.Context, id uint, req dto.UpdateServerRequest) (*dto.Server, error) {

	server, err := ss.serverRepository.GetByID(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to get server for update", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	if req.Name != nil {
		server.Name = *req.Name
	}

	updateErr := ss.serverRepository.Update(ctx, server)
	if errors.Is(updateErr, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if updateErr != nil {
		ss.logger.Error("failed to update server", logger.Error(updateErr))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(*server)
	return &result, nil
}

func (ss *ServerService) DeleteServer(ctx context.Context, id uint) error {

	err := ss.serverRepository.Delete(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to delete server", logger.Error(err))
		return apperrors.ErrInternal
	}

	if err := ss.endpointRepository.DeleteByServerID(ctx, id); err != nil {
		ss.logger.Error("failed to clean up endpoint resources", logger.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}

func (ss *ServerService) SearchServers(
	ctx context.Context, params dto.SearchParams, createdByID uint,
) ([]dto.Server, int64, error) {

	servers, total, err := ss.searchRepository.Search(ctx, params, createdByID)
	if err != nil {
		ss.logger.Error("failed to search servers", logger.Error(err))
		return nil, 0, apperrors.ErrInternal
	}

	result := lo.Map(servers, func(item domain.Server, _ int) dto.Server {
		return dto.ServerFromDomain(item)
	})

	return result, total, nil
}
