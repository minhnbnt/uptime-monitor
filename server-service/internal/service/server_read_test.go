package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

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
