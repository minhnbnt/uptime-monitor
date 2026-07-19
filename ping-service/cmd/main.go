package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/app"
)

func main() {

	configPath := flag.String("config", "", "path to config file")
	dev := flag.Bool("dev", false, "enable dev features")
	flag.Parse()

	injector := do.New()
	app.RegisterPackages(injector, *configPath, *dev)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Wait()

	waitgroup.Go(func() { _, _ = injector.ShutdownOnSignalsWithContext(ctx) })

	waitgroup.Go(func() { app.RunStreamConsumer(ctx, injector) })
	waitgroup.Go(func() { app.RunZSetWorker(ctx, injector) })
	waitgroup.Go(func() { app.RunPingGRPCServer(ctx, injector) })

	waitgroup.Go(func() { app.RunHealthCheckServer(ctx, injector) })
}
