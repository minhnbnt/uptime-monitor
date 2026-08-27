package infrastructure

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
	monitorrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/repository"
)

type FreshnessToucher interface {
	Touch(ctx context.Context, endpointID uint, lease time.Duration) error
}

type RecordStatusWorker struct {
	statusStore   StatusStore
	eventRecorder EventRecorder
	freshness     FreshnessToucher
	logger        *slog.Logger
}

func RegisterRecordStatusWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RecordStatusWorker, error) {
		return &RecordStatusWorker{
			statusStore:   do.MustInvoke[*monitorrepo.RedisServerEventRepository](i),
			eventRecorder: do.MustInvoke[*grpcclient.EventRecorderClient](i),
			freshness:     do.MustInvoke[*monitorrepo.FreshnessStore](i),
			logger:        do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (w *RecordStatusWorker) RecordWithTimestamp(ctx context.Context, event *domain.EventWithTimestamp, freshness time.Duration) error {

	endpointID := event.Event.EndpointID

	// Refresh the staleness deadline before anything else so every recorded
	// event — push or poll — counts as evidence of life.
	if err := w.freshness.Touch(ctx, endpointID, freshness); err != nil {
		w.logger.Warn(
			"failed to touch freshness",
			slog.Int64("endpointID", int64(endpointID)),
			slog.Any("error", err),
		)
	}

	// Historical/stale events are boundary markers inserted at a past
	// timestamp, so they are recorded as-is without the live last-status
	// dedupe (which assumes now-ordered processing and would misbehave if an
	// event is inserted in the middle of the timeline).
	if err := w.eventRecorder.RecordEventAt(ctx, endpointID, event.Event.Status, event.Time); err != nil {
		return err
	}

	if err := w.statusStore.SetStatus(ctx, endpointID, event.Event.Status); err != nil {
		return err
	}

	return nil
}

func (w *RecordStatusWorker) Record(ctx context.Context, event *domain.ServerEvent, freshness time.Duration) error {

	// Refresh the staleness deadline before anything else so every recorded
	// event — push or poll — counts as evidence of life.
	if err := w.freshness.Touch(ctx, event.EndpointID, freshness); err != nil {
		w.logger.Warn(
			"failed to touch freshness",
			slog.Int64("endpointID", int64(event.EndpointID)),
			slog.Any("error", err),
		)
	}

	lastStatus, err := w.statusStore.GetStatus(ctx, event.EndpointID)
	if err != nil {
		w.logger.Warn(
			"failed to get status from redis",
			slog.Int64("endpointID", int64(event.EndpointID)),
			slog.Any("error", err),
		)
		return nil
	}

	if lastStatus == event.Status {
		return nil
	}

	if err := w.eventRecorder.RecordEventAt(ctx, event.EndpointID, event.Status, time.Now()); err != nil {
		return err
	}

	if err := w.statusStore.SetStatus(ctx, event.EndpointID, event.Status); err != nil {
		return err
	}

	return nil
}
