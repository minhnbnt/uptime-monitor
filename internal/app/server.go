package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/rs/cors"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	apidocs "github.com/minhnbnt/uptime-monitor/api"
	"github.com/minhnbnt/uptime-monitor/generated/api"
	authmiddleware "github.com/minhnbnt/uptime-monitor/internal/features/auth/middleware"
	"github.com/minhnbnt/uptime-monitor/internal/server"
)

func RunWebServer(ctx context.Context, i do.Injector, dev bool) {

	logger := do.MustInvoke[*zap.Logger](i)

	errorHandler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {

		logger.Error("request validation failed", zap.Error(err))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		_ = json.NewEncoder(w).Encode(api.ErrorResponse{
			Error: api.ErrorResponseError{
				Code:    "VALIDATION_ERROR",
				Message: "invalid request",
			},
		})
	}

	compositeHandler := do.MustInvoke[*server.CompositeHandler](i)
	authMiddleware := do.MustInvoke[*authmiddleware.AuthMiddleware](i)

	srv, err := api.NewServer(
		compositeHandler,
		authMiddleware,
		api.WithPathPrefix(""),
		api.WithErrorHandler(errorHandler),
	)

	if err != nil {
		logger.Panic("failed to create server", zap.Error(err))
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	handler := corsMiddleware.Handler(srv)

	if dev {

		docsHandler, err := apidocs.GetHandler("Uptime Monitor API")
		if err != nil {
			logger.Panic("failed to get API docs", zap.Error(err))
		}

		mux := http.NewServeMux()
		mux.Handle("/docs/", http.StripPrefix("/docs", docsHandler))
		mux.Handle("/", handler)
		handler = mux
	}

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: handler,
	}

	go func() {
		<-ctx.Done()
		if err := httpServer.Close(); err != nil {
			logger.Panic("failed to shutdown server", zap.Error(err))
		}
	}()

	if err := httpServer.ListenAndServe(); err != nil {
		logger.Panic("failed to run server", zap.Error(err))
	}
}
