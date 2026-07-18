package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"

	"github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/ping/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	pinghandler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/handler"
	pinggrpcserver "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcserver"
)

func RunZSetWorker(ctx context.Context, i do.Injector) {
	runner := do.MustInvoke[*pinghandler.ZSetWorkerRunner](i)
	runner.RunZSetWorker(ctx)
}

func RunStreamConsumer(ctx context.Context, i do.Injector) {
	worker := do.MustInvoke[*pinghandler.EndpointEventWorker](i)
	worker.Run(ctx)
}

func RunPingGRPCServer(ctx context.Context, injector do.Injector) {

	log := do.MustInvoke[*slog.Logger](injector)
	grpcServer := do.MustInvoke[*grpc.Server](injector)
	server := do.MustInvoke[*pinggrpcserver.PingServer](injector)

	pingv1.RegisterPingServiceServer(grpcServer, server)

	listener := do.MustInvoke[net.Listener](injector)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	log.Info("ping gRPC server starting", slog.String("addr", listener.Addr().String()))
	if err := grpcServer.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
		log.Error("ping grpc server error", slog.Any("error", err))
		panic(err)
	}
}

func RunHealthCheckServer(ctx context.Context, injector do.Injector) {

	config := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintln(w, "OK")
	})

	srv := &http.Server{
		Addr:    ":" + config.Server.Port,
		Handler: mux,
	}

	go func() {

		<-ctx.Done()

		if err := srv.Close(); err != nil {
			log.Error("server close error", slog.Any("error", err))
		}
	}()

	err := srv.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return
	}

	if err != nil {
		log.Error("server error", slog.Any("error", err))
		panic(err)
	}
}
