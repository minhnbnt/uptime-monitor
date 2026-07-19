package app

import (
	"context"
	"errors"
	"log/slog"
	"net"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/handler"
)

func getGrpcServer(ctx context.Context, injector do.Injector) *grpc.Server {

	grpcServer := grpc.NewServer()
	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	recoderService := do.MustInvoke[*handler.EventRecorderServer](injector)
	eventv1.RegisterEventRecorderServiceServer(grpcServer, recoderService)

	statusService := do.MustInvoke[*handler.StatusServer](injector)
	eventv1.RegisterStatusServiceServer(grpcServer, statusService)

	ontimeService := do.MustInvoke[*handler.OntimeGRPCServer](injector)
	eventv1.RegisterOntimeServiceServer(grpcServer, ontimeService)

	return grpcServer
}

func RunGRPCServer(ctx context.Context, injector do.Injector) {

	log := do.MustInvoke[*slog.Logger](injector)
	cfg := do.MustInvoke[*config.Config](injector)

	addr := ":" + cfg.GRPC.Port
	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(ctx, "tcp", addr)
	if err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
		panic(err)
	}

	grpcServer := getGrpcServer(ctx, injector)

	log.Info("ontime gRPC server starting", slog.String("addr", addr))
	err = grpcServer.Serve(listener)
	if errors.Is(err, grpc.ErrServerStopped) {
		return
	}

	if err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
		panic(err)
	}
}
