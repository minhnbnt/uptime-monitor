package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
	scheduler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/redis/scheduler"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service/httpcheck"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/utils"
)

type pingWorker interface {
	CheckObjectStatus(ctx context.Context, params *dto.K8sObjectCheckParams) (running bool, err error)
}

type recordWorker interface {
	Record(ctx context.Context, event *domain.ServerEvent) error
}

type scoreUpdater interface {
	Update(ctx context.Context, serverID uint, nextScore int64) error
}

type httpChecker interface {
	Check(ctx context.Context, sv *dto.Server) (ok bool, err error)
}

type PingLoopService struct {
	pingWorker         pingWorker
	recordStatusWorker recordWorker
	scoreUpdater       scoreUpdater
	httpChecker        httpChecker
	logger             *slog.Logger
}

func RegisterPingService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingLoopService, error) {
		return &PingLoopService{
			pingWorker:         do.MustInvoke[*k8sclient.K8sClient](i),
			recordStatusWorker: do.MustInvoke[*infrastructure.RecordStatusWorker](i),
			scoreUpdater:       do.MustInvoke[*scheduler.ScoreUpdater](i),
			httpChecker:        do.MustInvoke[*httpcheck.HTTPChecker](i),
			logger:             do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *PingLoopService) checkServer(ctx context.Context, sv *dto.Server) (running bool, err error) {

	if sv.HTTPCheckParams != nil {
		return s.httpChecker.Check(ctx, sv)
	}

	return s.pingWorker.CheckObjectStatus(ctx, &sv.K8sObjectCheckParams)
}

func (s *PingLoopService) pingAndRecordServer(ctx context.Context, task dto.PingTask) {

	sv := task.Server

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic recovered", slog.Any("error", r))
		}
	}()

	isUp, pingErr := s.checkServer(ctx, sv)
	if errors.Is(pingErr, httpcheck.ErrStaleDomain) {
		s.logger.Info(
			"stale cached domain, invalidated; skipping event",
			slog.String("namespace", sv.Namespace),
			slog.String("kind", sv.Kind),
			slog.String("object_id", sv.ObjectID),
			slog.Uint64("server_id", uint64(sv.ID)),
		)
		return
	}
	if pingErr != nil {
		s.logger.Warn(
			"ping failed",
			slog.String("namespace", sv.Namespace),
			slog.String("kind", sv.Kind),
			slog.String("object_id", sv.ObjectID),
			slog.Uint64("server_id", uint64(sv.ID)),
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
			slog.Uint64("server_id", uint64(sv.ID)),
			slog.Any("error", err),
		)
		return
	}

	event := domain.ServerEvent{
		ID:       id,
		ServerID: sv.ID,
		Status:   status,
	}

	if err := s.Record(ctx, &event); err != nil {
		s.logger.Error(
			"record event",
			slog.Uint64("server_id", uint64(sv.ID)),
			slog.Any("error", err),
		)
	}

	nextScore, err := utils.NextExecutionTime(sv.ID, sv.Interval)
	if err != nil {
		s.logger.Error(
			"calculate next score",
			slog.Uint64("server_id", uint64(sv.ID)),
			slog.Any("error", err),
		)
		return
	}

	if err := s.scoreUpdater.Update(ctx, sv.ID, nextScore.UnixMilli()); err != nil {
		s.logger.Error(
			"update score",
			slog.Uint64("server_id", uint64(sv.ID)),
			slog.Any("error", err),
		)
	}
}

func (s *PingLoopService) Record(ctx context.Context, event *domain.ServerEvent) error {
	return s.recordStatusWorker.Record(ctx, event)
}

func (s *PingLoopService) Run(ctx context.Context, channel <-chan dto.PingTask) {

	for {
		select {
		case task, ok := <-channel:
			if !ok {
				return
			}

			if task.Server == nil {
				continue
			}

			s.pingAndRecordServer(ctx, task)

		case <-ctx.Done():
			return
		}
	}
}
