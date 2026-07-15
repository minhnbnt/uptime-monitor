package handler

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type EndpointEventWorker struct {
	service *service.EndpointEventService
}

func RegisterEndpointEventWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointEventWorker, error) {
		return &EndpointEventWorker{
			service: do.MustInvoke[*service.EndpointEventService](i),
		}, nil
	})
}

func (w *EndpointEventWorker) Run(ctx context.Context) {
	w.service.Run(ctx)
}
