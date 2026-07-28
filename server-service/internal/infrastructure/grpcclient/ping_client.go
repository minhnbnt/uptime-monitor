package grpcclient

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"

	pingv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/ping/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
)

type PingClient struct {
	client pingv1.PingServiceClient
}

func NewPingClient(cc *config.GRPCClientWrapper) *PingClient {
	client := pingv1.NewPingServiceClient(cc.GetConn())
	return &PingClient{client: client}
}

func newPingClient(i do.Injector) (*PingClient, error) {

	cfg := do.MustInvoke[*config.Config](i)
	addr := cfg.GRPC.PingAddr
	if addr == "" {
		addr = "localhost:50053"
	}

	wrapper, err := config.NewGRPCClientWrapper(addr)
	if err != nil {
		return nil, fmt.Errorf("ping gRPC client: %w", err)
	}

	return NewPingClient(wrapper), nil
}

func RegisterPingClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingClient, error) {
		return newPingClient(i)
	})
}

func (c *PingClient) Ping(ctx context.Context, req *pingv1.PingRequest) (bool, error) {

	resp, err := c.client.Ping(ctx, req)
	if err != nil {
		return false, fmt.Errorf("ping gRPC: %w", err)
	}

	if resp.Error != "" {
		return false, fmt.Errorf("%s", resp.Error)
	}

	return resp.Running, nil
}
