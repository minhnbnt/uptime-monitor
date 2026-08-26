package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
)

type fakeEndpointRepo struct {
	upserted *domain.Endpoint
	deleted  []uint
}

func (f *fakeEndpointRepo) UpsertEndpoint(_ context.Context, e domain.Endpoint) error {
	f.upserted = &e
	return nil
}

func (f *fakeEndpointRepo) DeleteByServerID(_ context.Context, id uint) error {
	f.deleted = append(f.deleted, id)
	return nil
}

type fakeSetServerRepo struct {
	mockResolveRepo
	server *domain.Server
}

func (f *fakeSetServerRepo) GetByID(context.Context, uint) (*domain.Server, error) {
	return f.server, nil
}

func newTestEndpointService(srv *fakeSetServerRepo, ep *fakeEndpointRepo) *EndpointService {
	return &EndpointService{
		serverRepository:   srv,
		endpointRepository: ep,
		logger:             slog.Default(),
	}
}

func pullRequest() dto.SetCheckMethodRequest {
	return dto.SetCheckMethodRequest{
		Method:       dto.CheckMethodPull,
		URL:          "https://example.com/health",
		HTTPMethod:   "GET",
		Interval:     30_000_000_000,
		Timeout:      10_000_000_000,
		ExpectedCode: 200,
	}
}

func TestEndpointService_SetCheckMethod(t *testing.T) {

	t.Run("push deletes the endpoint instead of upserting", func(t *testing.T) {
		srv := &fakeSetServerRepo{server: &domain.Server{CreatedByID: 42}}
		ep := &fakeEndpointRepo{}
		svc := newTestEndpointService(srv, ep)

		err := svc.SetCheckMethod(t.Context(), 7, 42, dto.SetCheckMethodRequest{Method: dto.CheckMethodPush})
		if err != nil {
			t.Fatalf("SetCheckMethod error: %v", err)
		}
		if len(ep.deleted) != 1 || ep.deleted[0] != 7 {
			t.Errorf("deleted = %v, want [7]", ep.deleted)
		}
		if ep.upserted != nil {
			t.Errorf("upserted = %v, want nil", ep.upserted)
		}
	})

	t.Run("pull upserts the endpoint and keeps it", func(t *testing.T) {
		srv := &fakeSetServerRepo{server: &domain.Server{CreatedByID: 42}}
		ep := &fakeEndpointRepo{}
		svc := newTestEndpointService(srv, ep)

		req := pullRequest()
		if err := svc.SetCheckMethod(t.Context(), 7, 42, req); err != nil {
			t.Fatalf("SetCheckMethod error: %v", err)
		}
		if ep.upserted == nil || ep.upserted.URL != req.URL {
			t.Errorf("upserted = %v, want url %q", ep.upserted, req.URL)
		}
		if len(ep.deleted) != 0 {
			t.Errorf("deleted = %v, want none", ep.deleted)
		}
	})

	t.Run("rejects non-owner", func(t *testing.T) {
		srv := &fakeSetServerRepo{server: &domain.Server{CreatedByID: 42}}
		ep := &fakeEndpointRepo{}
		svc := newTestEndpointService(srv, ep)

		if err := svc.SetCheckMethod(t.Context(), 7, 99, pullRequest()); !errors.Is(err, apperrors.ErrForbidden) {
			t.Errorf("err = %v, want ErrForbidden", err)
		}
	})
}
