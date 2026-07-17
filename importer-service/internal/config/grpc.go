package config

import (
	"fmt"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
)

type ServerClient struct {
	serverv1.ServerServiceClient
	shutdown func() error
}

func (c *ServerClient) Shutdown() error {
	return c.shutdown()
}

func RegisterServerClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerClient, error) {
		cfg := do.MustInvoke[*Config](i)

		conn, err := grpc.NewClient(cfg.GRPC.ServerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("grpc dial: %w", err)
		}

		return &ServerClient{
			ServerServiceClient: serverv1.NewServerServiceClient(conn),
			shutdown:            conn.Close,
		}, nil
	})
}
