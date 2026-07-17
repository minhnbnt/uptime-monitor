package handler

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/samber/lo/it"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
)

const batchChunkSize = 100

type ServerServer struct {
	serverv1.UnimplementedServerServiceServer
	serverService    *service.ServerService
	serverRepo       *repository.ServerRepository
	endpointRepo     *repository.EndpointRepository
	logger           *slog.Logger
}

func NewServerServer(serverService *service.ServerService, serverRepo *repository.ServerRepository, endpointRepo *repository.EndpointRepository, logger *slog.Logger) *ServerServer {
	return &ServerServer{
		serverService: serverService,
		serverRepo:    serverRepo,
		endpointRepo:  endpointRepo,
		logger:        logger,
	}
}

func RegisterServerServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerServer, error) {
		return NewServerServer(
			do.MustInvoke[*service.ServerService](i),
			do.MustInvoke[*repository.ServerRepository](i),
			do.MustInvoke[*repository.EndpointRepository](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (s *ServerServer) GetServer(ctx context.Context, req *serverv1.GetServerRequest) (*serverv1.GetServerResponse, error) {

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

func (s *ServerServer) ListServers(ctx context.Context, req *serverv1.ListServersRequest) (*serverv1.ListServersResponse, error) {

	limit, offset := int(req.PerPage), int((req.Page-1)*req.PerPage)
	servers, err := s.serverRepo.List(ctx, uint(req.UserId), limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}

	resp := &serverv1.ListServersResponse{
		Servers: make([]*serverv1.ServerBrief, 0, len(servers)),
	}

	for _, sv := range servers {
		resp.Servers = append(resp.Servers, &serverv1.ServerBrief{
			Id:        uint64(sv.ID),
			Name:      sv.Name,
			CreatedAt: sv.CreatedAt.UnixMilli(),
		})
	}

	return resp, nil
}

func (s *ServerServer) SearchServers(ctx context.Context, req *serverv1.SearchServersRequest) (*serverv1.SearchServersResponse, error) {

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

	resp := &serverv1.SearchServersResponse{
		Servers: lo.Map(servers, func(sv dto.Server, _ int) *serverv1.ServerWithEndpoint {
			return mapServerToProto(sv)
		}),
	}

	return resp, nil
}

func (s *ServerServer) BatchCreateServers(ctx context.Context, req *serverv1.BatchCreateServersRequest) (*serverv1.BatchCreateServersResponse, error) {

	var results []*serverv1.BatchCreateServerResult

	for chunk := range it.Chunk(slices.Values(req.Servers), batchChunkSize) {
		servers := buildDomainServers(chunk)
		if err := s.serverRepo.BatchCreateServers(ctx, servers); err != nil {
			s.logger.Error("batch create servers failed", slog.Any("error", err))
			for _, input := range chunk {
				results = append(results, &serverv1.BatchCreateServerResult{
					Row:   input.Row,
					Name:  input.Name,
					Url:   input.Url,
					Error: err.Error(),
				})
			}
			continue
		}

		for i, sv := range servers {
			results = append(results, &serverv1.BatchCreateServerResult{
				Row:       chunk[i].Row,
				Name:      sv.Name,
				Url:       chunk[i].Url,
				ServerId:  uint64(sv.ID),
			})
		}

		endpoints := buildDomainEndpoints(chunk, servers)
		if len(endpoints) == 0 {
			continue
		}

		if err := s.endpointRepo.BatchCreateEndpoints(ctx, endpoints); err != nil {
			s.logger.Error("batch create endpoints failed", slog.Any("error", err))
			for i := range servers {
				results[len(results)-len(servers)+i].Error = "endpoint creation failed"
			}
		}
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
		p.MonitorStatus = string(sv.Endpoint.MonitorStatus)
	}

	if p.MonitorStatus == "" {
		p.MonitorStatus = string(domain.StatusOff)
	}

	return p
}

func buildDomainServers(inputs []*serverv1.ServerWithEndpointInput) []domain.Server {
	return lo.Map(inputs, func(in *serverv1.ServerWithEndpointInput, _ int) domain.Server {
		return domain.Server{
			Name:        in.Name,
			CreatedByID: uint(in.UserId),
		}
	})
}

func buildDomainEndpoints(inputs []*serverv1.ServerWithEndpointInput, servers []domain.Server) []domain.Endpoint {
	endpoints := make([]domain.Endpoint, 0, len(servers))
	for i, sv := range servers {
		if inputs[i].Url == "" {
			continue
		}
		endpoints = append(endpoints, domain.Endpoint{
			ServerID:     sv.ID,
			URL:          inputs[i].Url,
			Interval:     time.Duration(inputs[i].IntervalMs) * time.Millisecond,
			Timeout:      time.Duration(inputs[i].TimeoutMs) * time.Millisecond,
			Method:       inputs[i].Method,
			ExpectedCode: int(inputs[i].ExpectedCode),
		})
	}
	return endpoints
}
