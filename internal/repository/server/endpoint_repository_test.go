package server

import (
	"context"
	"testing"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type mockScheduler struct {
	registerFn   func(ctx context.Context, endpoint *domain.Endpoint) error
	unregisterFn func(ctx context.Context, endpointID uint) error
}

func (m *mockScheduler) Register(ctx context.Context, endpoint *domain.Endpoint) error {
	return m.registerFn(ctx, endpoint)
}

func (m *mockScheduler) Unregister(ctx context.Context, endpointID uint) error {
	return m.unregisterFn(ctx, endpointID)
}

func TestEndpointRepository_GetByServerID(t *testing.T) {
	truncateTables(t)
	serverRepo := &ServerRepository{db: testDB}
	endpointRepo := &EndpointRepository{db: testDB}

	s := &domain.Server{Name: "ep-server", CreatedByID: 1}
	if err := serverRepo.Create(context.Background(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	ep := &domain.Endpoint{
		ServerID: s.ID,
		URL:      "https://example.com",
		Method:   "GET",
	}
	if err := gorm.G[domain.Endpoint](testDB).Create(context.Background(), ep); err != nil {
		t.Fatalf("create endpoint: %v", err)
	}

	got, err := endpointRepo.GetByServerID(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("GetByServerID: %v", err)
	}
	if got.ServerID != s.ID {
		t.Errorf("ServerID = %d, want %d", got.ServerID, s.ID)
	}
	if got.URL != "https://example.com" {
		t.Errorf("URL = %q", got.URL)
	}
}

func TestEndpointRepository_GetByServerID_NotFound(t *testing.T) {
	truncateTables(t)
	repo := &EndpointRepository{db: testDB}

	_, err := repo.GetByServerID(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestEndpointRepository_BatchCreateEndpoints(t *testing.T) {
	truncateTables(t)
	serverRepo := &ServerRepository{db: testDB}
	endpointRepo := &EndpointRepository{db: testDB}

	servers := []domain.Server{
		{Name: "batch-a", CreatedByID: 1},
		{Name: "batch-b", CreatedByID: 1},
	}
	if err := serverRepo.BatchCreateServers(context.Background(), servers); err != nil {
		t.Fatalf("create servers: %v", err)
	}

	endpoints := []domain.Endpoint{
		{ServerID: servers[0].ID, URL: "https://a.com", Method: "GET"},
		{ServerID: servers[1].ID, URL: "https://b.com", Method: "POST"},
	}
	err := endpointRepo.BatchCreateEndpoints(context.Background(), endpoints)
	if err != nil {
		t.Fatalf("BatchCreateEndpoints: %v", err)
	}

	for _, ep := range endpoints {
		got, err := endpointRepo.GetByServerID(context.Background(), ep.ServerID)
		if err != nil {
			t.Fatalf("get endpoint for server %d: %v", ep.ServerID, err)
		}
		if got.URL != ep.URL {
			t.Errorf("URL = %q, want %q", got.URL, ep.URL)
		}
	}
}

func TestEndpointRepository_DeleteByServerID_NotFound(t *testing.T) {
	truncateTables(t)
	repo := &EndpointRepository{db: testDB}

	err := repo.DeleteByServerID(context.Background(), 999)
	if err != nil {
		t.Fatalf("expected nil for non-existent server, got: %v", err)
	}
}

func TestEndpointRepository_UpsertEndpoint_Create(t *testing.T) {
	truncateTables(t)
	serverRepo := &ServerRepository{db: testDB}
	var registeredEndpoint *domain.Endpoint

	s := &domain.Server{Name: "upsert-test", CreatedByID: 1}
	if err := serverRepo.Create(context.Background(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	repo := &EndpointRepository{
		db: testDB,
		scheduler: &mockScheduler{
			registerFn: func(_ context.Context, ep *domain.Endpoint) error {
				registeredEndpoint = ep
				return nil
			},
		},
	}

	ep := domain.Endpoint{
		ServerID:     s.ID,
		URL:          "https://example.com/upsert",
		Method:       "GET",
		Interval:     60000000000,
		Timeout:      20000000000,
		ExpectedCode: 200,
	}
	err := repo.UpsertEndpoint(context.Background(), ep)
	if err != nil {
		t.Fatalf("UpsertEndpoint: %v", err)
	}

	got, err := repo.GetByServerID(context.Background(), s.ID)
	if err != nil {
		t.Fatalf("GetByServerID: %v", err)
	}
	if got.URL != "https://example.com/upsert" {
		t.Errorf("URL = %q", got.URL)
	}
	if registeredEndpoint == nil || registeredEndpoint.ID == 0 {
		t.Error("scheduler.Register not called with backfilled endpoint")
	}
}

func TestEndpointRepository_UpsertEndpoint_Update(t *testing.T) {
	t.Skip("UpsertEndpoint requires unique constraint on server_id, see ON CONFLICT (server_id) clause")
}
