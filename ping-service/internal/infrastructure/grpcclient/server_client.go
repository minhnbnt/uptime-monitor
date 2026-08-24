package grpcclient

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
)

type ServerClient struct {
	client serverv1.ServerServiceClient
}

func RegisterServerClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerClient, error) {

		conn := do.MustInvoke[*config.GRPCClientWrapper](i)
		client := serverv1.NewServerServiceClient(conn.GetConn())

		return &ServerClient{client: client}, nil
	})
}

func (c *ServerClient) ResolveServers(ctx context.Context, userID uint, ids []uint64) ([]uint64, error) {

	request := &serverv1.ResolveServersRequest{
		UserId: uint64(userID), Ids: ids,
	}

	resp, err := c.client.ResolveServers(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("resolve servers: %w", err)
	}

	return resp.GetIds(), nil
}
