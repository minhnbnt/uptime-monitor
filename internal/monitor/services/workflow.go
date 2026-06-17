package services

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	infra "github.com/minhnbnt/uptime-monitor/internal/monitor/infrastructure"
)

type PingService struct {
	pingWorker         *infra.PingWorker
	recordStatusWorker *infra.RecordStatusWorker
}

func RegisterPingService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingService, error) {
		return &PingService{
			pingWorker:         do.MustInvoke[*infra.PingWorker](i),
			recordStatusWorker: do.MustInvoke[*infra.RecordStatusWorker](i),
		}, nil
	})
}

func (s *PingService) Ping(ctx context.Context, method, url string) (int, error) {
	return s.pingWorker.Ping(ctx, method, url)
}

func (s *PingService) Record(ctx context.Context, event *domain.ServerEvent) error {
	return s.recordStatusWorker.Record(ctx, event)
}
