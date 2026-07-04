package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	monitorrepo "github.com/minhnbnt/uptime-monitor/internal/features/ping/repository"
	"github.com/minhnbnt/uptime-monitor/internal/features/ping/scheduler"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

func TestIntegration_DeleteByServerID_FullCleanup(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	serverRepo := &ServerRepository{db: testDB}
	s := &domain.Server{Name: "delete-full-cleanup", CreatedByID: 1}
	if err := serverRepo.Create(t.Context(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	zsetScheduler := scheduler.NewZSetScheduleRepository(testRedis)
	metaCache := scheduler.NewEndpointMetaCache(testRedis)
	statusStore := monitorrepo.NewRedisServerEventRepository(testRedis)

	endpointRepo := NewEndpointRepositoryWithDeps(
		testDB, zsetScheduler, statusStore, metaCache,
	)

	endpoints := []domain.Endpoint{
		{ServerID: s.ID, URL: "https://delete-test.com", Method: "GET", Interval: 30 * time.Second},
	}
	if err := endpointRepo.BatchCreateEndpoints(t.Context(), endpoints); err != nil {
		t.Fatalf("BatchCreateEndpoints: %v", err)
	}

	created, err := endpointRepo.GetByServerID(t.Context(), s.ID)
	if err != nil {
		t.Fatalf("GetByServerID: %v", err)
	}

	if err := statusStore.SetStatus(t.Context(), created.ID, domain.StatusOn); err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	// pre-conditions
	zscore, err := testRedis.ZScore(t.Context(), "scheduler:queue", fmt.Sprint(created.ID)).Result()
	if err != nil {
		t.Fatalf("ZScore before delete: %v", err)
	}
	if zscore <= 0 {
		t.Errorf("expected positive ZScore before delete, got %f", zscore)
	}

	status, err := statusStore.GetStatus(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetStatus before delete: %v", err)
	}
	if status != domain.StatusOn {
		t.Errorf("status = %q, want %q before delete", status, domain.StatusOn)
	}

	cached, err := metaCache.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("metaCache.Get before delete: %v", err)
	}
	if cached.URL != "https://delete-test.com" {
		t.Errorf("cached URL = %q, want %q", cached.URL, "https://delete-test.com")
	}

	// act
	if err := endpointRepo.DeleteByServerID(t.Context(), s.ID); err != nil {
		t.Fatalf("DeleteByServerID: %v", err)
	}

	// assert DB
	_, err = endpointRepo.GetByServerID(t.Context(), s.ID)
	if err == nil {
		t.Error("expected error getting deleted endpoint, got nil")
	}

	// assert ZSET
	_, err = testRedis.ZScore(t.Context(), "scheduler:queue", fmt.Sprint(created.ID)).Result()
	if err != redis.Nil {
		t.Fatalf("expected ZSET member removed, ZScore err=%v", err)
	}

	// assert status
	status, err = statusStore.GetStatus(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("GetStatus after delete: %v", err)
	}
	if status != "" {
		t.Errorf("status = %q, want empty after delete", status)
	}

	// assert meta cache
	_, err = metaCache.Get(t.Context(), created.ID)
	if err == nil {
		t.Error("expected metaCache.Get error after delete, got nil")
	}
}

func TestIntegration_DeleteByServerID_NoRedisArtifacts(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	serverRepo := &ServerRepository{db: testDB}
	s := &domain.Server{Name: "delete-no-redis", CreatedByID: 1}
	if err := serverRepo.Create(t.Context(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	endpointRepo := NewEndpointRepositoryWithDeps(
		testDB,
		scheduler.NewZSetScheduleRepository(testRedis),
		monitorrepo.NewRedisServerEventRepository(testRedis),
		scheduler.NewEndpointMetaCache(testRedis),
	)

	ep := domain.Endpoint{ServerID: s.ID, URL: "https://no-redis.com", Method: "GET"}
	if err := gorm.G[domain.Endpoint](testDB).Create(t.Context(), &ep); err != nil {
		t.Fatalf("create endpoint: %v", err)
	}

	if err := endpointRepo.DeleteByServerID(t.Context(), s.ID); err != nil {
		t.Fatalf("DeleteByServerID: %v", err)
	}

	_, err := endpointRepo.GetByServerID(t.Context(), s.ID)
	if err == nil {
		t.Error("expected error getting deleted endpoint, got nil")
	}
}

func TestIntegration_UpsertEndpoint_UpdatePath(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	serverRepo := &ServerRepository{db: testDB}
	zsetScheduler := scheduler.NewZSetScheduleRepository(testRedis)
	metaCache := scheduler.NewEndpointMetaCache(testRedis)
	statusStore := monitorrepo.NewRedisServerEventRepository(testRedis)
	endpointRepo := NewEndpointRepositoryWithDeps(testDB, zsetScheduler, statusStore, metaCache)

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

	// ZSET has entry after first Upsert
	score, err := testRedis.ZScore(t.Context(), "scheduler:queue", fmt.Sprint(origID)).Result()
	if err != nil {
		t.Fatalf("ZScore after first Upsert: %v", err)
	}
	if score <= 0 {
		t.Errorf("score = %f, want > 0", score)
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

	// Only 1 row for this server_id
	var count int64
	testDB.Model(&domain.Endpoint{}).Where("server_id = ?", s.ID).Count(&count)
	if count != 1 {
		t.Errorf("endpoint count = %d, want 1", count)
	}

	// ZSET re-registered
	score2, err := testRedis.ZScore(t.Context(), "scheduler:queue", fmt.Sprint(origID)).Result()
	if err != nil {
		t.Fatalf("ZScore after second Upsert: %v", err)
	}
	if score2 <= 0 {
		t.Errorf("score = %f, want > 0 after update", score2)
	}

	// Meta cache cleared
	_, err = metaCache.Get(t.Context(), origID)
	if err == nil {
		t.Error("expected meta cache empty after Upsert")
	}
}

func TestIntegration_UpsertEndpoint_ClearsPreExistingCache(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	serverRepo := &ServerRepository{db: testDB}
	zsetScheduler := scheduler.NewZSetScheduleRepository(testRedis)
	metaCache := scheduler.NewEndpointMetaCache(testRedis)
	statusStore := monitorrepo.NewRedisServerEventRepository(testRedis)
	endpointRepo := NewEndpointRepositoryWithDeps(testDB, zsetScheduler, statusStore, metaCache)

	s := &domain.Server{Name: "upsert-pre-cache", CreatedByID: 1}
	if err := serverRepo.Create(t.Context(), s); err != nil {
		t.Fatalf("create server: %v", err)
	}

	ep := domain.Endpoint{
		ServerID: s.ID, URL: "https://pre-cache.com", Method: "GET",
		Interval: 30 * time.Second,
	}
	if err := endpointRepo.UpsertEndpoint(t.Context(), ep); err != nil {
		t.Fatalf("first UpsertEndpoint: %v", err)
	}

	created, _ := endpointRepo.GetByServerID(t.Context(), s.ID)

	// Simulate read-through cache population (as EndpointProvider would do)
	cacheEntry := domain.Endpoint{
		Model:        gorm.Model{ID: created.ID},
		URL:          "https://pre-cache.com",
		Method:       "GET",
		ExpectedCode: 200,
		Interval:     30 * time.Second,
	}
	if err := metaCache.Set(t.Context(), cacheEntry); err != nil {
		t.Fatalf("metaCache.Set: %v", err)
	}

	if _, err := metaCache.Get(t.Context(), created.ID); err != nil {
		t.Fatalf("metaCache.Get before Upsert: %v", err)
	}

	// Upsert should delete the pre-existing cache
	ep2 := domain.Endpoint{
		ServerID: s.ID, URL: "https://after-cache.com", Method: "PUT",
		Interval: 30 * time.Second,
	}
	if err := endpointRepo.UpsertEndpoint(t.Context(), ep2); err != nil {
		t.Fatalf("second UpsertEndpoint: %v", err)
	}

	if _, err := metaCache.Get(t.Context(), created.ID); err == nil {
		t.Error("expected meta cache cleared after Upsert, but key still exists")
	}
}

func TestIntegration_DeleteByServerID_NotFound(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	endpointRepo := NewEndpointRepositoryWithDeps(
		testDB,
		scheduler.NewZSetScheduleRepository(testRedis),
		monitorrepo.NewRedisServerEventRepository(testRedis),
		scheduler.NewEndpointMetaCache(testRedis),
	)

	err := endpointRepo.DeleteByServerID(t.Context(), 999)
	if err != nil {
		t.Fatalf("expected nil for non-existent server, got: %v", err)
	}
}
