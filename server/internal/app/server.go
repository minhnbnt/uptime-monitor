package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/rs/cors"
	"github.com/samber/do/v2"

	apidocs "github.com/minhnbnt/uptime-monitor/api"
	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/authclient"
	"github.com/minhnbnt/uptime-monitor/internal/config"
	servergrpc "github.com/minhnbnt/uptime-monitor/internal/grpc"
	"github.com/minhnbnt/uptime-monitor/internal/server"
)

func RunWebServer(ctx context.Context, i do.Injector, dev bool) {

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

	compositeHandler := do.MustInvoke[*server.CompositeHandler](i)
	authMiddleware := do.MustInvoke[*authclient.AuthMiddleware](i)

	rawSrv, err := api.NewServer(
		compositeHandler,
		api.WithPathPrefix(""),
		api.WithErrorHandler(errorHandler),
	)

	srv := authMiddleware.XUserIDMiddleware(rawSrv)

	if err != nil {
		log.Error("failed to create server", slog.Any("error", err))
		panic(err)
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
			log.Error("failed to get API docs", slog.Any("error", err))
			panic(err)
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
			log.Error("failed to shutdown server", slog.Any("error", err))
			panic(err)
		}
	}()

	err = httpServer.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		log.Info("server closed")
		return
	}

	if err != nil {
		log.Error("failed to run server", slog.Any("error", err))
		panic(err)
	}
}

func RunGRPCServer(ctx context.Context, i do.Injector) {

	log := do.MustInvoke[*slog.Logger](i)
	cfg := do.MustInvoke[*config.Config](i)

	endpointSrv := do.MustInvoke[*servergrpc.EndpointServer](i)
	serverSrv := do.MustInvoke[*servergrpc.ServerServer](i)

	addr := ":" + cfg.GRPC.Port
	log.Info("gRPC server starting", slog.String("addr", addr))

	if err := servergrpc.StartGRPCServer(ctx, addr, endpointSrv, serverSrv); err != nil {
		log.Error("gRPC server failed", slog.Any("error", err))
		panic(err)
	}
}
