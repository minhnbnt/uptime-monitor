package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

type ServerReader interface {
	ListServers(ctx context.Context, createdByID uuid.UUID, page, perPage int) ([]dto.Server, int64, error)
	GetServer(ctx context.Context, id uint) (*dto.Server, error)
	SearchServers(ctx context.Context, params dto.SearchParams, createdByID uuid.UUID) ([]dto.Server, int64, error)
	CountByStatus(ctx context.Context, userID uuid.UUID) (total, online, offline int64, err error)
}

type ServerWriter interface {
	CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uuid.UUID) (*dto.Server, error)
	UpdateServer(ctx context.Context, id uint, userID uuid.UUID, req dto.UpdateServerRequest) (*dto.Server, error)
	DeleteServer(ctx context.Context, id uint, userID uuid.UUID) error
}

type K8sObjectService interface {
	CreateK8sObject(ctx context.Context, req dto.CreateK8sObjectRequest, createdByID uuid.UUID) (*dto.Server, error)
	DeleteK8sObject(ctx context.Context, req dto.DeleteK8sObjectRequest) error
}

type EndpointService interface {
	TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error)
}
