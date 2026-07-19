package app

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"

	"github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/ping/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	pinggrpcserver "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcserver"
)

func getGrpcServer(ctx context.Context, injector do.Injector) *grpc.Server {

	grpcServer := grpc.NewServer()
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	server := do.MustInvoke[*pinggrpcserver.PingServer](injector)
	pingv1.RegisterPingServiceServer(grpcServer, server)

	return grpcServer
}

func RunPingGRPCServer(ctx context.Context, injector do.Injector) {

	log := do.MustInvoke[*slog.Logger](injector)
	cfg := do.MustInvoke[*config.Config](injector)

	addr := ":" + cfg.GRPC.Port
	if cfg.GRPC.Port == "" {
		addr = ":50053"
	}
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		log.Error("ping grpc server error", slog.Any("error", err))
		panic(err)
	}

	grpcServer := getGrpcServer(ctx, injector)

	log.Info("ping gRPC server starting", slog.String("addr", addr))
	err = grpcServer.Serve(listener)
	if errors.Is(err, grpc.ErrServerStopped) {
		return
	}

	if err != nil {
		log.Error("ping grpc server error", slog.Any("error", err))
		panic(err)
	}
}
