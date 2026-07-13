package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/app"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	dev := flag.Bool("dev", false, "enable dev features")
	flag.Parse()

	injector := do.New()
	app.RegisterPackages(injector, *configPath, *dev)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	go func() { _, _ = injector.ShutdownOnSignalsWithContext(ctx) }()

	go app.RunStreamConsumer(ctx, injector)
	go app.RunZSetWorker(ctx, injector)

	health := do.MustInvoke[*slog.Logger](injector)
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	srv := &http.Server{
		Addr:    ":" + do.MustInvoke[*config.Config](injector).Server.Port,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Close()
	}()

	health.Info("ping-service starting")

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) && err != nil {
		health.Error("server error", slog.Any("error", err))
		os.Exit(1)
	}
}
