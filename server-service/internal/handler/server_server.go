package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
)

type ServerServer struct {
	serverv1.UnsafeServerServiceServer
	serverService *service.ServerService
	batchService  *service.ServerBatchService
	logger        *slog.Logger
}

func NewServerServer(
	serverService *service.ServerService,
	batchService *service.ServerBatchService,
	logger *slog.Logger,
) *ServerServer {
	return &ServerServer{
		serverService: serverService,
		batchService:  batchService,
		logger:        logger,
	}
}

func RegisterServerServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerServer, error) {
		return NewServerServer(
			do.MustInvoke[*service.ServerService](i),
			do.MustInvoke[*service.ServerBatchService](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (s *ServerServer) GetServer(
	ctx context.Context,
	req *serverv1.GetServerRequest,
) (*serverv1.GetServerResponse, error) {

	server, err := s.serverService.GetServer(ctx, uint(req.ServerId))
	if errors.Is(err, apperrors.ErrNotFound) {
		return nil, fmt.Errorf("server not found")
	}
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	if server.CreatedByID != uint(req.UserId) {
		return nil, fmt.Errorf("server not found")
	}

	return &serverv1.GetServerResponse{
		Server: serverBriefFromDTO(*server),
	}, nil
}

func (s *ServerServer) ListServers(
	ctx context.Context,
	req *serverv1.ListServersRequest,
) (*serverv1.ListServersResponse, error) {

	servers, _, err := s.serverService.ListServers(
		ctx, uint(req.UserId),
		int(req.Page), int(req.PerPage),
	)

	if err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}

	resp := &serverv1.ListServersResponse{}
	resp.Servers = lo.Map(servers, func(sv dto.Server, _ int) *serverv1.ServerBrief {
		return serverBriefFromDTO(sv)
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
	resp.Servers = lo.Map(
		servers,
		func(sv dto.Server, _ int) *serverv1.ServerWithEndpoint {
			return mapServerToProto(sv)
		},
	)

	return resp, nil
}

func (s *ServerServer) CountServersByStatus(
	ctx context.Context,
	req *serverv1.CountServersByStatusRequest,
) (*serverv1.CountServersByStatusResponse, error) {

	total, online, offline, err := s.serverService.CountByStatus(ctx, uint(req.UserId))
	if err != nil {
		s.logger.Error("count servers by status failed", slog.Any("error", err))
		return nil, fmt.Errorf("count servers by status: %w", err)
	}

	return &serverv1.CountServersByStatusResponse{
		Total:   total,
		Online:  online,
		Offline: offline,
	}, nil
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

func serverBriefFromDTO(sv dto.Server) *serverv1.ServerBrief {
	return &serverv1.ServerBrief{
		Id:        uint64(sv.ID),
		Name:      sv.Name,
		CreatedAt: sv.CreatedAt.UnixMilli(),
	}
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
