package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/handler"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/excel"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/grpcclient"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
)

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	packages := []func(do.Injector){

		config.RegisterConfigPath(configPath),
		config.RegisterLogger(dev),
		config.RegisterGORMDB,
		config.RegisterRedisClient,

		repository.RegisterServerRepository,
		repository.RegisterEndpointRepository,
		repository.RegisterParadeDBSearcher,
		repository.RegisterStreamEventPublisher,

		excel.RegisterExcelExporter,
		excel.RegisterExcelParser,

		grpcclient.RegisterEventClient,

		service.RegisterServerService,
		service.RegisterEndpointService,
		service.RegisterImportService,

		handler.RegisterServerHandler,
		handler.RegisterEndpointHandler,
		handler.RegisterImportHandler,
		handler.RegisterCompositeHandler,

		handler.RegisterEndpointServer,
		handler.RegisterServerServer,
	}

	for _, p := range packages {
		p(injector)
	}
}
