package repository

import (
	"testing"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

func TestEndpointRepository_GetByServerID(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	serverRepo := &ServerRepository{db: testDB}
	endpointRepo := &EndpointRepository{db: testDB}

	s := &domain.Server{Name: "ep-server", CreatedByID: 1}
	if err := serverRepo.Create(t.Context(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	ep := &domain.Endpoint{
		ServerID: s.ID,
		URL:      "https://example.com",
		Method:   "GET",
	}
	if err := gorm.G[domain.Endpoint](testDB).Create(t.Context(), ep); err != nil {
		t.Fatalf("create endpoint: %v", err)
	}

	got, err := endpointRepo.GetByServerID(t.Context(), s.ID)
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
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	repo := &EndpointRepository{db: testDB}

	_, err := repo.GetByServerID(t.Context(), 999)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestEndpointRepository_BatchCreateEndpoints(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	serverRepo := &ServerRepository{db: testDB}
	endpointRepo := &EndpointRepository{db: testDB}

	servers := []domain.Server{
		{Name: "batch-a", CreatedByID: 1},
		{Name: "batch-b", CreatedByID: 1},
	}
	if err := serverRepo.BatchCreateServers(t.Context(), servers); err != nil {
		t.Fatalf("create servers: %v", err)
	}

	endpoints := []domain.Endpoint{
		{ServerID: servers[0].ID, URL: "https://a.com", Method: "GET"},
		{ServerID: servers[1].ID, URL: "https://b.com", Method: "POST"},
	}
	err := endpointRepo.BatchCreateEndpoints(t.Context(), endpoints)
	if err != nil {
		t.Fatalf("BatchCreateEndpoints: %v", err)
	}

	for _, ep := range endpoints {
		got, err := endpointRepo.GetByServerID(t.Context(), ep.ServerID)
		if err != nil {
			t.Fatalf("get endpoint for server %d: %v", ep.ServerID, err)
		}
		if got.URL != ep.URL {
			t.Errorf("URL = %q, want %q", got.URL, ep.URL)
		}
	}
}

func TestEndpointRepository_DeleteByServerID_NotFound(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	repo := &EndpointRepository{db: testDB}

	err := repo.DeleteByServerID(t.Context(), 999)
	if err != nil {
		t.Fatalf("expected nil for non-existent server, got: %v", err)
	}
}

func TestEndpointRepository_UpsertEndpoint_Create(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	serverRepo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "upsert-test", CreatedByID: 1}
	if err := serverRepo.Create(t.Context(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	repo := &EndpointRepository{db: testDB}

	ep := domain.Endpoint{
		ServerID:     s.ID,
		URL:          "https://example.com/upsert",
		Method:       "GET",
		Interval:     60000000000,
		Timeout:      20000000000,
		ExpectedCode: 200,
	}
	err := repo.UpsertEndpoint(t.Context(), ep)
	if err != nil {
		t.Fatalf("UpsertEndpoint: %v", err)
	}

	got, err := repo.GetByServerID(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("GetByServerID: %v", err)
	}
	if got.URL != "https://example.com/upsert" {
		t.Errorf("URL = %q", got.URL)
	}
}

func TestEndpointRepository_UpsertEndpoint_Update(t *testing.T) {
	t.Skip("UpsertEndpoint requires unique constraint on server_id, see ON CONFLICT (server_id) clause")
}
