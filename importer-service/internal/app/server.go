package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/handler"
	"github.com/samber/do/v2"
)

func RunWebServer(ctx context.Context, injector do.Injector) {

	importHandler := do.MustInvoke[*handler.ImportHandler](injector)

	srv, err := api.NewServer(importHandler, api.WithPathPrefix(""))
	if err != nil {
		panic(err)
	}

	cfg := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	authMW := authclient.NewAuthMiddleware(log)
	mux := http.NewServeMux()
	mux.Handle("/", authMW.XUserIDMiddleware(srv))

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

	log.Info("importer-service starting", slog.String("port", cfg.Server.Port))

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
