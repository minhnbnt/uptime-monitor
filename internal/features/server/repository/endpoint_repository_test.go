package repository

import (
	"context"
	"errors"
	"testing"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type mockScheduler struct {
	registerFn      func(ctx context.Context, endpoint *domain.Endpoint) error
	registerBatchFn func(ctx context.Context, endpoints []domain.Endpoint) error
	unregisterFn    func(ctx context.Context, endpointID uint) error
}

func (m *mockScheduler) Register(ctx context.Context, endpoint *domain.Endpoint) error {
	return m.registerFn(ctx, endpoint)
}

func (m *mockScheduler) RegisterBatch(ctx context.Context, endpoints []domain.Endpoint) error {
	if m.registerBatchFn != nil {
		return m.registerBatchFn(ctx, endpoints)
	}
	return nil
}

func (m *mockScheduler) Unregister(ctx context.Context, endpointID uint) error {
	return m.unregisterFn(ctx, endpointID)
}

type mockMetaCache struct {
	setMultiFn func(ctx context.Context, endpoints []domain.Endpoint) error
	deleteFn   func(ctx context.Context, id uint) error
}

func (m *mockMetaCache) SetMulti(ctx context.Context, endpoints []domain.Endpoint) error {
	if m.setMultiFn != nil {
		return m.setMultiFn(ctx, endpoints)
	}
	return nil
}

func (m *mockMetaCache) Delete(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
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
	endpointRepo := &EndpointRepository{
		db: testDB,
		scheduler: &mockScheduler{
			registerBatchFn: func(_ context.Context, _ []domain.Endpoint) error { return nil },
		},
		metaCache: &mockMetaCache{},
	}

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
		metaCache: &mockMetaCache{},
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

func TestEndpointRepository_UpsertEndpoint_CallsMetaCacheDelete(t *testing.T) {
	truncateTables(t)
	serverRepo := &ServerRepository{db: testDB}

	s := &domain.Server{Name: "upsert-cache-test", CreatedByID: 1}
	if err := serverRepo.Create(context.Background(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	var deletedEndpointID uint
	repo := &EndpointRepository{
		db: testDB,
		scheduler: &mockScheduler{
			registerFn: func(_ context.Context, _ *domain.Endpoint) error { return nil },
		},
		metaCache: &mockMetaCache{
			deleteFn: func(_ context.Context, id uint) error {
				deletedEndpointID = id
				return nil
			},
		},
	}

	ep := domain.Endpoint{
		ServerID:     s.ID,
		URL:          "https://example.com/cache-delete",
		Method:       "GET",
		Interval:     60000000000,
		Timeout:      10000000000,
		ExpectedCode: 200,
	}
	if err := repo.UpsertEndpoint(context.Background(), ep); err != nil {
		t.Fatalf("UpsertEndpoint: %v", err)
	}

	if deletedEndpointID == 0 {
		t.Error("metaCache.Delete not called")
	}
}

func TestEndpointRepository_BatchCreateEndpoints_CallsRegisterBatch(t *testing.T) {
	truncateTables(t)
	serverRepo := &ServerRepository{db: testDB}

	servers := []domain.Server{
		{Name: "reg-batch-a", CreatedByID: 1},
		{Name: "reg-batch-b", CreatedByID: 1},
	}
	if err := serverRepo.BatchCreateServers(context.Background(), servers); err != nil {
		t.Fatalf("create servers: %v", err)
	}

	var capturedEndpoints []domain.Endpoint
	endpointRepo := &EndpointRepository{
		db: testDB,
		scheduler: &mockScheduler{
			registerBatchFn: func(_ context.Context, endpoints []domain.Endpoint) error {
				capturedEndpoints = endpoints
				return nil
			},
		},
		metaCache: &mockMetaCache{
			setMultiFn: func(_ context.Context, _ []domain.Endpoint) error { return nil },
		},
	}

	endpoints := []domain.Endpoint{
		{ServerID: servers[0].ID, URL: "https://reg-batch-a.com", Method: "GET"},
		{ServerID: servers[1].ID, URL: "https://reg-batch-b.com", Method: "POST"},
	}
	if err := endpointRepo.BatchCreateEndpoints(context.Background(), endpoints); err != nil {
		t.Fatalf("BatchCreateEndpoints: %v", err)
	}

	if len(capturedEndpoints) != 2 {
		t.Fatalf("RegisterBatch called with %d endpoints, want 2", len(capturedEndpoints))
	}
	if capturedEndpoints[0].ID == 0 {
		t.Error("RegisterBatch received endpoint with ID=0, expected auto-incremented ID")
	}
	if capturedEndpoints[0].URL != "https://reg-batch-a.com" {
		t.Errorf("captured[0].URL = %q, want %q", capturedEndpoints[0].URL, "https://reg-batch-a.com")
	}
}

func TestEndpointRepository_BatchCreateEndpoints_CallsSetMulti(t *testing.T) {
	truncateTables(t)
	serverRepo := &ServerRepository{db: testDB}

	servers := []domain.Server{
		{Name: "cache-batch-a", CreatedByID: 1},
		{Name: "cache-batch-b", CreatedByID: 1},
	}
	if err := serverRepo.BatchCreateServers(context.Background(), servers); err != nil {
		t.Fatalf("create servers: %v", err)
	}

	var capturedEndpoints []domain.Endpoint
	endpointRepo := &EndpointRepository{
		db: testDB,
		scheduler: &mockScheduler{
			registerBatchFn: func(_ context.Context, _ []domain.Endpoint) error { return nil },
		},
		metaCache: &mockMetaCache{
			setMultiFn: func(_ context.Context, endpoints []domain.Endpoint) error {
				capturedEndpoints = endpoints
				return nil
			},
		},
	}

	endpoints := []domain.Endpoint{
		{ServerID: servers[0].ID, URL: "https://cache-batch-a.com", Method: "GET"},
		{ServerID: servers[1].ID, URL: "https://cache-batch-b.com", Method: "POST"},
	}
	if err := endpointRepo.BatchCreateEndpoints(context.Background(), endpoints); err != nil {
		t.Fatalf("BatchCreateEndpoints: %v", err)
	}

	if len(capturedEndpoints) != 2 {
		t.Fatalf("SetMulti called with %d endpoints, want 2", len(capturedEndpoints))
	}
	if capturedEndpoints[0].ID == 0 {
		t.Error("SetMulti received endpoint with ID=0, expected auto-incremented ID")
	}
	if capturedEndpoints[0].URL != "https://cache-batch-a.com" {
		t.Errorf("captured[0].URL = %q, want %q", capturedEndpoints[0].URL, "https://cache-batch-a.com")
	}
}

func TestEndpointRepository_BatchCreateEndpoints_RegisterBatchError(t *testing.T) {
	truncateTables(t)
	serverRepo := &ServerRepository{db: testDB}

	servers := []domain.Server{
		{Name: "reg-err", CreatedByID: 1},
	}
	if err := serverRepo.BatchCreateServers(context.Background(), servers); err != nil {
		t.Fatalf("create servers: %v", err)
	}

	endpointRepo := &EndpointRepository{
		db: testDB,
		scheduler: &mockScheduler{
			registerBatchFn: func(_ context.Context, _ []domain.Endpoint) error {
				return errors.New("scheduler unavailable")
			},
		},
		metaCache: &mockMetaCache{
			setMultiFn: func(_ context.Context, _ []domain.Endpoint) error { return nil },
		},
	}

	endpoints := []domain.Endpoint{
		{ServerID: servers[0].ID, URL: "https://reg-err.com", Method: "GET"},
	}
	err := endpointRepo.BatchCreateEndpoints(context.Background(), endpoints)
	if err == nil {
		t.Fatal("expected error from RegisterBatch, got nil")
	}
}

func TestEndpointRepository_BatchCreateEndpoints_SetMultiError(t *testing.T) {
	truncateTables(t)
	serverRepo := &ServerRepository{db: testDB}

	servers := []domain.Server{
		{Name: "cache-err", CreatedByID: 1},
	}
	if err := serverRepo.BatchCreateServers(context.Background(), servers); err != nil {
		t.Fatalf("create servers: %v", err)
	}

	endpointRepo := &EndpointRepository{
		db: testDB,
		scheduler: &mockScheduler{
			registerBatchFn: func(_ context.Context, _ []domain.Endpoint) error { return nil },
		},
		metaCache: &mockMetaCache{
			setMultiFn: func(_ context.Context, _ []domain.Endpoint) error {
				return errors.New("redis unavailable")
			},
		},
	}

	endpoints := []domain.Endpoint{
		{ServerID: servers[0].ID, URL: "https://cache-err.com", Method: "GET"},
	}
	err := endpointRepo.BatchCreateEndpoints(context.Background(), endpoints)
	if err == nil {
		t.Fatal("expected error from SetMulti, got nil")
	}
}
