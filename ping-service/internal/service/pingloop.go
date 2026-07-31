package service

import (
	"context"
	"log/slog"
	"net/url"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/k8sclient"
	scheduler "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/scheduler"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/utils"
)

type pingWorker interface {
	CheckObjectStatus(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error)
}

type urlResolver interface {
	ResolveURL(ctx context.Context, params *dto.CheckParams) (*url.URL, error)
}

type recordWorker interface {
	Record(ctx context.Context, event *domain.ServerEvent) error
}

type scoreUpdater interface {
	Update(ctx context.Context, serverID uint, nextScore int64) error
}

type PingLoopService struct {
	pingWorker         pingWorker
	recordStatusWorker recordWorker
	scoreUpdater       scoreUpdater
	urlResolver        urlResolver
	pingClient         *infrastructure.PingClient
	responseChecker    *ResponseChecker
	logger             *slog.Logger
}

func RegisterPingService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingLoopService, error) {
		return &PingLoopService{
			pingWorker:         do.MustInvoke[*k8sclient.K8sClient](i),
			recordStatusWorker: do.MustInvoke[*infrastructure.RecordStatusWorker](i),
			scoreUpdater:       do.MustInvoke[*scheduler.ScoreUpdater](i),
			urlResolver:        do.MustInvoke[*URLResolverService](i),
			pingClient:         do.MustInvoke[*infrastructure.PingClient](i),
			responseChecker:    do.MustInvoke[*ResponseChecker](i),
			logger:             do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *PingLoopService) checkServer(ctx context.Context, sv *domain.Server) (bool, error) {

	k8sParams := &dto.K8sObjectCheckParams{
		Namespace:     sv.Namespace,
		Kind:          sv.Kind,
		ObjectID:      sv.ObjectID,
		ContainerName: sv.ContainerName,
	}

	if sv.HTTPConfig != nil {
		return s.checkHTTPDNS(ctx, k8sParams, sv)
	}

	return s.pingWorker.CheckObjectStatus(ctx, k8sParams)
}

func (s *PingLoopService) checkHTTPDNS(ctx context.Context, k8sParams *dto.K8sObjectCheckParams, sv *domain.Server) (bool, error) {

	httpParams := &dto.HTTPCheckParams{
		Method:        sv.HTTPConfig.Method,
		Port:          sv.HTTPConfig.Port,
		EndpointPath:  sv.HTTPConfig.EndpointPath,
		ExpectedCode:  sv.HTTPConfig.ExpectedCode,
		BodyCheckExpr: sv.HTTPConfig.BodyCheckExpr,
	}

	params := &dto.CheckParams{
		K8sObjectCheckParams: *k8sParams,
		HTTPCheckParams:      httpParams,
	}

	url, err := s.urlResolver.ResolveURL(ctx, params)
	if err != nil {
		return false, err
	}

	resp, err := s.pingClient.Ping(ctx, sv.Timeout, httpParams.Method, url.String())
	if err != nil {
		return false, err
	}

	if err := s.responseChecker.CheckResponse(httpParams, *resp); err != nil {
		return false, err
	}

	return true, nil
}

func (s *PingLoopService) pingAndRecordServer(ctx context.Context, task PingTask) {

	sv := task.Server

	defer func() {
		if r := recover(); r != nil {
			s.logger.Error("panic recovered", slog.Any("error", r))
		}
	}()

	isUp, pingErr := s.checkServer(ctx, sv)
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

func (s *PingLoopService) Run(ctx context.Context, channel <-chan PingTask) {

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
