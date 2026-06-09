package infrastructure

import (
	"context"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/repository/monitor"
)

type RecordStatusWorker struct {
	statusStore StatusStore
	eventSaver  EventSaver
	logger      logger.Logger
}

func RegisterRecordStatusWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RecordStatusWorker, error) {
		return &RecordStatusWorker{
			statusStore: do.MustInvoke[*monitorrepo.RedisServerEventRepository](i),
			eventSaver:  do.MustInvoke[*monitorrepo.ServerEventRepository](i),
			logger:      do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (w *RecordStatusWorker) Record(ctx context.Context, event *domain.ServerEvent) error {

	event.Time = time.Now()

	lastStatus, err := w.statusStore.GetStatus(ctx, event.EndpointID)
	if err != nil {
		w.logger.Warn(
			"failed to get status from redis",
			logger.Int64("endpointID", int64(event.EndpointID)),
			logger.Error(err),
		)
		return nil
	}

	if lastStatus == event.Status {
		return nil
	}

	if err := w.eventSaver.Save(ctx, event); err != nil {
		return err
	}

	if err := w.statusStore.SetStatus(ctx, event.EndpointID, event.Status); err != nil {
		return err
	}

	return nil
}
