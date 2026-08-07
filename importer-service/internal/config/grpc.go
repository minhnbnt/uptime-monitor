package config

import (
	"fmt"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type GRPCClientWrapper struct {
	client *grpc.ClientConn
}

func (c *GRPCClientWrapper) GetClient() *grpc.ClientConn {
	return c.client
}

func (c *GRPCClientWrapper) Shutdown() error {
	return c.client.Close()
}

func RegisterGRPCClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*GRPCClientWrapper, error) {

		cfg := do.MustInvoke[*Config](i)

		opt := grpc.WithTransportCredentials(insecure.NewCredentials())
		conn, err := grpc.NewClient(cfg.GRPC.ServerAddr, opt)
		if err != nil {
			return nil, fmt.Errorf("grpc dial: %w", err)
		}

		return &GRPCClientWrapper{client: conn}, nil
	})
}
