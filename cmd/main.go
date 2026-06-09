package main

//go:generate go tool ogen --config ../ogen.yml --target ../generated/api --package api --clean ../api/spec.yaml

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

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
	revokedtokenrepo "github.com/minhnbnt/uptime-monitor/internal/repository/revokedtoken"
	schedulerrepo "github.com/minhnbnt/uptime-monitor/internal/repository/scheduler"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/handler"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
	servertmiddleware "github.com/minhnbnt/uptime-monitor/internal/server/middleware"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
	authservice "github.com/minhnbnt/uptime-monitor/internal/server/service/auth"
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
		revokedtokenrepo.RegisterRedisRevokedTokenRepository,

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
		authservice.RegisterAuthService,
		authservice.RegisterTokenGenerator,
		authservice.RegisterTokenValidator,
		monitorservices.RegisterPingService,
		monitorservices.RegisterLoopService,

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

func errorHandler(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)

	json.NewEncoder(w).Encode(api.ErrorResponse{
		Error: api.ErrorResponseError{
			Code:    "VALIDATION_ERROR",
			Message: err.Error(),
		},
	})
}

func runWebServer(ctx context.Context, i do.Injector) {

	logger := do.MustInvoke[*zap.Logger](i)

	compositeHandler := do.MustInvoke[*server.CompositeHandler](i)
	authMiddleware := servertmiddleware.RegisterAuthMiddleware(i)

	server, err := api.NewServer(
		compositeHandler,
		authMiddleware,
		api.WithPathPrefix(""),
		api.WithErrorHandler(errorHandler),
	)

	if err != nil {
		logger.Panic("failed to create server", zap.Error(err))
	}

	middleware := servertmiddleware.CORSMiddleware()

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: middleware(server),
	}

	go func() {
		<-ctx.Done()
		if err := httpServer.Close(); err != nil {
			logger.Panic("failed to shutdown server", zap.Error(err))
		}
	}()

	if err := httpServer.ListenAndServe(); err != nil {
		logger.Panic("failed to run server", zap.Error(err))
	}
}
