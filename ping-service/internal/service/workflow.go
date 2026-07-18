package service

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	infra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
)

type PingService struct {
	pingWorker         *infra.PingWorker
	responseChecker    *ResponseChecker
	recordStatusWorker *infra.RecordStatusWorker
}

func RegisterPingService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingService, error) {
		return &PingService{
			pingWorker:         do.MustInvoke[*infra.PingWorker](i),
			responseChecker:    do.MustInvoke[*ResponseChecker](i),
			recordStatusWorker: do.MustInvoke[*infra.RecordStatusWorker](i),
		}, nil
	})
}

func (s *PingService) Ping(ctx context.Context, ep *domain.Endpoint) (bool, error) {

	out, err := s.pingWorker.Ping(ctx, ep)
	if err != nil {
		return false, err
	}

	if err := s.responseChecker.CheckResponse(*ep, *out); err != nil {
		return false, err
	}

	return true, nil
}

func (s *PingService) Record(ctx context.Context, event *domain.ServerEvent) error {
	return s.recordStatusWorker.Record(ctx, event)
}
