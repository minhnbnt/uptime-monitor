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

type mockResolveRepo struct {
	ids []uint
	err error
}

func (m *mockResolveRepo) List(_ context.Context, _ uint, _, _ int) ([]domain.Server, error) {
	return nil, nil
}

func (m *mockResolveRepo) Count(_ context.Context, _ uint) (int64, error) {
	return 0, nil
}

func (m *mockResolveRepo) GetByID(_ context.Context, _ uint) (*domain.Server, error) {
	return nil, nil
}

func (m *mockResolveRepo) ResolveByIDs(_ context.Context, _ uint, _ []uint) ([]uint, error) {
	return m.ids, m.err
}

func TestServerReader_ResolveServers(t *testing.T) {

	t.Run("returns owned ids", func(t *testing.T) {
		repo := &mockResolveRepo{ids: []uint{7, 9}}
		reader := &ServerReader{
			serverRepository: repo,
			logger:           slog.Default(),
		}

		ids, err := reader.ResolveServers(t.Context(), 42, []uint{7, 8, 9})
		if err != nil {
			t.Fatalf("ResolveServers error: %v", err)
		}
		if len(ids) != 2 || ids[0] != 7 || ids[1] != 9 {
			t.Errorf("ids = %v, want [7 9]", ids)
		}
	})

	t.Run("maps repo error to internal", func(t *testing.T) {
		reader := &ServerReader{
			serverRepository: &mockResolveRepo{err: errors.New("boom")},
			logger:           slog.Default(),
		}

		if _, err := reader.ResolveServers(t.Context(), 42, []uint{7}); !errors.Is(err, apperrors.ErrInternal) {
			t.Errorf("err = %v, want ErrInternal", err)
		}
	})
}

type mockStatusClient struct {
	statuses map[uint]domain.ServerStatus
	err      error
}

func (m *mockStatusClient) GetCurrentStatuses(_ context.Context, _ []uint) (map[uint]domain.ServerStatus, error) {
	return m.statuses, m.err
}

func (m *mockStatusClient) CountByStatus(_ context.Context, _ uint) (int64, int64, error) {
	return 0, 0, nil
}

func TestServerReader_applyStatuses(t *testing.T) {

	withEndpoint := &dto.Server{ID: 1, Endpoint: &dto.Endpoint{ID: 10}}
	noEndpoint := &dto.Server{ID: 2}
	servers := []*dto.Server{withEndpoint, noEndpoint}

	reader := &ServerReader{
		statusClient: &mockStatusClient{statuses: map[uint]domain.ServerStatus{10: domain.StatusOn}},
		logger:       slog.Default(),
	}

	reader.applyStatuses(t.Context(), servers)

	if servers[0].MonitorStatus != domain.StatusOn {
		t.Errorf("server with endpoint: want ON, got %q", servers[0].MonitorStatus)
	}
	if servers[1].MonitorStatus != "" {
		t.Errorf("server without endpoint: want empty, got %q", servers[1].MonitorStatus)
	}
}

func TestServerReader_applyStatuses_errorIsBestEffort(t *testing.T) {

	servers := []*dto.Server{{ID: 1, Endpoint: &dto.Endpoint{ID: 10}}}

	reader := &ServerReader{
		statusClient: &mockStatusClient{err: errors.New("boom")},
		logger:       slog.Default(),
	}

	reader.applyStatuses(t.Context(), servers)
	if servers[0].MonitorStatus != "" {
		t.Errorf("on error status should stay empty, got %q", servers[0].MonitorStatus)
	}
}

func TestServerReader_applyStatuses_noEndpoint(t *testing.T) {

	servers := []*dto.Server{{ID: 1}}
	reader := &ServerReader{
		statusClient: &mockStatusClient{statuses: map[uint]domain.ServerStatus{10: domain.StatusOff}},
		logger:       slog.Default(),
	}

	reader.applyStatuses(t.Context(), servers)
	if servers[0].MonitorStatus != "" {
		t.Errorf("no endpoint: want empty, got %q", servers[0].MonitorStatus)
	}
}
