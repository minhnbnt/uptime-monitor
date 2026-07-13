package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/argon2"
	authhandler "github.com/minhnbnt/uptime-monitor/internal/features/auth/handler"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/jwt"
	authmiddleware "github.com/minhnbnt/uptime-monitor/internal/features/auth/middleware"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/features/auth/repository"
	authservice "github.com/minhnbnt/uptime-monitor/internal/features/auth/service"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/token"
	digesthandler "github.com/minhnbnt/uptime-monitor/internal/features/digest/handler"
	digestinfra "github.com/minhnbnt/uptime-monitor/internal/features/digest/infrastructure"
	digestrepo "github.com/minhnbnt/uptime-monitor/internal/features/digest/repository"
	digestservice "github.com/minhnbnt/uptime-monitor/internal/features/digest/service"
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
	serverhandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/features/server/infrastructure"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	featservice "github.com/minhnbnt/uptime-monitor/internal/features/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/server"
)

func providersAfterConfig(dev bool) []func(do.Injector) {
	return []func(do.Injector){

		config.RegisterLogger(dev),
		config.RegisterGORMDB,
		config.RegisterRedisClient,
		config.RegisterJwtConfig,
		config.RegisterTokenConfig,
		config.RegisterArgon2Config,
		config.RegisterTemporalClient,

		serverrepo.RegisterServerRepository,
		serverrepo.RegisterEndpointRepository,
		pingsched.RegisterTemporalSchedulerRepository,
		authrepo.RegisterUserRepository,
		authrepo.RegisterRedisRevokedTokenRepository,
		serverrepo.RegisterParadeDBSearcher,

		pingrepo.RegisterServerEventRepository,
		pingrepo.RegisterRedisServerEventRepository,

		digestrepo.RegisterNotificationConfigRepository,

		ontimerepo.RegisterOntineRepository,
		ontimerepo.RegisterOntimeCacheRepository,

		pinginfra.RegisterPingWorker,
		pinginfra.RegisterRecordStatusWorker,
		config.RegisterMailClient,
		digestinfra.RegisterMailer,

		pingsched.RegisterZSetScheduleRepository,
		pingsched.RegisterScoreUpdater,
		pingsched.RegisterEndpointFetcher,
		pingsched.RegisterEndpointProvider,
		pingsched.RegisterEndpointMetaCache,

		jwt.RegisterProvider,
		argon2.RegisterArgon2PasswordEncoder,
		serverinfra.RegisterExcelExporter,
		serverinfra.RegisterExcelParser,
		digestinfra.RegisterDigestStarter,

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
		digestservice.RegisterDigestService,
		notifyservice.RegisterNotificationService,

		serverhandler.RegisterServerHandler,
		serverhandler.RegisterEndpointHandler,
		authhandler.RegisterAuthHandler,
		importerhandler.RegisterImportHandler,
		ontimehandler.RegisterOntimeHandler,
		notificationhandler.RegisterNotificationHandler,

		authmiddleware.RegisterAuthMiddleware,

		server.RegisterCompositeHandler,
		pinghandler.RegisterTemporalWorkerRunner,
		pinghandler.RegisterZSetWorkerRunner,
		digesthandler.RegisterDigestWorkerRunner,
	}
}

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	config.RegisterConfigPath(configPath)(injector)

	for _, p := range providersAfterConfig(dev) {
		p(injector)
	}
}

func RegisterPackagesFromConfig(injector do.Injector, cfg *config.Config, dev bool) {

	config.RegisterConfig(cfg)(injector)

	for _, p := range providersAfterConfig(dev) {
		p(injector)
	}
}
