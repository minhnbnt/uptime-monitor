package app

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/samber/do/v2"
	"google.golang.org/grpc"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/handler"
)

func RunWebServer(ctx context.Context, injector do.Injector) {

	ontimeHandler := do.MustInvoke[*handler.OntimeHandler](injector)

	srv, err := api.NewServer(ontimeHandler, api.WithPathPrefix(""))
	if err != nil {
		panic(err)
	}

	cfg := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	auth := do.MustInvoke[*authclient.AuthMiddleware](injector)

	mux := http.NewServeMux()
	mux.Handle("/", auth.XUserIDMiddleware(srv))

	httpServer := http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		if err := httpServer.Close(); err != nil {
			log.Error("failed to shutdown server", slog.Any("error", err))
		}
	}()

	log.Info("ontime-service starting", slog.String("port", cfg.Server.Port))

	err = httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		log.Info("server closed")
		return
	}

	if err != nil {
		log.Error("server error", slog.Any("error", err))
		panic(err)
	}
}

func RunGRPCServer(ctx context.Context, injector do.Injector) {

	listener := do.MustInvoke[net.Listener](injector)
	grpcServer := do.MustInvoke[*grpc.Server](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	addr := listener.Addr().String()
	log.Info("ontime gRPC server starting", slog.String("addr", addr))

	err := grpcServer.Serve(listener)
	if errors.Is(err, grpc.ErrServerStopped) {
		return
	}
	if err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
		panic(err)
	}
}
