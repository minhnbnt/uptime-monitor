package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/handler"
	"github.com/samber/do/v2"
)

func RunWebServer(ctx context.Context, injector do.Injector) {

	authHandler := do.MustInvoke[*handler.AuthHandler](injector)

	srv, err := api.NewServer(authHandler, authHandler, api.WithPathPrefix(""))
	if err != nil {
		panic(err)
	}

	cfg := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	forwardAuthHandler := do.MustInvoke[*handler.ForwardAuthHandler](injector)

	mux := http.NewServeMux()

	mux.Handle("/auth/verify", forwardAuthHandler)
	mux.Handle("/", srv)

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

	log.Info("auth-service starting", slog.String("port", cfg.Server.Port))

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
