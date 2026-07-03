package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

type ServerService interface {
	ListServers(ctx context.Context, createdByID uint, page, perPage int) ([]dto.Server, error)
	CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error)
	GetServer(ctx context.Context, id uint) (*dto.Server, error)
	UpdateServer(ctx context.Context, id uint, userID uint, req dto.UpdateServerRequest) (*dto.Server, error)
	DeleteServer(ctx context.Context, id uint, userID uint) error
	SearchServers(ctx context.Context, params dto.SearchParams, createdByID uint) ([]dto.Server, int64, error)
}

type EndpointService interface {
	SetCheckMethod(ctx context.Context, serverID uint, req dto.SetCheckMethodRequest) error
	TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error)
}
