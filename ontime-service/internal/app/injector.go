package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	ontimehandler "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/handler"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/repository"
	ontime "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/service"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/serverclient"
)

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	packages := []func(do.Injector){

		config.RegisterConfigPath(configPath),
		config.RegisterLogger(dev),
		config.RegisterGORMDB,
		config.RegisterRedisClient,
		config.RegisterGRPC,
		config.RegisterGRPCClient,

		serverclient.RegisterClient,

		ontimerepo.RegisterOntineRepository,
		ontimerepo.RegisterOntimeCacheRepository,
		ontime.RegisterBatcher,
		ontime.RegisterOntimeService,

		authclient.RegisterAuthMiddleware,
		ontimehandler.RegisterOntimeHandler,
	}

	for _, p := range packages {
		p(injector)
	}
}
