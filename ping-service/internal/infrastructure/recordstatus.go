package infrastructure

import (
	"context"
	"log/slog"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/grpcclient"
	monitorrepo "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis/cache"
)

type RecordStatusWorker struct {
	statusStore   StatusStore
	eventRecorder EventRecorder
	logger        *slog.Logger
}

func RegisterRecordStatusWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RecordStatusWorker, error) {
		return &RecordStatusWorker{
			statusStore:   do.MustInvoke[*monitorrepo.RedisServerEventRepository](i),
			eventRecorder: do.MustInvoke[*grpcclient.EventRecorderClient](i),
			logger:        do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (w *RecordStatusWorker) Record(ctx context.Context, event *domain.ServerEvent) error {

	event.Time = time.Now()

	w.logger.Debug("record status", "serverID", event.ServerID, "status", event.Status)

	lastStatus, err := w.statusStore.GetStatus(ctx, event.ServerID)
	if err == nil && lastStatus == event.Status {
		return nil
	}

	if err != nil {
		w.logger.Warn(
			"failed to get status from redis",
			slog.Uint64("serverID", uint64(event.ServerID)),
			slog.Any("error", err),
		)
	}

	if err := w.eventRecorder.RecordEvent(ctx, event.ServerID, event.Status); err != nil {
		return err
	}

	if err := w.statusStore.SetStatus(ctx, event.ServerID, event.Status); err != nil {
		return err
	}

	return nil
}
