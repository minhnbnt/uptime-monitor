package main

//go:generate go tool ogen --config ../../.ogen.yml --target ../../generated/api --package api --clean ../../api/spec.yaml

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

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/app"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/handler"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/token"
)

func main() {

	configPath := flag.String("config", "", "path to config file")
	dev := flag.Bool("dev", false, "enable dev features")
	flag.Parse()

	injector := do.New()
	app.RegisterPackages(injector, *configPath, *dev)

	authHandler := do.MustInvoke[*handler.AuthHandler](injector)

	srv, err := api.NewServer(authHandler, authHandler, api.WithPathPrefix(""))
	if err != nil {
		panic(err)
	}

	cfg := do.MustInvoke[*config.Config](injector)
	log := do.MustInvoke[*slog.Logger](injector)

	tokenValidator := do.MustInvoke[*token.TokenValidator](injector)

	mux := http.NewServeMux()
	mux.Handle("/auth/verify", handler.NewForwardAuthHandler(tokenValidator))
	mux.Handle("/", srv)

	httpServer := http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

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
		os.Exit(1)
	}
}
