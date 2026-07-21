package handler

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/service"
)

type OwnershipWorker struct {
	service *service.OwnershipService
}

func RegisterOwnershipWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OwnershipWorker, error) {
		return &OwnershipWorker{
			service: do.MustInvoke[*service.OwnershipService](i),
		}, nil
	})
}

func (w *OwnershipWorker) Run(ctx context.Context) {
	w.service.Run(ctx)
}
