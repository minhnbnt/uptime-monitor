package events

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis/cache"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis/consumer"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis/scheduler"
)

type EndpointEventService struct {
	consumer    *consumer.StreamEventConsumer
	multiplexer *EventMultiplexer
}

func RegisterEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointEventService, error) {

		sched := do.MustInvoke[*scheduler.ZSetScheduleRepository](i)
		cache := do.MustInvoke[*cache.ServerMetaCache](i)
		offsetStore := do.MustInvoke[*consumer.RedisOffsetStore](i)

		eventHandler := &ServerEventHandler{
			scheduler:   sched,
			serverCache: cache,
			offsetStore: offsetStore,
		}

		httpConfigHandler := &HTTPConfigEventHandler{cache: cache, offsetStore: offsetStore}

		streamConsumer := do.MustInvoke[*consumer.StreamEventConsumer](i)

		multiplexer := &EventMultiplexer{
			consumer: streamConsumer,
			Handlers: map[string]EventHandler{
				"uptime.public.servers":             eventHandler,
				"uptime.public.server_http_configs": httpConfigHandler,
			},
		}

		return &EndpointEventService{
			consumer:    streamConsumer,
			multiplexer: multiplexer,
		}, nil
	})
}

func (s *EndpointEventService) Run(ctx context.Context) {
	s.consumer.Run(ctx, s.multiplexer.GetTopics(), s.multiplexer)
}
