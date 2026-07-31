package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

type ServerReader interface {
	ListServers(ctx context.Context, createdByID uint, page, perPage int) ([]dto.Server, int64, error)
	GetServer(ctx context.Context, id uint) (*dto.Server, error)
	SearchServers(ctx context.Context, params dto.SearchParams, createdByID uint) ([]dto.Server, int64, error)
	CountByStatus(ctx context.Context, userID uint) (total, online, offline int64, err error)
}

type ServerWriter interface {
	CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error)
	UpdateServer(ctx context.Context, id uint, userID uint, req dto.UpdateServerRequest) (*dto.Server, error)
	DeleteServer(ctx context.Context, id uint, userID uint) error
}

type K8sObjectService interface {
	CreateK8sObject(ctx context.Context, req dto.CreateK8sObjectRequest, createdByID uint) (*dto.Server, error)
	DeleteK8sObject(ctx context.Context, req dto.DeleteK8sObjectRequest) error
}

type EndpointService interface {
	TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error)
}
