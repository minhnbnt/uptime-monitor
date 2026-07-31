package events

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type EndpointEventService struct {
	consumer    *redis.StreamEventConsumer
	multiplexer *EventMultiplexer
}

func RegisterEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointEventService, error) {

		sched := do.MustInvoke[*scheduler.ZSetScheduleRepository](i)
		cache := do.MustInvoke[*scheduler.ServerMetaCache](i)

		eventHandler := &ServerEventHandler{
			scheduler:   sched,
			serverCache: cache,
		}

		httpConfigHandler := &HTTPConfigEventHandler{cache: cache}

		consumer := do.MustInvoke[*redis.StreamEventConsumer](i)

		multiplexer := &EventMultiplexer{
			consumer: consumer,
			Handlers: map[string]EventHandler{
				"uptime.public.servers":             eventHandler,
				"uptime.public.server_http_configs": httpConfigHandler,
			},
		}

		return &EndpointEventService{
			consumer:    consumer,
			multiplexer: multiplexer,
		}, nil
	})
}

func (s *EndpointEventService) Run(ctx context.Context) {
	s.consumer.Run(ctx, s.multiplexer.GetTopics(), s.multiplexer)
}
