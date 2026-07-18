package handler

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
)

type ServerServer struct {
	serverv1.UnsafeServerServiceServer
	serverService *service.ServerService
	batchService  *service.ServerBatchService
	serverRepo    *repository.ServerRepository
	logger        *slog.Logger
}

func NewServerServer(serverService *service.ServerService, batchService *service.ServerBatchService, serverRepo *repository.ServerRepository, logger *slog.Logger) *ServerServer {
	return &ServerServer{
		serverService: serverService,
		batchService:  batchService,
		serverRepo:    serverRepo,
		logger:        logger,
	}
}

func RegisterServerServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerServer, error) {
		return NewServerServer(
			do.MustInvoke[*service.ServerService](i),
			do.MustInvoke[*service.ServerBatchService](i),
			do.MustInvoke[*repository.ServerRepository](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (s *ServerServer) GetServer(
	ctx context.Context,
	req *serverv1.GetServerRequest,
) (*serverv1.GetServerResponse, error) {

	server, err := s.serverRepo.GetByID(ctx, uint(req.ServerId))
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	if server.CreatedByID != uint(req.UserId) {
		return nil, fmt.Errorf("server not found")
	}

	return &serverv1.GetServerResponse{
		Server: &serverv1.ServerBrief{
			Id:        uint64(server.ID),
			Name:      server.Name,
			CreatedAt: server.CreatedAt.UnixMilli(),
		},
	}, nil
}

func (s *ServerServer) ListServers(
	ctx context.Context,
	req *serverv1.ListServersRequest,
) (*serverv1.ListServersResponse, error) {

	limit, offset := int(req.PerPage), int((req.Page-1)*req.PerPage)
	servers, err := s.serverRepo.List(ctx, uint(req.UserId), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}

	resp := &serverv1.ListServersResponse{}
	resp.Servers = lo.Map(servers, func(sv domain.Server, _ int) *serverv1.ServerBrief {
		return &serverv1.ServerBrief{
			Id:        uint64(sv.ID),
			Name:      sv.Name,
			CreatedAt: sv.CreatedAt.UnixMilli(),
		}
	})

	return resp, nil
}

func (s *ServerServer) SearchServers(
	ctx context.Context,
	req *serverv1.SearchServersRequest,
) (*serverv1.SearchServersResponse, error) {

	params := dto.SearchParams{
		Q:         req.Q,
		From:      int(req.From),
		To:        int(req.To),
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
	}

	servers, _, err := s.serverService.SearchServers(ctx, params, uint(req.UserId))
	if err != nil {
		s.logger.Error("search servers failed", slog.Any("error", err))
		return nil, fmt.Errorf("search servers: %w", err)
	}

	resp := &serverv1.SearchServersResponse{}
	resp.Servers = lo.Map(servers, func(sv dto.Server, _ int) *serverv1.ServerWithEndpoint {
		return mapServerToProto(sv)
	})

	return resp, nil
}

func (s *ServerServer) BatchCreateServers(
	ctx context.Context,
	req *serverv1.BatchCreateServersRequest,
) (*serverv1.BatchCreateServersResponse, error) {

	results, err := s.batchService.BatchCreateServers(ctx, req.Servers)
	if err != nil {
		return nil, fmt.Errorf("batch create servers: %w", err)
	}

	return &serverv1.BatchCreateServersResponse{Results: results}, nil
}

func mapServerToProto(sv dto.Server) *serverv1.ServerWithEndpoint {

	p := &serverv1.ServerWithEndpoint{
		Id:        uint64(sv.ID),
		Name:      sv.Name,
		CreatedAt: sv.CreatedAt.UnixMilli(),
	}

	if sv.Endpoint != nil {
		p.Url = sv.Endpoint.URL
		p.Method = sv.Endpoint.Method
		p.ExpectedCode = int32(sv.Endpoint.ExpectedCode)
		p.IntervalMs = sv.Endpoint.Interval.Milliseconds()
		p.TimeoutMs = sv.Endpoint.Timeout.Milliseconds()
	}

	return p
}
