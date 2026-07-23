package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	infra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
)

type pingWorker interface {
	Ping(ctx context.Context, ep *domain.Endpoint) (*infra.Response, error)
}

type recordWorker interface {
	Record(ctx context.Context, event *domain.ServerEvent) error
}

type PingLoopService struct {
	pingWorker         pingWorker
	responseChecker    *ResponseChecker
	recordStatusWorker recordWorker
	logger             *slog.Logger
}

func RegisterPingService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingLoopService, error) {
		return &PingLoopService{
			pingWorker:         do.MustInvoke[*infra.PingClient](i),
			responseChecker:    do.MustInvoke[*ResponseChecker](i),
			recordStatusWorker: do.MustInvoke[*infra.RecordStatusWorker](i),
			logger:             do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *PingLoopService) pingAndRecordEndpoint(ctx context.Context, ep *domain.Endpoint) {

	isUp, pingErr := s.Ping(ctx, ep)

	if pingErr != nil {
		s.logger.Warn(
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
		s.logger.Error(
			"generate event id",
			slog.Int64("endpoint", int64(ep.ID)),
			slog.Any("error", err),
		)
		return
	}

	event := domain.ServerEvent{
		ID:         id,
		EndpointID: ep.ID,
		Status:     status,
	}

	if err := s.Record(ctx, &event); err != nil {
		s.logger.Error(
			"record event",
			slog.Int64("endpoint", int64(ep.ID)),
			slog.Any("error", err),
		)
	}
}

func (s *PingLoopService) Ping(ctx context.Context, ep *domain.Endpoint) (bool, error) {

	out, err := s.pingWorker.Ping(ctx, ep)
	if err != nil {
		return false, err
	}

	if err := s.responseChecker.CheckResponse(*ep, *out); err != nil {
		return false, err
	}

	return true, nil
}

func (s *PingLoopService) Record(ctx context.Context, event *domain.ServerEvent) error {
	return s.recordStatusWorker.Record(ctx, event)
}

func (s *PingLoopService) Run(ctx context.Context, channel <-chan *domain.Endpoint) {

	for {
		select {
		case ep, ok := <-channel:
			if !ok {
				return
			}

			s.pingAndRecordEndpoint(ctx, ep)

		case <-ctx.Done():
			return
		}
	}
}
