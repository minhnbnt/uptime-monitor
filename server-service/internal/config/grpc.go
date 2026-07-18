package config

import (
	"context"
	"fmt"
	"net"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

func newGRPCServer(i do.Injector) (*grpc.Server, error) {
	return grpc.NewServer(), nil
}

func newGRPCListener(i do.Injector) (net.Listener, error) {
	cfg := do.MustInvoke[*Config](i)
	port := cfg.GRPC.Port
	if port == "" {
		port = "50053"
	}
	lc := net.ListenConfig{}
	return lc.Listen(context.Background(), "tcp", ":"+port)
}

func RegisterGRPC(i do.Injector) {
	do.Provide(i, newGRPCServer)
	do.Provide(i, newGRPCListener)
}
