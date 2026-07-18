package handler

import (
	"context"
	"iter"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type PingService interface {
	Ping(ctx context.Context, ep *domain.Endpoint) (bool, error)
	Record(ctx context.Context, event *domain.ServerEvent) error
}

type LoopRunner interface {
	Run(ctx context.Context, dueHandler service.DueHandler)
}

type ZSetWorkerRunner struct {
	loopService LoopRunner
	pingService PingService
	logger      *slog.Logger
}

func RegisterZSetWorkerRunner(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ZSetWorkerRunner, error) {
		return &ZSetWorkerRunner{
			loopService: do.MustInvoke[*service.LoopService](i),
			pingService: do.MustInvoke[*service.PingService](i),
			logger:      do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (r *ZSetWorkerRunner) RunZSetWorker(ctx context.Context) {

	handler := func(ctx context.Context, endpoints iter.Seq[*domain.Endpoint]) {

		waitGroup := sync.WaitGroup{}
		defer waitGroup.Wait()

		for ep := range endpoints {
			waitGroup.Go(func() { r.pingAndRecordEndpoint(ctx, ep) })
		}
	}

	r.loopService.Run(ctx, handler)
}

func (r *ZSetWorkerRunner) pingAndRecordEndpoint(ctx context.Context, ep *domain.Endpoint) {

	isUp, pingErr := r.pingService.Ping(ctx, ep)

	if pingErr != nil {
		r.logger.Warn(
			"ping failed",
			slog.String("url", ep.URL),
			slog.String("method", ep.Method),
			slog.Int("expected_code", ep.ExpectedCode),
			slog.Int64("endpoint_id", int64(ep.ID)),
			slog.Any("error", pingErr),
		)
	}

	status := domain.StatusOn
	if pingErr != nil || !isUp {
		status = domain.StatusOff
	}

	id, err := uuid.NewV7()
	if err != nil {
		r.logger.Error(
			"generate event id",
			slog.Int64("endpoint", int64(ep.ID)),
			slog.Any("error", err),
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
			slog.Int64("endpoint", int64(ep.ID)),
			slog.Any("error", err),
		)
	}
}
