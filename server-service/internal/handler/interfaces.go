package handler

import (
	"context"

	"github.com/google/uuid"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

type ServerService interface {
	ListServers(ctx context.Context, createdByID uuid.UUID, page, perPage int) ([]dto.Server, int64, error)
	CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uuid.UUID) (*dto.Server, error)
	GetServer(ctx context.Context, id uint) (*dto.Server, error)
	UpdateServer(ctx context.Context, id uint, userID uuid.UUID, req dto.UpdateServerRequest) (*dto.Server, error)
	DeleteServer(ctx context.Context, id uint, userID uuid.UUID) error
	SearchServers(ctx context.Context, params dto.SearchParams, createdByID uuid.UUID) ([]dto.Server, int64, error)
	CountByStatus(ctx context.Context, userID uuid.UUID) (total, online, offline int64, err error)
}

type EndpointService interface {
	SetCheckMethod(ctx context.Context, serverID uint, userID uuid.UUID, req dto.SetCheckMethodRequest) error
	TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error)
}
