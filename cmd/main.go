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
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/argon2"
	authhandler "github.com/minhnbnt/uptime-monitor/internal/features/auth/handler"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/jwt"
	authmiddleware "github.com/minhnbnt/uptime-monitor/internal/features/auth/middleware"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/features/auth/repository"
	authservice "github.com/minhnbnt/uptime-monitor/internal/features/auth/service"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/token"
	importerhandler "github.com/minhnbnt/uptime-monitor/internal/features/importer/handler"
	importerservice "github.com/minhnbnt/uptime-monitor/internal/features/importer/service"
	notificationhandler "github.com/minhnbnt/uptime-monitor/internal/features/notification/handler"
	notifyservice "github.com/minhnbnt/uptime-monitor/internal/features/notification/service"
	ontimehandler "github.com/minhnbnt/uptime-monitor/internal/features/ontime/handler"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/ontime/repository"
	ontimeservice "github.com/minhnbnt/uptime-monitor/internal/features/ontime/service"
	pinghandler "github.com/minhnbnt/uptime-monitor/internal/features/ping/handler"
	pinginfra "github.com/minhnbnt/uptime-monitor/internal/features/ping/infrastructure"
	pingrepo "github.com/minhnbnt/uptime-monitor/internal/features/ping/repository"
	pingsched "github.com/minhnbnt/uptime-monitor/internal/features/ping/scheduler"
	pingservice "github.com/minhnbnt/uptime-monitor/internal/features/ping/service"
	featserverhandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/features/server/infrastructure"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	featservice "github.com/minhnbnt/uptime-monitor/internal/features/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/server"
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
		pingsched.RegisterTemporalSchedulerRepository,
		authrepo.RegisterUserRepository,
		authrepo.RegisterRedisRevokedTokenRepository,
		serverrepo.RegisterParadeDBSearcher,

		pingrepo.RegisterServerEventRepository,
		pingrepo.RegisterRedisServerEventRepository,

		pingrepo.RegisterNotificationConfigRepository,

		ontimerepo.RegisterOntineRepository,
		ontimerepo.RegisterOntimeCacheRepository,

		pinginfra.RegisterPingWorker,
		pinginfra.RegisterRecordStatusWorker,
		config.RegisterMailClient,
		pinginfra.RegisterMailer,

		pingsched.RegisterZSetScheduleRepository,
		pingsched.RegisterScoreUpdater,
		pingsched.RegisterEndpointFetcher,
		pingsched.RegisterEndpointProvider,
		pingsched.RegisterEndpointMetaCache,

		jwt.RegisterProvider,
		argon2.RegisterArgon2PasswordEncoder,
		serverinfra.RegisterExcelGenerator,
		pinginfra.RegisterDigestStarter,

		featservice.RegisterServerService,
		featservice.RegisterEndpointService,
		importerservice.RegisterImportService,
		ontimeservice.RegisterBatcher,
		ontimeservice.RegisterOntimeService,
		authservice.RegisterAuthService,
		token.RegisterTokenGenerator,
		token.RegisterTokenValidator,
		pingservice.RegisterPingService,
		pingservice.RegisterLoopService,
		pingservice.RegisterDigestService,
		notifyservice.RegisterNotificationService,

		featserverhandler.RegisterServerHandler,
		featserverhandler.RegisterEndpointHandler,
		authhandler.RegisterAuthHandler,
		importerhandler.RegisterImportHandler,
		ontimehandler.RegisterOntimeHandler,
		notificationhandler.RegisterNotificationHandler,

		authmiddleware.RegisterAuthMiddleware,

		server.RegisterCompositeHandler,
		pinghandler.RegisterTemporalWorkerRunner,
		pinghandler.RegisterZSetWorkerRunner,
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
		waitgroup.Go(func() { runTemporalWorker(ctx, injector) })
	}

	if *enableServer {
		waitgroup.Go(func() { runWebServer(ctx, injector, *dev) })
	}
}

func runTemporalWorker(ctx context.Context, i do.Injector) {

	temporal := do.MustInvoke[*pinghandler.TemporalWorkerRunner](i)
	log := do.MustInvoke[logger.Logger](i)

	err := temporal.RunTemporalWorker(ctx)
	if err != nil {
		log.Error("Temporal worker failed", logger.Error(err))
	}
}

func runWorker(ctx context.Context, i do.Injector) {

	cfg := do.MustInvoke[*config.Config](i)
	log := do.MustInvoke[logger.Logger](i)

	switch cfg.Scheduler.Backend {

	case "redis":
		runner := do.MustInvoke[*pinghandler.ZSetWorkerRunner](i)
		err := runner.RunZSetWorker(ctx)
		if err != nil {
			log.Error("ZSet worker failed", logger.Error(err))
		}

	case "temporal":
		return

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
	authMiddleware := do.MustInvoke[*authmiddleware.AuthMiddleware](i)

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
