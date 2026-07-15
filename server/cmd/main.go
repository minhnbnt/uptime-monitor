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

	"github.com/minhnbnt/uptime-monitor/internal/app"
)

func main() {

	configPath := flag.String("config", "", "path to config file")

	enableServer := flag.Bool("server", true, "start HTTP API server")

	dev := flag.Bool("dev", false, "enable dev features (API docs)")

	flag.Parse()

	injector := do.New()
	app.RegisterPackages(injector, *configPath, *dev)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGTERM,
		os.Interrupt,
	)

	defer stop()

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Wait()

	waitgroup.Go(func() { _, _ = injector.ShutdownOnSignalsWithContext(ctx) })

	if *enableServer {
		waitgroup.Go(func() { app.RunWebServer(ctx, injector, *dev) })
		waitgroup.Go(func() { app.RunGRPCServer(ctx, injector) })
	}
}
