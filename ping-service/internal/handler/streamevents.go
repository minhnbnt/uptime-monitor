package handler

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service/events"
)

type EndpointEventWorker struct {
	service *events.EndpointEventService
}

func RegisterEndpointEventWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointEventWorker, error) {
		return &EndpointEventWorker{
			service: do.MustInvoke[*events.EndpointEventService](i),
		}, nil
	})
}

func (w *EndpointEventWorker) Run(ctx context.Context) {
	w.service.Run(ctx)
}
