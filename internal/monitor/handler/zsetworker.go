package handler

import (
	"context"
	"iter"
	"sync"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/monitor/services"
)

type ZSetWorkerRunner struct {
	loopService *services.LoopService
	pingService *services.PingService
	logger      logger.Logger
}

func RegisterZSetWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetWorkerRunner, error) {
		return &ZSetWorkerRunner{
			loopService: do.MustInvoke[*services.LoopService](i),
			pingService: do.MustInvoke[*services.PingService](i),
			logger:      do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (r *ZSetWorkerRunner) RunZSetWorker(ctx context.Context) error {

	handler := func(ctx context.Context, endpoints iter.Seq[*domain.Endpoint]) {

		waitGroup := sync.WaitGroup{}
		defer waitGroup.Wait()

		for ep := range endpoints {
			waitGroup.Go(func() { r.pingAndRecordEndpoint(ctx, ep) })
		}
	}

	return r.loopService.Run(ctx, handler)
}

func (r *ZSetWorkerRunner) pingAndRecordEndpoint(ctx context.Context, ep *domain.Endpoint) {

	statusCode, pingErr := r.pingService.Ping(ctx, ep.Method, ep.URL)

	status := domain.StatusOn
	if pingErr != nil || statusCode != ep.ExpectedCode {
		status = domain.StatusOff
	}

	id, err := uuid.NewV7()
	if err != nil {
		r.logger.Error(
			"generate event id",
			logger.Int64("endpoint", int64(ep.ID)),
			logger.Error(err),
		)
		return
	}

	event := &domain.ServerEvent{
		ID:         id,
		EndpointID: ep.ID,
		Status:     status,
	}

	if err := r.pingService.Record(ctx, event); err != nil {
		r.logger.Error(
			"record event",
			logger.Int64("endpoint", int64(ep.ID)),
			logger.Error(err),
		)
	}
}
