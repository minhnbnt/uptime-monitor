package service

import (
	"context"
	"errors"
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
)

type fakeStatusRepo struct {
	statuses  []repository.CurrentStatus
	statusErr error
}

func (f *fakeStatusRepo) GetCurrentStatuses(_ context.Context, _ []uint) ([]repository.CurrentStatus, error) {
	return f.statuses, f.statusErr
}

func (f *fakeStatusRepo) CountByStatus(_ context.Context, _ []uint) (int64, int64, error) {
	return 0, 0, nil
}

func (f *fakeStatusRepo) CountByStatusByUserID(_ context.Context, _ uint) (int64, int64, error) {
	return 0, 0, nil
}

var _ EventRepository = (*fakeStatusRepo)(nil)

type fakeOwnerRepo struct {
	owned    []uint
	ownedErr error
}

func (f *fakeOwnerRepo) GetOwnedServerIDs(_ context.Context, _ uint, _ []uint) ([]uint, error) {
	return f.owned, f.ownedErr
}

func (f *fakeOwnerRepo) GetOwnedServers(_ context.Context, _ uint, _ []uint) ([]repository.OwnedServer, error) {
	return nil, nil
}

func (f *fakeOwnerRepo) ListByUser(_ context.Context, _ uint) ([]repository.OwnedServer, error) {
	return nil, nil
}

var _ ServerOwnerRepository = (*fakeOwnerRepo)(nil)

type noopRecorder struct{}

func (noopRecorder) Save(_ context.Context, _ *domain.ServerEvent) error { return nil }

var _ EventRecorder = (*noopRecorder)(nil)

func TestGetServersStatusesTooLarge(t *testing.T) {
	ids := make([]uint, MaxStatusIDs+1)
	svc := NewEventService(noopRecorder{}, &fakeStatusRepo{}, &fakeOwnerRepo{})

	_, err := svc.GetServersStatuses(context.Background(), 1, ids)
	if !errors.Is(err, apperrors.ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
}

func TestGetServersStatusesForbiddenWhenNotOwned(t *testing.T) {
	// user owns only [10], but requests [10, 11]
	svc := NewEventService(noopRecorder{}, &fakeStatusRepo{}, &fakeOwnerRepo{owned: []uint{10}})

	_, err := svc.GetServersStatuses(context.Background(), 1, []uint{10, 11})
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestGetServersStatusesReturnsLatestAndUnknown(t *testing.T) {
	svc := NewEventService(noopRecorder{}, &fakeStatusRepo{
		statuses: []repository.CurrentStatus{{EndpointID: 10, Status: "ON"}},
	}, &fakeOwnerRepo{
		owned: []uint{10, 11},
	})

	got, err := svc.GetServersStatuses(context.Background(), 1, []uint{10, 11})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
	if got[0].EndpointID != 10 || got[0].Status != dto.ServerStatusOn {
		t.Errorf("got[0] = %+v, want {10 ON}", got[0])
	}
	if got[1].EndpointID != 11 || got[1].Status != dto.ServerStatusUnknown {
		t.Errorf("got[1] = %+v, want {11 UNKNOWN}", got[1])
	}
}
