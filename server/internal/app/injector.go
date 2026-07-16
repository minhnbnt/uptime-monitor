package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/authclient"
	"github.com/minhnbnt/uptime-monitor/internal/config"
	importerhandler "github.com/minhnbnt/uptime-monitor/internal/features/importer/handler"
	importerservice "github.com/minhnbnt/uptime-monitor/internal/features/importer/service"
	ontimehandler "github.com/minhnbnt/uptime-monitor/internal/features/ontime/handler"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/ontime/repository"
	ontimeservice "github.com/minhnbnt/uptime-monitor/internal/features/ontime/service"
	infraPing "github.com/minhnbnt/uptime-monitor/internal/features/ping/infrastructure"
	serverhandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/features/server/infrastructure"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	featservice "github.com/minhnbnt/uptime-monitor/internal/features/server/service"
	servergrpc "github.com/minhnbnt/uptime-monitor/internal/grpc"
	"github.com/minhnbnt/uptime-monitor/internal/grpcclient"
	"github.com/minhnbnt/uptime-monitor/internal/features/ping/scheduler"
	"github.com/minhnbnt/uptime-monitor/internal/server"
)

func providersAfterConfig(dev bool) []func(do.Injector) {
	return []func(do.Injector){

		config.RegisterLogger(dev),
		config.RegisterGORMDB,
		config.RegisterRedisClient,
		config.RegisterTemporalClient,

		config.RegisterGRPCClients,
		grpcclient.RegisterEventClient,

		serverrepo.RegisterServerRepository,
		serverrepo.RegisterEndpointRepository,
		serverrepo.RegisterParadeDBSearcher,

		ontimerepo.RegisterOntineRepository,
		ontimerepo.RegisterOntimeCacheRepository,

		serverinfra.RegisterExcelExporter,
		serverinfra.RegisterExcelParser,
		infraPing.RegisterPingWorker,

		featservice.RegisterServerService,
		featservice.RegisterEndpointService,
		importerservice.RegisterImportService,
		ontimeservice.RegisterBatcher,
		ontimeservice.RegisterOntimeService,

		serverhandler.RegisterServerHandler,
		serverhandler.RegisterEndpointHandler,
		importerhandler.RegisterImportHandler,
		ontimehandler.RegisterOntimeHandler,

		scheduler.RegisterEndpointMetaCache,
		scheduler.RegisterEndpointFetcher,
		scheduler.RegisterEndpointProvider,

		servergrpc.RegisterEndpointServer,
		servergrpc.RegisterServerServer,

		authclient.RegisterAuthMiddleware,

		server.RegisterCompositeHandler,
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
