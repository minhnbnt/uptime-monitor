package main

//go:generate go tool ogen --config ../.ogen.yml --target ../generated/api --package api --clean ../api/spec.yaml

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/app"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	dev := flag.Bool("dev", false, "enable dev features")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	injector := do.New()
	app.RegisterPackages(injector, *configPath, *dev)

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Wait()

	waitgroup.Go(func() { _, _ = injector.ShutdownOnSignalsWithContext(ctx) })
	waitgroup.Go(func() { app.RunWebServer(ctx, injector) })
	waitgroup.Go(func() { app.RunGRPCServer(ctx, injector) })
}
