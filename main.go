package main

//go:generate go tool oapi-codegen -config=oapi-codegen.yml api/spec.yaml

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/config"
	temporalcfg "github.com/minhnbnt/uptime-monitor/internal/config/temporal"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	monitorhandler "github.com/minhnbnt/uptime-monitor/internal/monitor/handler"
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrashtructure"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/monitor/infrashtructure/repository"
	monitorservices "github.com/minhnbnt/uptime-monitor/internal/monitor/services"
	"github.com/minhnbnt/uptime-monitor/internal/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/handler"
	repo "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
)

func main() {

	enableServer := flag.Bool("server", true, "start HTTP API server")
	enableWorker := flag.Bool("worker", true, "start Temporal worker")
	flag.Parse()

	injector := do.New(

		config.RegisterZapLogger,
		config.RegisterGORMDB,
		config.RegisterRedisClient,
		temporalcfg.RegisterConfig,
		temporalcfg.RegisterClient,

		logger.RegisterLogger,
		repo.RegisterServerRepository,
		repo.RegisterEndpointRepository,
		repo.RegisterPingSchedulerRepository,

		monitorrepo.RegisterServerEventRepository,
		monitorrepo.RegisterRedisServerEventRepository,

		infra.RegisterPingWorker,
		infra.RegisterRecordPingStatusWorker,

		service.RegisterServerService,
		service.RegisterEndpointService,
		monitorservices.RegisterPingService,

		handler.RegisterRequestValidator,
		handler.RegisterServerHandler,
		handler.RegisterEndpointHandler,

		server.RegisterCompositeHandler,
		monitorhandler.RegisterTemporalWorkerRunner,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Wait()

	waitgroup.Go(func() { injector.ShutdownOnSignalsWithContext(ctx) })

	if *enableWorker {
		waitgroup.Go(func() { runWorker(ctx, injector) })
	}

	if *enableServer {
		waitgroup.Go(func() { runWebServer(ctx, injector) })
	}
}

func runWorker(ctx context.Context, i do.Injector) {
	runner := do.MustInvoke[*monitorhandler.TemporalWorkerRunner](i)
	runner.RunTemporalWorker(ctx)
}

func runWebServer(ctx context.Context, i do.Injector) {

	router := gin.Default()
	router.Use(cors.Default())

	logger := do.MustInvoke[*zap.Logger](i)

	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.GET("/api/v1/hello", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "Hello, world!"})
	})

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	go func() {

		<-ctx.Done()

		if err := httpServer.Close(); err != nil {
			logger.Panic("failed to shutdown server", zap.Error(err))
		}
	}()

	handler := do.MustInvoke[*server.CompositeHandler](i)
	api.RegisterHandlers(router, handler)

	if err := httpServer.ListenAndServe(); err != nil {
		logger.Panic("failed to run server", zap.Error(err))
	}
}
