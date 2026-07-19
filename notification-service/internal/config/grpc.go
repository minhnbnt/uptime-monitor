package config

import (
	"fmt"

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

func RegisterGRPCClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*GRPCClientWrapper, error) {
		cfg := do.MustInvoke[*Config](i)
		return NewGRPCClientWrapper(cfg.GRPC.ServerAddr)
	})
}

func (w *GRPCClientWrapper) Shutdown() error {
	return w.conn.Close()
}

func (w *GRPCClientWrapper) GetConn() *grpc.ClientConn {
	return w.conn
}

type GRPCOntimeClientWrapper struct {
	conn *grpc.ClientConn
}

func NewGRPCOntimeClientWrapper(host string) (*GRPCOntimeClientWrapper, error) {
	credentials := grpc.WithTransportCredentials(insecure.NewCredentials())
	conn, err := grpc.NewClient(host, credentials)
	if err != nil {
		return nil, fmt.Errorf("grpc dial ontime: %w", err)
	}
	return &GRPCOntimeClientWrapper{conn: conn}, nil
}

func RegisterGRPCOntimeClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*GRPCOntimeClientWrapper, error) {
		cfg := do.MustInvoke[*Config](i)
		return NewGRPCOntimeClientWrapper(cfg.GRPC.EventAddr)
	})
}

func (w *GRPCOntimeClientWrapper) Shutdown() error {
	return w.conn.Close()
}

func (w *GRPCOntimeClientWrapper) GetConn() *grpc.ClientConn {
	return w.conn
}
