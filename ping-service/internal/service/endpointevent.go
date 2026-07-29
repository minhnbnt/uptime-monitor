package service

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
)

type ServerEventHandler struct {
	scheduler   *scheduler.ZSetScheduleRepository
	serverCache *scheduler.ServerMetaCache
}

func (e *ServerEventHandler) OnCreate(ctx context.Context, server domain.Server) error {
	return e.scheduler.Register(ctx, &server)
}

func (e *ServerEventHandler) OnUpdate(ctx context.Context, server domain.Server) error {

	err := e.serverCache.Delete(ctx, server.ID)
	if err != nil {
		return err
	}

	return e.scheduler.Register(ctx, &server)
}

func (e *ServerEventHandler) OnDelete(ctx context.Context, id uint) error {

	err := e.serverCache.Delete(ctx, id)
	if err != nil {
		return err
	}

	return e.scheduler.Unregister(ctx, id)
}

type EndpointEventService struct {
	consumer     *redis.StreamEventConsumer
	eventHandler *ServerEventHandler
}

func RegisterEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointEventService, error) {

		sched := do.MustInvoke[*scheduler.ZSetScheduleRepository](i)
		cache := do.MustInvoke[*scheduler.ServerMetaCache](i)
		eventHandler := &ServerEventHandler{
			scheduler:   sched,
			serverCache: cache,
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
