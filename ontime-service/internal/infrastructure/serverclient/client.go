package serverclient

import (
	"context"
	"fmt"
	"time"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/proto/gen/server/v1"
	"github.com/samber/do/v2"

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

func (c *Client) GetServer(ctx context.Context, serverID, userID uint) (*ServerBrief, error) {
	resp, err := c.client.GetServer(ctx, &serverv1.GetServerRequest{
		UserId:   uint64(userID),
		ServerId: uint64(serverID),
	})
	if err != nil {
		return nil, fmt.Errorf("get server: %w", err)
	}

	return &ServerBrief{
		ID:        uint(resp.Server.Id),
		Name:      resp.Server.Name,
		CreatedAt: time.UnixMilli(resp.Server.CreatedAt),
	}, nil
}

func (c *Client) ListServers(ctx context.Context, userID uint, page, perPage int) ([]ServerBrief, error) {
	resp, err := c.client.ListServers(ctx, &serverv1.ListServersRequest{
		UserId:  uint64(userID),
		Page:    int32(page),
		PerPage: int32(perPage),
	})
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
