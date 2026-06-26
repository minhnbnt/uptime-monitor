package handler

import (
	"context"
	"iter"
	"sync"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/ping/service"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

type PingService interface {
	Ping(ctx context.Context, method, url string) (int, error)
	Record(ctx context.Context, event *domain.ServerEvent) error
}

type LoopRunner interface {
	Run(ctx context.Context, dueHandler service.DueHandler) error
}

type ZSetWorkerRunner struct {
	loopService LoopRunner
	pingService PingService
	logger      logger.Logger
}

func RegisterZSetWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetWorkerRunner, error) {
		return &ZSetWorkerRunner{
			loopService: do.MustInvoke[*service.LoopService](i),
			pingService: do.MustInvoke[*service.PingService](i),
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

	if pingErr != nil {
		r.logger.Error(
			"ping failed",
			logger.String("url", ep.URL),
			logger.String("method", ep.Method),
			logger.Int("status_code", statusCode),
			logger.Int("expected_code", ep.ExpectedCode),
			logger.Int64("endpoint_id", int64(ep.ID)),
			logger.Error(pingErr),
		)
	}

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
