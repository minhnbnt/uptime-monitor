package infrastructure

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	monitorrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/grpcclient"
)

type RecordStatusWorker struct {
	statusStore           StatusStore
	eventSaver            EventSaver
	endpointStatusUpdater EndpointStatusUpdater
	logger                *slog.Logger
}

func RegisterRecordStatusWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RecordStatusWorker, error) {
		return &RecordStatusWorker{
			statusStore:           do.MustInvoke[*monitorrepo.RedisServerEventRepository](i),
			eventSaver:            do.MustInvoke[*monitorrepo.ServerEventRepository](i),
			endpointStatusUpdater: do.MustInvoke[*grpcclient.EndpointClient](i),
			logger:                do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (w *RecordStatusWorker) handleOnCacheMiss(ctx context.Context, event *domain.ServerEvent) (bool, error) {

	dbStatus, err := w.eventSaver.GetLatestStatus(ctx, event.EndpointID)
	if err != nil {

		w.logger.Warn(
			"failed to get latest status from db",
			slog.Int64("endpointID", int64(event.EndpointID)),
			slog.Any("error", err),
		)

		// Unsure — proceed to save; ontime calculator handles duplicates
		return true, nil
	}

	if dbStatus == event.Status {
		return false, w.statusStore.SetStatus(ctx, event.EndpointID, event.Status)
	}

	// Different status or no events yet — real transition
	return true, nil
}

func (w *RecordStatusWorker) Record(ctx context.Context, event *domain.ServerEvent) error {

	event.Time = time.Now()

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

	isCacheMiss := lastStatus == ""
	if isCacheMiss {

		proceed, err := w.handleOnCacheMiss(ctx, event)
		if err != nil {
			return err
		}

		if !proceed {
			return nil
		}
	}

	if err := w.eventSaver.Save(ctx, event); err != nil {
		return err
	}

	if err := w.statusStore.SetStatus(ctx, event.EndpointID, event.Status); err != nil {
		return err
	}

	if err := w.endpointStatusUpdater.UpdateMonitorStatus(ctx, event.EndpointID, event.Status); err != nil {
		return err
	}

	return nil
}
