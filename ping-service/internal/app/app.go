package app

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	pinghandler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/handler"
	pinginfra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
	pinggrpcserver "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcserver"
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
		pingrepo.RegisterPushRateLimiter,
		pingrepo.RegisterFreshnessStore,
		grpcclient.RegisterEndpointClient,
		grpcclient.RegisterEventRecorderClient,
		grpcclient.RegisterServerClient,

		pingsched.RegisterZSetScheduleRepository,
		pingsched.RegisterZSetTaskClaimer,
		pingsched.RegisterScoreUpdater,
		pingsched.RegisterEndpointMetaCache,
		pingsched.RegisterEndpointProvider,

		pinginfra.RegisterPingWorker,
		pinginfra.RegisterBodyChecker,
		pinginfra.RegisterRecordStatusWorker,

		pinggrpcserver.RegisterPingServer,

		pingservice.RegisterResponseChecker,

		pingservice.RegisterPingService,
		pingservice.RegisterLoopService,
		pingservice.RegisterPushEventService,
		pingservice.RegisterStaleLoopService,

		pingredis.RegisterStreamEventConsumer,
		pingservice.RegisterEventService,
		pinghandler.RegisterEndpointEventWorker,
		pinghandler.RegisterZSetWorkerRunner,
		pinghandler.RegisterStaleWorkerRunner,
		pinghandler.RegisterPushEventHandler,
	}
}

func RegisterPackages(injector do.Injector, configPath string, dev bool) {

	config.RegisterConfigPath(configPath)(injector)

	for _, p := range providers(dev) {
		p(injector)
	}
}
