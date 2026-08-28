package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
)

type EventRecorder interface {
	Save(ctx context.Context, event *domain.ServerEvent) error
}

// ServerOwnerRepository authorizes server ownership and supplies each server's
// creation time for ontime windowing, all from the ontime DB (server_owners).
// The concrete implementation lives in the repository package.
type ServerOwnerRepository interface {
	GetOwnedServerIDs(ctx context.Context, userID uint, serverIDs []uint) ([]uint, error)
	GetOwnedServers(ctx context.Context, userID uint, serverIDs []uint) ([]repository.OwnedServer, error)
	ListByUser(ctx context.Context, userID uint) ([]repository.OwnedServer, error)
	CountOwnedServers(ctx context.Context, userID uint) (int64, error)
}

type EventRepository interface {
	GetCurrentStatuses(ctx context.Context, endpointIDs []uint) ([]repository.CurrentStatus, error)
	CountByStatus(ctx context.Context, endpointIDs []uint) (online, offline int64, err error)
	CountByStatusByUserID(ctx context.Context, userID uint) (online, offline int64, err error)
}

type EventService struct {
	recorder  EventRecorder
	repo      EventRepository
	ownerRepo ServerOwnerRepository
}

func NewEventService(r EventRecorder, repo EventRepository, ownerRepo ServerOwnerRepository) *EventService {
	return &EventService{recorder: r, repo: repo, ownerRepo: ownerRepo}
}

func RegisterEventService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventService, error) {
		return NewEventService(
			do.MustInvoke[*repository.ServerEventRepository](i),
			do.MustInvoke[*repository.EventRepository](i),
			do.MustInvoke[*repository.ServerOwnerRepository](i),
		), nil
	})
}

func (s *EventService) RecordEvent(ctx context.Context, req dto.RecordEventRequest) error {

	id, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate event id: %w", err)
	}

	event := &domain.ServerEvent{
		ID:         id,
		Time:       req.Time,
		EndpointID: req.EndpointID,
		Status:     domain.ServerStatus(req.Status),
	}

	return s.recorder.Save(ctx, event)
}

func (s *EventService) GetCurrentStatuses(ctx context.Context, endpointIDs []uint) ([]dto.EndpointStatus, error) {

	rows, err := s.repo.GetCurrentStatuses(ctx, endpointIDs)
	if err != nil {
		return nil, err
	}

	results := lo.Map(rows, func(r repository.CurrentStatus, _ int) dto.EndpointStatus {
		return dto.EndpointStatus{
			EndpointID: r.EndpointID,
			Status:     dto.ServerStatus(r.Status),
		}
	})

	return results, nil
}

// MaxStatusIDs caps how many servers can be queried in one GetServersStatuses call.
const MaxStatusIDs = 100

func (s *EventService) GetServersStatuses(ctx context.Context, userID uint, serverIDs []uint) ([]dto.EndpointStatus, error) {

	if len(serverIDs) > MaxStatusIDs {
		return nil, apperrors.ErrBadRequest
	}

	owned, err := s.ownerRepo.GetOwnedServerIDs(ctx, userID, serverIDs)
	if err != nil {
		return nil, err
	}

	if len(owned) != len(serverIDs) {
		return nil, apperrors.ErrForbidden
	}

	rows, err := s.repo.GetCurrentStatuses(ctx, serverIDs)
	if err != nil {
		return nil, err
	}

	byID := make(map[uint]dto.EndpointStatus, len(rows))
	for _, r := range rows {
		byID[r.EndpointID] = dto.EndpointStatus{
			EndpointID: r.EndpointID,
			Status:     dto.ServerStatus(r.Status),
		}
	}

	out := make([]dto.EndpointStatus, 0, len(serverIDs))
	for _, id := range serverIDs {

		st, ok := byID[id]
		if !ok {
			st = dto.EndpointStatus{
				EndpointID: id,
				Status:     dto.ServerStatusUnknown,
			}
		}

		out = append(out, st)
	}

	return out, nil
}

func (s *EventService) CountByStatus(ctx context.Context, endpointIDs []uint) (online, offline int64, err error) {
	return s.repo.CountByStatus(ctx, endpointIDs)
}

func (s *EventService) CountByStatusByUserID(ctx context.Context, userID uint) (online, offline int64, err error) {
	return s.repo.CountByStatusByUserID(ctx, userID)
}

// CountServersByStatus returns the total number of servers a user owns plus the
// online/offline split, sourced entirely from the ontime DB (server_owners +
// server_events). This is the server-service /servers/count endpoint moved here.
func (s *EventService) CountServersByStatus(ctx context.Context, userID uint) (total, online, offline int64, err error) {

	total, err = s.ownerRepo.CountOwnedServers(ctx, userID)
	if err != nil {
		return 0, 0, 0, err
	}

	if total == 0 {
		return 0, 0, 0, nil
	}

	online, offline, err = s.repo.CountByStatusByUserID(ctx, userID)
	if err != nil {
		return 0, 0, 0, err
	}

	return total, online, offline, nil
}

var (
	_ EventRecorder   = (*repository.ServerEventRepository)(nil)
	_ EventRepository = (*repository.EventRepository)(nil)
)
