package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	pinghandler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/handler"
)

func RunZSetWorker(ctx context.Context, i do.Injector) {
	runner := do.MustInvoke[*pinghandler.ZSetWorkerRunner](i)
	runner.RunZSetWorker(ctx)
}

func RunStreamConsumer(ctx context.Context, i do.Injector) {
	worker := do.MustInvoke[*pinghandler.EndpointEventWorker](i)
	worker.Run(ctx)
}

func RunHealthCheckServer(ctx context.Context, injector do.Injector) {

	config := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
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
