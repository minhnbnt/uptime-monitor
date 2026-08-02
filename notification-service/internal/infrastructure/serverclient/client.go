package serverclient

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type Client struct {
	client serverv1.ServerServiceClient
	logger *slog.Logger
}

func RegisterClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Client, error) {

		wrapper := do.MustInvoke[*config.GRPCClientWrapper](i)

		return &Client{
			client: serverv1.NewServerServiceClient(wrapper.GetConn()),
			logger: do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (a *Client) List(ctx context.Context, createdByID uuid.UUID, limit, offset int) ([]domain.Server, error) {

	req := serverv1.ListServersRequest{
		UserId:  createdByID.String(),
		Page:    int32(offset/limit) + 1,
		PerPage: int32(limit),
	}

	a.logger.Debug(
		"serverclient.List: sending gRPC request",
		slog.String("user_id", createdByID.String()),
		slog.Int("limit", limit),
		slog.Int("offset", offset),
	)

	resp, err := a.client.ListServers(ctx, &req)
	if err != nil {
		a.logger.Error(
			"serverclient.List: gRPC call failed",
			slog.String("user_id", createdByID.String()),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("list servers: %w", err)
	}

	servers := make([]domain.Server, 0, len(resp.Servers))
	for _, s := range resp.Servers {
		servers = append(servers, domain.Server{ID: uint(s.Id), Name: s.Name})
	}

	return servers, nil
}

func (a *Client) CountByStatus(ctx context.Context, createdByID uuid.UUID) (total, online, offline int64, err error) {

	req := serverv1.CountServersByStatusRequest{UserId: createdByID.String()}
	a.logger.Debug(
		"serverclient.CountByStatus: sending gRPC request",
		slog.String("user_id", createdByID.String()),
	)

	resp, err := a.client.CountServersByStatus(ctx, &req)
	if err != nil {
		a.logger.Error(
			"serverclient.CountByStatus: gRPC call failed",
			slog.String("user_id", createdByID.String()),
			slog.Any("error", err),
		)
		return 0, 0, 0, fmt.Errorf("count servers by status: %w", err)
	}

	return resp.Total, resp.Online, resp.Offline, nil
}
