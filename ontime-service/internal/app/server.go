package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	ontimehandler "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/handler"
)

func RunWebServer(ctx context.Context, injector do.Injector) {

	handler := do.MustInvoke[*ontimehandler.OntimeHandler](injector)

	srv, err := api.NewServer(handler, api.WithPathPrefix(""))
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
