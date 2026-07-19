package serverclient

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
)

type ServerBrief struct {
	ID        uint
	Name      string
	CreatedAt time.Time
}

type Client struct {
	client serverv1.ServerServiceClient
}

func RegisterClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Client, error) {
		wrapper := do.MustInvoke[*config.GRPCClientWrapper](i)
		client := serverv1.NewServerServiceClient(wrapper.GetConn())
		return &Client{client: client}, nil
	})
}

func (c *Client) GetServer(
	ctx context.Context,
	serverID, userID uint,
) (*ServerBrief, error) {

	request := serverv1.GetServerRequest{
		ServerId: uint64(serverID),
		UserId:   uint64(userID),
	}

	resp, err := c.client.GetServer(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	return &ServerBrief{
		CreatedAt: time.UnixMilli(resp.Server.CreatedAt),
		ID:        uint(resp.Server.Id),
		Name:      resp.Server.Name,
	}, nil
}

func (c *Client) ListServers(
	ctx context.Context,
	userID uint,
	page, perPage int,
) ([]ServerBrief, error) {

	request := serverv1.ListServersRequest{
		Page:    int32(page),
		PerPage: int32(perPage),
		UserId:  uint64(userID),
	}

	resp, err := c.client.ListServers(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("list servers: %w", err)
	}

	servers := make([]ServerBrief, 0, len(resp.Servers))
	for _, s := range resp.Servers {
		servers = append(servers, ServerBrief{
			ID:        uint(s.Id),
			Name:      s.Name,
			CreatedAt: time.UnixMilli(s.CreatedAt),
		})
	}

	return servers, nil
}
