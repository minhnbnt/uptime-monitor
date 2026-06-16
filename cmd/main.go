package main

//go:generate go tool ogen --config ../.ogen.yml --target ../generated/api --package api --clean ../api/spec.yaml

import (
	"context"
	"encoding/json"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rs/cors"
	"github.com/samber/do/v2"
	"go.uber.org/zap"

	apidocs "github.com/minhnbnt/uptime-monitor/api"
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
	searchrepo "github.com/minhnbnt/uptime-monitor/internal/repository/search"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/handler"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
	"github.com/minhnbnt/uptime-monitor/internal/server/middleware"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
	authservice "github.com/minhnbnt/uptime-monitor/internal/server/service/auth"
	ontime "github.com/minhnbnt/uptime-monitor/internal/server/service/ontime"
)

func main() {

	configPath := flag.String("config", "", "path to config file")

	enableServer := flag.Bool("server", true, "start HTTP API server")
	enableWorker := flag.Bool("worker", true, "start background worker")

	dev := flag.Bool("dev", false, "enable dev features (API docs)")

	injector := do.New(

		config.RegisterConfigPath(*configPath),
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
		searchrepo.RegisterParadeDBSearcher,

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
		serverinfra.RegisterExcelGenerator,

		service.RegisterServerService,
		service.RegisterEndpointService,
		service.RegisterImportService,
		ontime.RegisterBatcher,
		ontime.RegisterOntimeService,
		authservice.RegisterAuthService,
		authservice.RegisterTokenGenerator,
		authservice.RegisterTokenValidator,
		monitorservices.RegisterPingService,
		monitorservices.RegisterLoopService,

		handler.RegisterServerHandler,
		handler.RegisterEndpointHandler,
		handler.RegisterAuthHandler,
		handler.RegisterImportHandler,

		middleware.RegisterAuthMiddleware,

		server.RegisterCompositeHandler,
		monitorhandler.RegisterTemporalWorkerRunner,
		monitorhandler.RegisterZSetWorkerRunner,
	)

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)

	defer stop()

	waitgroup := sync.WaitGroup{}
	defer waitgroup.Wait()

	waitgroup.Go(func() { _, _ = injector.ShutdownOnSignalsWithContext(ctx) })

	if *enableWorker {
		waitgroup.Go(func() { runWorker(ctx, injector) })
	}

	if *enableServer {
		waitgroup.Go(func() { runWebServer(ctx, injector, *dev) })
	}
}

func runWorker(ctx context.Context, i do.Injector) {

	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[logger.Logger](i)

	switch cfg.Scheduler.Backend {
	case "temporal":
		runner := do.MustInvoke[*monitorhandler.TemporalWorkerRunner](i)
		runner.RunTemporalWorker(ctx)

	case "redis":
		runner := do.MustInvoke[*monitorhandler.ZSetWorkerRunner](i)
		_ = runner.RunZSetWorker(ctx)

	default:
		log.Panic(
			"unknown scheduler backend",
			logger.String("backend", cfg.Scheduler.Backend),
		)
	}
}

func runWebServer(ctx context.Context, i do.Injector, dev bool) {

	logger := do.MustInvoke[*zap.Logger](i)

	errorHandler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("request validation failed", zap.Error(err))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		_ = json.NewEncoder(w).Encode(api.ErrorResponse{
			Error: api.ErrorResponseError{
				Code:    "VALIDATION_ERROR",
				Message: "invalid request",
			},
		})
	}

	compositeHandler := do.MustInvoke[*server.CompositeHandler](i)
	authMiddleware := do.MustInvoke[*middleware.AuthMiddleware](i)

	server, err := api.NewServer(
		compositeHandler,
		authMiddleware,
		api.WithPathPrefix(""),
		api.WithErrorHandler(errorHandler),
	)

	if err != nil {
		logger.Panic("failed to create server", zap.Error(err))
	}

	corsMiddleware := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	handler := corsMiddleware.Handler(server)

	if dev {

		docsHandler, err := apidocs.GetHandler("Uptime Monitor API")
		if err != nil {
			logger.Panic("failed to get API docs", zap.Error(err))
		}

		mux := http.NewServeMux()

		mux.Handle("/docs/", http.StripPrefix("/docs", docsHandler))
		mux.Handle("/", handler)

		handler = mux
	}

	httpServer := http.Server{
		Addr:    ":8080",
		Handler: handler,
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
