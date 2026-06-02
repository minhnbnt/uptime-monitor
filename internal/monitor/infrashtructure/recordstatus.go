package infrashtructure

import (
	"context"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/monitor/infrashtructure/repository"
)

type RecordPingStatusWorker struct {
	redisRepo *repository.RedisServerEventRepository
	dbRepo    *repository.ServerEventRepository
	logger    logger.Logger
}

func RegisterRecordPingStatusWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RecordPingStatusWorker, error) {
		return &RecordPingStatusWorker{
			redisRepo: do.MustInvoke[*repository.RedisServerEventRepository](i),
			dbRepo:    do.MustInvoke[*repository.ServerEventRepository](i),
			logger:    do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (w *RecordPingStatusWorker) Record(ctx context.Context, event *domain.ServerEvent) error {

	event.Time = time.Now()

	lastStatus, err := w.redisRepo.GetStatus(ctx, event.EndpointID)
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

	if err := w.dbRepo.Save(ctx, event); err != nil {
		return err
	}

	if err := w.redisRepo.SetStatus(ctx, event.EndpointID, event.Status); err != nil {
		return err
	}

	return nil
}
