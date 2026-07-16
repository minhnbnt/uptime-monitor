package config

import (
	"fmt"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	GRPCClientNameServer = "server"
	GRPCClientNameEvent  = "event"
)

type GRPCClientWrapper struct {
	conn *grpc.ClientConn
}

func NewGRPCClientWrapper(host string) (*GRPCClientWrapper, error) {

	credentials := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.NewClient(host, credentials)

	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return &GRPCClientWrapper{conn: conn}, nil
}

func (w *GRPCClientWrapper) Shutdown() error {
	return w.conn.Close()
}

func (w *GRPCClientWrapper) GetConn() *grpc.ClientConn {
	return w.conn
}

func RegisterGRPCClients(i do.Injector) {

	do.ProvideNamed(i, GRPCClientNameServer, func(i do.Injector) (*GRPCClientWrapper, error) {
		cfg := do.MustInvoke[*Config](i)
		return NewGRPCClientWrapper(cfg.GRPC.ServerAddr)
	})

	do.ProvideNamed(i, GRPCClientNameEvent, func(i do.Injector) (*GRPCClientWrapper, error) {
		cfg := do.MustInvoke[*Config](i)
		return NewGRPCClientWrapper(cfg.GRPC.EventAddr)
	})
}
