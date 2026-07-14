package repository

import (
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

func TestIntegration_DeleteByServerID_FullCleanup(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	serverRepo := &ServerRepository{db: testDB}
	s := &domain.Server{Name: "delete-full-cleanup", CreatedByID: 1}
	if err := serverRepo.Create(t.Context(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	endpointRepo := &EndpointRepository{db: testDB}

	endpoints := []domain.Endpoint{
		{ServerID: s.ID, URL: "https://delete-test.com", Method: "GET", Interval: 30 * time.Second},
	}
	if err := endpointRepo.BatchCreateEndpoints(t.Context(), endpoints); err != nil {
		t.Fatalf("BatchCreateEndpoints: %v", err)
	}

	_, err := endpointRepo.GetByServerID(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("GetByServerID: %v", err)
	}

	if err := endpointRepo.DeleteByServerID(t.Context(), s.ID); err != nil {
		t.Fatalf("DeleteByServerID: %v", err)
	}

	_, err = endpointRepo.GetByServerID(t.Context(), s.ID)
	if err == nil {
		t.Error("expected error getting deleted endpoint, got nil")
	}
}

func TestIntegration_UpsertEndpoint_UpdatePath(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	serverRepo := &ServerRepository{db: testDB}
	endpointRepo := &EndpointRepository{db: testDB}

	s := &domain.Server{Name: "upsert-update-path", CreatedByID: 1}
	if err := serverRepo.Create(t.Context(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	ep := domain.Endpoint{
		ServerID: s.ID, URL: "https://original.com", Method: "GET",
		Interval: 30 * time.Second, ExpectedCode: 200,
	}
	if err := endpointRepo.UpsertEndpoint(t.Context(), ep); err != nil {
		t.Fatalf("first UpsertEndpoint: %v", err)
	}

	created, err := endpointRepo.GetByServerID(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("GetByServerID: %v", err)
	}
	origID := created.ID

	if created.URL != "https://original.com" {
		t.Errorf("URL = %q, want %q", created.URL, "https://original.com")
	}

	// Second Upsert (UPDATE) — same server_id
	ep2 := domain.Endpoint{
		ServerID: s.ID, URL: "https://updated.com", Method: "POST",
		Interval: 60 * time.Second, ExpectedCode: 201,
	}
	if err := endpointRepo.UpsertEndpoint(t.Context(), ep2); err != nil {
		t.Fatalf("second UpsertEndpoint: %v", err)
	}

	updated, err := endpointRepo.GetByServerID(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("GetByServerID after second Upsert: %v", err)
	}

	if updated.ID != origID {
		t.Errorf("ID changed: %d -> %d (expected same row)", origID, updated.ID)
	}
	if updated.URL != "https://updated.com" {
		t.Errorf("URL = %q, want %q", updated.URL, "https://updated.com")
	}
	if updated.Method != "POST" {
		t.Errorf("Method = %q, want %q", updated.Method, "POST")
	}
	if updated.ExpectedCode != 201 {
		t.Errorf("ExpectedCode = %d, want %d", updated.ExpectedCode, 201)
	}
	if updated.Interval != 60*time.Second {
		t.Errorf("Interval = %v, want %v", updated.Interval, 60*time.Second)
	}

	var count int64
	testDB.Model(&domain.Endpoint{}).Where("server_id = ?", s.ID).Count(&count)
	if count != 1 {
		t.Errorf("endpoint count = %d, want 1", count)
	}
}

func TestIntegration_DeleteByServerID_NotFound(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	endpointRepo := &EndpointRepository{db: testDB}

	err := endpointRepo.DeleteByServerID(t.Context(), 999)
	if err != nil {
		t.Fatalf("expected nil for non-existent server, got: %v", err)
	}
}
