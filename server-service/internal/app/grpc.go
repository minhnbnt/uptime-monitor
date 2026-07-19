package app

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"

	endpointv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/endpoint/v1"
	serverv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/server/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/handler"
)

func getGrpcServer(ctx context.Context, injector do.Injector) *grpc.Server {

	grpcServer := grpc.NewServer()
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	endpointSrv := do.MustInvoke[*handler.EndpointServer](injector)
	serverSrv := do.MustInvoke[*handler.ServerServer](injector)
	endpointv1.RegisterEndpointServiceServer(grpcServer, endpointSrv)
	serverv1.RegisterServerServiceServer(grpcServer, serverSrv)

	return grpcServer
}

func RunGRPCServer(ctx context.Context, injector do.Injector) {

	cfg := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	addr := ":" + cfg.GRPC.Port
	log.Info("gRPC server starting", slog.String("addr", addr))

	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
		panic(err)
	}

	grpcServer := getGrpcServer(ctx, injector)

	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		log.Error("gRPC server error", slog.Any("error", err))
		panic(err)
	}
}
