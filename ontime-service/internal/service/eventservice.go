package service

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/recorder"
	eventrepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
)

type EventRecorder interface {
	RecordEvent(ctx context.Context, endpointID uint, status domain.ServerStatus) error
}

type EventRepository interface {
	GetCurrentStatuses(ctx context.Context, endpointIDs []uint) ([]eventrepo.CurrentStatus, error)
	CountByStatus(ctx context.Context, endpointIDs []uint) (online, offline int64, err error)
}

type EventService struct {
	recorder EventRecorder
	repo     EventRepository
}

func NewEventService(r EventRecorder, repo EventRepository) *EventService {
	return &EventService{recorder: r, repo: repo}
}

func RegisterEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventService, error) {
		return NewEventService(
			do.MustInvoke[*recorder.DedupRecorder](i),
			do.MustInvoke[*eventrepo.EventRepository](i),
		), nil
	})
}

func (s *EventService) RecordEvent(ctx context.Context, req dto.RecordEventRequest) error {
	return s.recorder.RecordEvent(ctx, req.EndpointID, domain.ServerStatus(req.Status))
}

func (s *EventService) GetCurrentStatuses(ctx context.Context, endpointIDs []uint) ([]dto.EndpointStatus, error) {

	rows, err := s.repo.GetCurrentStatuses(ctx, endpointIDs)
	if err != nil {
		return nil, err
	}

	results := lo.Map(rows, func(r eventrepo.CurrentStatus, _ int) dto.EndpointStatus {
		return dto.EndpointStatus{
			EndpointID: r.EndpointID,
			Status:     dto.ServerStatus(r.Status),
		}
	})

	return results, nil
}

func (s *EventService) CountByStatus(ctx context.Context, endpointIDs []uint) (online, offline int64, err error) {
	return s.repo.CountByStatus(ctx, endpointIDs)
}

var (
	_ EventRecorder   = (*recorder.DedupRecorder)(nil)
	_ EventRepository = (*eventrepo.EventRepository)(nil)
)
