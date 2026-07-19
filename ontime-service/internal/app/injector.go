package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/handler"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/recorder"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/serverclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/service"
)

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	packages := []func(do.Injector){

		config.RegisterConfigPath(configPath),
		config.RegisterLogger(dev),
		config.RegisterGORMDB,
		config.RegisterRedisClient,
		config.RegisterGRPCClient,

		serverclient.RegisterClient,

		repository.RegisterOntineRepository,
		repository.RegisterOntimeCacheRepository,
		repository.RegisterEventRepository,
		service.RegisterBatcher,
		service.RegisterOntimeService,
		service.RegisterEventService,

		handler.RegisterOntimeHandler,
		handler.RegisterEventRecorderServer,
		handler.RegisterStatusServer,
		handler.RegisterOntimeGRPCServer,

		recorder.RegisterDedupRecorder,
	}

	for _, p := range packages {
		p(injector)
	}
}
