package main

//go:generate go tool oapi-codegen -config=../oapi-codegen.yml ../api/spec.yaml

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
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrastructure"
	monitorservices "github.com/minhnbnt/uptime-monitor/internal/monitor/services"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/repository/auth"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/repository/monitor"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/repository/ontime"
	schedulerrepo "github.com/minhnbnt/uptime-monitor/internal/repository/scheduler"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/handler"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
	servertmiddleware "github.com/minhnbnt/uptime-monitor/internal/server/middleware"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
)

func main() {

	enableServer := flag.Bool("server", true, "start HTTP API server")
	enableWorker := flag.Bool("worker", true, "start background worker")
	schedulerBackend := flag.String("scheduler-backend", "temporal", "scheduler backend: temporal | redis")
	flag.Parse()

	injector := do.New(

		config.RegisterZapLogger,
		config.RegisterGORMDB,
		config.RegisterRedisClient,
		config.RegisterJwtConfig,
		config.RegisterTokenConfig,
		config.RegisterArgon2Config,
		temporalcfg.RegisterConfig,
		temporalcfg.RegisterClient,

		logger.RegisterLogger,
		serverrepo.RegisterServerRepository,
		serverrepo.RegisterEndpointRepository,
		schedulerrepo.RegisterTemporalSchedulerRepository,
		authrepo.RegisterUserRepository,

		monitorrepo.RegisterServerEventRepository,
		monitorrepo.RegisterRedisServerEventRepository,

		ontimerepo.RegisterOntimeCacheRepository,

		infra.RegisterPingWorker,
		infra.RegisterRecordStatusWorker,

		schedulerrepo.RegisterZSetScheduleRepository,
		schedulerrepo.RegisterScoreUpdater,
		schedulerrepo.RegisterEndpointFetcher,
		schedulerrepo.RegisterEndpointProvider,
		schedulerrepo.RegisterEndpointMetaCache,

		jwtutil.RegisterProvider,
		serverinfra.RegisterArgon2PasswordEncoder,

		service.RegisterServerService,
		service.RegisterEndpointService,
		service.RegisterOntimeService,
		service.RegisterAuthService,
		service.RegisterTokenGenerator,
		service.RegisterTokenValidator,
		monitorservices.RegisterPingService,
		monitorservices.RegisterLoopService,

		handler.RegisterRequestValidator,
		handler.RegisterServerHandler,
		handler.RegisterEndpointHandler,
		handler.RegisterAuthHandler,

		server.RegisterCompositeHandler,
		monitorhandler.RegisterTemporalWorkerRunner,
		monitorhandler.RegisterZSetWorkerRunner,
	)

	schedulerrepo.RegisterSchedulerBackend(injector, schedulerrepo.SchedulerBackend(*schedulerBackend))

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
	backend := do.MustInvoke[*schedulerrepo.SchedulerBackend](i)

	switch *backend {
	case schedulerrepo.SchedulerBackendTemporal:
		runner := do.MustInvoke[*monitorhandler.TemporalWorkerRunner](i)
		runner.RunTemporalWorker(ctx)
	case schedulerrepo.SchedulerBackendRedis:
		runner := do.MustInvoke[*monitorhandler.ZSetWorkerRunner](i)
		_ = runner.RunZSetWorker(ctx)
	}
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

	compositeHandler := do.MustInvoke[*server.CompositeHandler](i)
	authMiddleware := servertmiddleware.AuthRequired(i)
	api.RegisterHandlersWithOptions(router, compositeHandler, api.GinServerOptions{
		Middlewares: []api.MiddlewareFunc{api.MiddlewareFunc(authMiddleware)},
	})

	if err := httpServer.ListenAndServe(); err != nil {
		logger.Panic("failed to run server", zap.Error(err))
	}
}
