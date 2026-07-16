package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	ontimegrpc "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/grpc"
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
		config.RegisterGRPC,
		config.RegisterGRPCClient,

		serverclient.RegisterClient,

		repository.RegisterOntineRepository,
		repository.RegisterOntimeCacheRepository,
		service.RegisterBatcher,
		service.RegisterOntimeService,

		handler.RegisterOntimeHandler,

		recorder.RegisterDedupRecorder,
		ontimegrpc.RegisterEventServer,
	}

	for _, p := range packages {
		p(injector)
	}
}
