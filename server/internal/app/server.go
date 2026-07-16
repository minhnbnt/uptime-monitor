package app

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/samber/do/v2"

	apidocs "github.com/minhnbnt/uptime-monitor/api"
	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/authclient"
	"github.com/minhnbnt/uptime-monitor/internal/config"
	servergrpc "github.com/minhnbnt/uptime-monitor/internal/grpc"
	"github.com/minhnbnt/uptime-monitor/internal/server"
)

func GetErrorHandler(log *slog.Logger) api.ErrorHandler {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {

		log.Error("request failed", slog.Any("error", err))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		_ = json.NewEncoder(w).Encode(api.ErrorResponse{
			Error: api.ErrorResponseError{
				Code:    "INTERNAL_ERROR",
				Message: "internal server error",
			},
		})
	}
}

func NewDevHandler(handler http.Handler, log *slog.Logger) http.Handler {

	docsHandler, err := apidocs.GetHandler("Uptime Monitor API")
	if err != nil {
		log.Error("failed to get API docs", slog.Any("error", err))
		panic(err)
	}

	mux := http.NewServeMux()

	mux.Handle("/docs/", http.StripPrefix("/docs", docsHandler))
	mux.Handle("/", handler)

	return mux
}

func RunWebServer(ctx context.Context, i do.Injector, dev bool) {

	log := do.MustInvoke[*slog.Logger](i)

	errorHandler := GetErrorHandler(log)
	compositeHandler := do.MustInvoke[*server.CompositeHandler](i)
	authMiddleware := do.MustInvoke[*authclient.AuthMiddleware](i)

	apiHandler, err := api.NewServer(
		compositeHandler,
		api.WithPathPrefix(""),
		api.WithErrorHandler(errorHandler),
	)

	if err != nil {
		log.Error("failed to create server", slog.Any("error", err))
		panic(err)
	}

	handler := authMiddleware.XUserIDMiddleware(apiHandler)
	if dev {
		handler = NewDevHandler(handler, log)
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

	endpointSrv := do.MustInvoke[*servergrpc.EndpointServer](i)
	serverSrv := do.MustInvoke[*servergrpc.ServerServer](i)

	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[*slog.Logger](i)

	addr := ":" + cfg.GRPC.Port
	log.Info("gRPC server starting", slog.String("addr", addr))

	if err := servergrpc.StartGRPCServer(ctx, addr, endpointSrv, serverSrv); err != nil {
		log.Error("gRPC server failed", slog.Any("error", err))
		panic(err)
	}
}
