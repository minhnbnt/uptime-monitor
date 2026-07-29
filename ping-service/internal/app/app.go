package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	pinghandler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/handler"
	pinginfra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
	pinggrpcserver "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcserver"
	pingk8s "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
	pingredis "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis"
	pingrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/repository"
	pingsched "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
	pingservice "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

func providers(dev bool) []func(do.Injector) {

	return []func(do.Injector){

		config.RegisterLogger(dev),
		config.RegisterRedisClient,
		config.RegisterGRPCClient,

		pingrepo.RegisterRedisServerEventRepository,
		grpcclient.RegisterEndpointClient,
		grpcclient.RegisterEventRecorderClient,

		pingsched.RegisterZSetScheduleRepository,
		pingsched.RegisterZSetTaskClaimer,
		pingsched.RegisterScoreUpdater,
		pingsched.RegisterServerMetaCache,
		pingsched.RegisterServerProvider,

		pinginfra.RegisterPingWorker,
		pinginfra.RegisterBodyChecker,
		pinginfra.RegisterRecordStatusWorker,

		config.RegisterK8sClient,
		pingk8s.RegisterK8sClient,

		pinggrpcserver.RegisterPingServer,

		pingservice.RegisterResponseChecker,

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
