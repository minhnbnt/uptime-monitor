package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

type ServerWriter interface {
	Create(ctx context.Context, s *domain.Server) error
	Update(ctx context.Context, s *domain.Server) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*domain.Server, error)
}

type ServerService struct {
	*ServerReader
	serverWriter       ServerWriter
	endpointRepository EndpointRepository
}

func RegisterServerService(i do.Injector) {

	do.Provide(i, func(i do.Injector) (*ServerService, error) {
		return &ServerService{
			ServerReader:       do.MustInvoke[*ServerReader](i),
			serverWriter:       do.MustInvoke[*repository.ServerRepository](i),
			endpointRepository: do.MustInvoke[*repository.EndpointRepository](i),
		}, nil
	})
}

func (ss *ServerService) CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error) {

	server := domain.Server{
		Name:        req.Name,
		CreatedByID: createdByID,
	}

	if err := ss.serverWriter.Create(ctx, &server); err != nil {
		ss.logger.Error("failed to create server", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(server)
	return &result, nil
}

func (ss *ServerService) UpdateServer(ctx context.Context, id uint, userID uint, req dto.UpdateServerRequest) (*dto.Server, error) {

	server, err := ss.serverWriter.GetByID(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to get server for update", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	if server.CreatedByID != userID {
		return nil, apperrors.ErrForbidden
	}

	if req.Name != nil {
		server.Name = *req.Name
	}

	updateErr := ss.serverWriter.Update(ctx, server)
	if errors.Is(updateErr, apperrors.ErrNotFound) {
		return nil, apperrors.ErrNotFound
	}

	if updateErr != nil {
		ss.logger.Error("failed to update server", slog.Any("error", updateErr))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(*server)
	return &result, nil
}

func (ss *ServerService) DeleteServer(ctx context.Context, id uint, userID uint) error {

	server, err := ss.serverWriter.GetByID(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to get server", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	if server.CreatedByID != userID {
		return apperrors.ErrForbidden
	}

	err = ss.serverWriter.Delete(ctx, id)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrNotFound
	}
	if err != nil {
		ss.logger.Error("failed to delete server", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	if err := ss.endpointRepository.DeleteByServerID(ctx, id); err != nil {
		ss.logger.Error("failed to clean up endpoint resources", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	return nil
}
