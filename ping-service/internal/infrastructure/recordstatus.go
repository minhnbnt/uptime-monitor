package infrastructure

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	monitorrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
)

type RecordStatusWorker struct {
	statusStore           StatusStore
	eventRecorder         EventRecorder
	endpointStatusUpdater EndpointStatusUpdater
	logger                *slog.Logger
}

func RegisterRecordStatusWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RecordStatusWorker, error) {
		return &RecordStatusWorker{
			statusStore:           do.MustInvoke[*monitorrepo.RedisServerEventRepository](i),
			eventRecorder:         do.MustInvoke[*grpcclient.EventRecorderClient](i),
			endpointStatusUpdater: do.MustInvoke[*grpcclient.EndpointClient](i),
			logger:                do.MustInvoke[*slog.Logger](i),
		}, nil
	})
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

	if err := w.eventRecorder.RecordEvent(ctx, event.EndpointID, event.Status); err != nil {
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
