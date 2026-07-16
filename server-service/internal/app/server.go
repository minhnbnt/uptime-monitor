package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/handler"
)

func RunWebServer(ctx context.Context, injector do.Injector) {

	composite := do.MustInvoke[*handler.CompositeHandler](injector)

	srv, err := api.NewServer(composite)
	if err != nil {
		panic(err)
	}

	cfg := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	middleWare := authclient.NewAuthMiddleware(log)

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: middleWare.XUserIDMiddleware(srv),
	}

	go func() {
		<-ctx.Done()
		if err := httpServer.Close(); err != nil {
			log.Error("failed to shutdown http server", slog.Any("error", err))
		}
	}()

	log.Info("server-service starting", slog.String("port", cfg.GRPC.Port))

	err = httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		log.Info("http server closed")
		return
	}

	if err != nil {
		log.Error("http server error", slog.Any("error", err))
		panic(err)
	}
}

func RunGRPCServer(ctx context.Context, injector do.Injector) {

	endpointSrv := do.MustInvoke[*handler.EndpointServer](injector)
	serverSrv := do.MustInvoke[*handler.ServerServer](injector)

	cfg := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	addr := ":" + cfg.GRPC.Port
	log.Info("gRPC server starting", slog.String("addr", addr))

	if err := handler.StartGRPCServer(ctx, addr, endpointSrv, serverSrv); err != nil {
		log.Error("gRPC server error", slog.Any("error", err))
		panic(err)
	}
}
