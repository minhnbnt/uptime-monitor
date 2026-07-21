package service

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type EndpointEventHandler struct {
	scheduler *scheduler.ZSetScheduleRepository
	cache     *scheduler.EndpointMetaCache
}

func (e *EndpointEventHandler) OnCreate(ctx context.Context, endpoint domain.Endpoint) error {
	return e.scheduler.Register(ctx, &endpoint)
}

func (e *EndpointEventHandler) OnUpdate(ctx context.Context, endpoint domain.Endpoint) error {

	err := e.cache.Delete(ctx, endpoint.ID)
	if err != nil {
		return err
	}

	return e.scheduler.Register(ctx, &endpoint)
}

func (e *EndpointEventHandler) OnDelete(ctx context.Context, id uint) error {

	err := e.cache.Delete(ctx, id)
	if err != nil {
		return err
	}

	return e.scheduler.Unregister(ctx, id)
}

type EndpointEventService struct {
	consumer     *redis.StreamEventConsumer
	eventHandler *EndpointEventHandler
}

func RegisterEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointEventService, error) {

		sched := do.MustInvoke[*scheduler.ZSetScheduleRepository](i)
		cache := do.MustInvoke[*scheduler.EndpointMetaCache](i)
		eventHandler := &EndpointEventHandler{
			scheduler: sched,
			cache:     cache,
		}

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
