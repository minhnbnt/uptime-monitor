package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/rs/cors"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/handler"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/authclient"
)

func RunWebServer(ctx context.Context, i do.Injector) {

	log := do.MustInvoke[*slog.Logger](i)

	errorHandler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		log.Error("request validation failed", slog.Any("error", err))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(api.ErrorResponse{
			Error: api.ErrorResponseError{
				Code:    "VALIDATION_ERROR",
				Message: "invalid request",
			},
		})
	}

	notificationHandler := do.MustInvoke[*handler.NotificationHandler](i)

	srv, err := api.NewServer(
		notificationHandler,
		api.WithPathPrefix(""),
		api.WithErrorHandler(errorHandler),
	)
	if err != nil {
		log.Error("failed to create server", slog.Any("error", err))
		panic(err)
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	muxHandler := authclient.XUserIDMiddleware(corsMiddleware.Handler(srv))

	cfg := do.MustInvoke[*config.Config](i)

	httpServer := http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: muxHandler,
	}

	go func() {
		<-ctx.Done()
		if err := httpServer.Close(); err != nil {
			log.Error("failed to shutdown server", slog.Any("error", err))
		}
	}()

	log.Info("notification-service starting", slog.String("port", cfg.Server.Port))

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

func RunDigestWorker(ctx context.Context, i do.Injector) {

	runner := do.MustInvoke[*handler.DigestWorkerRunner](i)
	log := do.MustInvoke[*slog.Logger](i)

	if err := runner.RunDigestWorker(ctx); err != nil {
		log.Error("digest worker failed", slog.Any("error", err))
		panic(err)
	}
}
