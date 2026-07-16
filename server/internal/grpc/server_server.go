package grpc

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
)

type ServerServer struct {
	serverv1.UnimplementedServerServiceServer
	serverRepo *repository.ServerRepository
}

func NewServerServer(serverRepo *repository.ServerRepository) *ServerServer {
	return &ServerServer{serverRepo: serverRepo}
}

func RegisterServerServer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerServer, error) {
		return NewServerServer(
			do.MustInvoke[*repository.ServerRepository](i),
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
