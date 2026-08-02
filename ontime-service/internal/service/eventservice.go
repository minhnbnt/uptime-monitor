package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
)

type EventRecorder interface {
	Save(ctx context.Context, event *domain.ServerEvent) error
}

type EventRepository interface {
	GetCurrentStatuses(ctx context.Context, serverIDs []uint) ([]repository.CurrentStatus, error)
	CountByStatus(ctx context.Context, serverIDs []uint) (online, offline int64, err error)
	CountByStatusByUserID(ctx context.Context, userID uuid.UUID) (online, offline int64, err error)
}

type EventService struct {
	recorder EventRecorder
	repo     EventRepository
	logger   *slog.Logger
}

func NewEventService(r EventRecorder, repo EventRepository, logger *slog.Logger) *EventService {
	return &EventService{recorder: r, repo: repo, logger: logger}
}

func RegisterEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventService, error) {
		return NewEventService(
			do.MustInvoke[*repository.ServerEventRepository](i),
			do.MustInvoke[*repository.EventRepository](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func (s *EventService) RecordEvent(ctx context.Context, req dto.RecordEventRequest) error {

	s.logger.Info("record event", "serverID", req.ServerID, "status", req.Status)

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate event id: %w", err)
	}

	event := &domain.ServerEvent{
		ID:       id,
		Time:     time.Now(),
		ServerID: req.ServerID,
		Status:   domain.ServerStatus(req.Status),
	}

	return s.recorder.Save(ctx, event)
}

func (s *EventService) GetCurrentStatuses(ctx context.Context, serverIDs []uint) ([]dto.EndpointStatus, error) {

	rows, err := s.repo.GetCurrentStatuses(ctx, serverIDs)
	if err != nil {
		return nil, err
	}

	results := lo.Map(rows, func(r repository.CurrentStatus, _ int) dto.EndpointStatus {
		return dto.EndpointStatus{
			ServerID: r.ServerID,
			Status:   dto.ServerStatus(r.Status),
		}
	})

	return results, nil
}

func (s *EventService) CountByStatus(ctx context.Context, serverIDs []uint) (online, offline int64, err error) {
	return s.repo.CountByStatus(ctx, serverIDs)
}

func (s *EventService) CountByStatusByUserID(ctx context.Context, userID uuid.UUID) (online, offline int64, err error) {
	return s.repo.CountByStatusByUserID(ctx, userID)
}

var (
	_ EventRecorder   = (*repository.ServerEventRepository)(nil)
	_ EventRepository = (*repository.EventRepository)(nil)
)
