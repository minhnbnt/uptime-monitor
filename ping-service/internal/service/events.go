package service

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
	"github.com/samber/do/v2"
)

type EndpointEventHandler struct {
	scheduler *scheduler.ZSetScheduleRepository
}

func (e *EndpointEventHandler) OnCreate(ctx context.Context, endpoint domain.Endpoint) error {
	return e.scheduler.Register(ctx, &endpoint)
}

func (e *EndpointEventHandler) OnUpdate(ctx context.Context, endpoint domain.Endpoint) error {
	return e.scheduler.Register(ctx, &endpoint)
}

func (e *EndpointEventHandler) OnDelete(ctx context.Context, id uint) error {
	return e.scheduler.Unregister(ctx, id)
}

type EndpointEventService struct {
	consumer     *redis.StreamEventConsumer
	eventHandler *EndpointEventHandler
}

func RegisterEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointEventService, error) {

		scheduler := do.MustInvoke[*scheduler.ZSetScheduleRepository](i)
		eventHandler := &EndpointEventHandler{scheduler: scheduler}

		consumer := do.MustInvoke[*redis.StreamEventConsumer](i)

		return &EndpointEventService{
			consumer:     consumer,
			eventHandler: eventHandler,
		}, nil
	})
}

func (s *EndpointEventService) Run(ctx context.Context) {
	s.consumer.Run(ctx, s.eventHandler)
}
