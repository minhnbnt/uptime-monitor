package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	pinghandler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/handler"
	pinginfra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/infrastructure"
	pingredis "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/infrastructure/redis"
	pingrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/repository"
	pingsched "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/scheduler"
	pingservice "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/service"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/grpcclient"
)

func providers(dev bool) []func(do.Injector) {
	return []func(do.Injector){

		config.RegisterLogger(dev),
		config.RegisterGORMDB,
		config.RegisterRedisClient,
		config.RegisterGRPCClient,

		pingrepo.RegisterServerEventRepository,
		pingrepo.RegisterRedisServerEventRepository,
		grpcclient.RegisterEndpointClient,

		pingsched.RegisterZSetScheduleRepository,
		pingsched.RegisterScoreUpdater,
		pingsched.RegisterEndpointProvider,

		pinginfra.RegisterPingWorker,
		pinginfra.RegisterRecordStatusWorker,

		pingservice.RegisterPingService,
		pingservice.RegisterLoopService,

		pingredis.RegisterStreamEventConsumer,
		pingservice.RegisterEventService,
		pinghandler.RegisterEndpointEventWorker,
		pinghandler.RegisterZSetWorkerRunner,
	}
}

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	config.RegisterConfigPath(configPath)(injector)

	for _, p := range providers(dev) {
		p(injector)
	}
}
