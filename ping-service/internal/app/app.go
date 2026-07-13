package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/grpcclient"
	pinghandler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/handler"
	pinginfra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/infrastructure"
	pingrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/repository"
	pingsched "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/scheduler"
	pingservice "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/service"
)

func providers(dev bool) []func(do.Injector) {
	return []func(do.Injector){
		config.RegisterLogger(dev),
		config.RegisterGORMDB,
		config.RegisterRedisClient,

		pingrepo.RegisterServerEventRepository,
		pingrepo.RegisterRedisServerEventRepository,

		pingsched.RegisterZSetScheduleRepository,
		pingsched.RegisterScoreUpdater,
		pingsched.RegisterEndpointProvider,

		pinginfra.RegisterPingWorker,
		pinginfra.RegisterRecordStatusWorker,

		pingservice.RegisterPingService,
		pingservice.RegisterLoopService,

		pinghandler.RegisterZSetWorkerRunner,
		pinghandler.RegisterStreamEventConsumer,
	}
}

func RegisterPackages(injector do.Injector, configPath string, dev bool) {
	config.RegisterConfigPath(configPath)(injector)

	for _, p := range providers(dev) {
		p(injector)
	}

	do.Provide(injector, func(i do.Injector) (*grpcclient.EndpointClient, error) {
		cfg := do.MustInvoke[*config.Config](i)
		log := do.MustInvoke[*slog.Logger](i)

		client, err := grpcclient.NewEndpointClient(cfg.GRPC.ServerAddr)
		if err != nil {
			return nil, fmt.Errorf("grpc client: %w", err)
		}

		log.Info("connected to gRPC server", slog.String("addr", cfg.GRPC.ServerAddr))
		return client, nil
	})
}

func RunZSetWorker(ctx context.Context, i do.Injector) {
	runner := do.MustInvoke[*pinghandler.ZSetWorkerRunner](i)
	runner.RunZSetWorker(ctx)
}

func RunStreamConsumer(ctx context.Context, i do.Injector) {
	consumer := do.MustInvoke[*pinghandler.StreamEventConsumer](i)
	consumer.Run(ctx)
}
