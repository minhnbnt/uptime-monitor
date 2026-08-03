package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	k8sclient "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/k8s"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
)

type K8sObjectService struct {
	serverWriter ServerWriter
	serverRepo   *repository.ServerRepository
	k8sClient    *k8sclient.K8sClient
	logger       *slog.Logger
}

func RegisterK8sObjectService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*K8sObjectService, error) {
		reader := do.MustInvoke[*ServerReader](i)
		return &K8sObjectService{
			serverWriter: do.MustInvoke[*repository.ServerRepository](i),
			serverRepo:   do.MustInvoke[*repository.ServerRepository](i),
			k8sClient:    do.MustInvoke[*k8sclient.K8sClient](i),
			logger:       reader.logger,
		}, nil
	})
}

func (s *K8sObjectService) CreateK8sObject(
	ctx context.Context,
	req dto.CreateK8sObjectRequest,
	createdByID uuid.UUID,
) (*dto.Server, error) {

	server := domain.Server{
		Name:          req.Name,
		Namespace:     req.Namespace,
		Kind:          "Pod",
		ObjectID:      req.ObjectID,
		ContainerName: req.ContainerName,
		Interval:      req.Interval,
		Timeout:       req.Timeout,
		CreatedByID:   createdByID,
		Managed:       true,
	}

	var config *domain.ServerHttpConfig
	if req.HttpConfig != nil {
		config = &domain.ServerHttpConfig{
			Port:          req.HttpConfig.Port,
			EndpointPath:  req.HttpConfig.EndpointPath,
			ExpectedCode:  req.HttpConfig.ExpectedCode,
			BodyCheckExpr: req.HttpConfig.BodyCheckExpr,
			Method:        defaultHttpMethod(req.HttpConfig.Method),
		}
	}

	if err := s.k8sClient.CreatePod(ctx, req.Namespace, req.ObjectID, toK8sContainers(req.Containers)); err != nil {
		s.logger.Error("failed to create pod", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	if err := s.serverWriter.Create(ctx, &server, config); err != nil {
		s.logger.Error("failed to create server", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	result := dto.ServerFromDomain(server)
	return &result, nil
}

func (s *K8sObjectService) DeleteK8sObject(ctx context.Context, userID uuid.UUID, req dto.DeleteK8sObjectRequest) error {

	server, err := s.serverRepo.GetByNamespaceObjectIDUnscoped(ctx, req.Namespace, req.ObjectID)
	if errors.Is(err, apperrors.ErrNotFound) {
		return apperrors.ErrNotFound
	}
	if err != nil {
		s.logger.Error("failed to get server by namespace/object_id", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	if server.CreatedByID != userID {
		return apperrors.ErrForbidden
	}

	if !server.Managed {
		return apperrors.ErrNotManaged
	}

	if err := s.k8sClient.DeletePod(ctx, req.Namespace, req.ObjectID); err != nil {
		s.logger.Error("failed to delete pod", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	return nil
}

func toK8sContainers(containers []dto.Container) []k8sclient.Container {
	out := make([]k8sclient.Container, 0, len(containers))
	for _, ctr := range containers {
		out = append(out, k8sclient.Container{Name: ctr.Name, Image: ctr.Image})
	}
	return out
}
