package grpcclient

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pingv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/ping/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
)

type PingClient struct {
	client pingv1.PingServiceClient
}

func NewPingClient(cc grpc.ClientConnInterface) *PingClient {
	return &PingClient{client: pingv1.NewPingServiceClient(cc)}
}

func newPingClient(i do.Injector) (*PingClient, error) {

	cfg := do.MustInvoke[*config.Config](i)
	addr := cfg.GRPC.PingAddr
	if addr == "" {
		addr = "localhost:50053"
	}

	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("ping gRPC client: %w", err)
	}

	return NewPingClient(cc), nil
}

func RegisterPingClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingClient, error) {
		return newPingClient(i)
	})
}

func (c *PingClient) Ping(
	ctx context.Context,
	method, url string,
	timeoutMs int64,
	expectedCode int32,
	bodyCheckExpr string,
) (int, error) {

	resp, err := c.client.Ping(ctx, &pingv1.PingRequest{
		Method:        method,
		Url:           url,
		TimeoutMs:     timeoutMs,
		ExpectedCode:  expectedCode,
		BodyCheckExpr: bodyCheckExpr,
	})
	if err != nil {
		return 0, fmt.Errorf("ping gRPC: %w", err)
	}

	if resp.Error != "" {
		return int(resp.StatusCode), fmt.Errorf("%s", resp.Error)
	}

	return int(resp.StatusCode), nil
}
