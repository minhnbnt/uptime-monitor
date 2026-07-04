package ontime

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/ontime/repository"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

// ---------- container lifecycle ----------

var testDB *gorm.DB
var testRedis *redis.Client
var testDSN string

func TestMain(m *testing.M) {

	flag.Parse()

	if !testing.Short() {
		ctx := context.Background()

		redisContainer, redisClient := testcontainers.StartRedis(ctx)
		defer func() { _ = redisContainer.Terminate(ctx) }()
		testRedis = redisClient

		container, dsn := testcontainers.StartPostgres(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testDSN = dsn
	}

	os.Exit(m.Run())
}

func initTestDB(tb testing.TB) *gorm.DB {
	tb.Helper()
	return testcontainers.CreateTestDB(tb, testDSN, func(db *gorm.DB) {
		if err := db.Create(&domain.User{
			Model:    gorm.Model{ID: 1},
			Email:    "test@test.com",
			Username: "test",
			Password: "x",
			Name:     "Test",
		}).Error; err != nil {
			tb.Fatalf("seed test user: %v", err)
		}
	})
}

func newServiceWithRedis(tb testing.TB) *OntimeService {
	tb.Helper()
	testcontainers.SkipIfShort(tb)

	return &OntimeService{
		serverRepository: serverrepo.NewServerRepository(testDB),
		batcher: NewBatcher(
			ontimerepo.NewOntineRepository(testDB),
			ontimerepo.NewOntimeCacheRepository(testRedis),
			logger.NewMockLogger(),
		),
	}
}

func newService(tb testing.TB) *OntimeService {
	tb.Helper()
	testcontainers.SkipIfShort(tb)

	return &OntimeService{
		serverRepository: serverrepo.NewServerRepository(testDB),
		batcher: &Batcher{
			ontineRepository: ontimerepo.NewOntineRepository(testDB),
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, _ []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return make(map[dto.BatchGetOntimeItem]float64), nil
				},
				mSetFn: func(_ context.Context, _ map[dto.BatchGetOntimeItem]float64) error {
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		},
	}
}

func seedServer(tb testing.TB, id uint, name string, createdAt time.Time) {
	tb.Helper()
	testDB.Create(&domain.Server{
		Model:       gorm.Model{ID: id, CreatedAt: createdAt, UpdatedAt: createdAt},
		Name:        name,
		CreatedByID: 1,
	})
}

func seedEndpoint(tb testing.TB, id, serverID uint) {
	tb.Helper()
	testDB.Create(&domain.Endpoint{
		Model:    gorm.Model{ID: id},
		ServerID: serverID,
		URL:      "https://example.com",
		Method:   "GET",
	})
}

func seedEvent(tb testing.TB, endpointID uint, status domain.ServerStatus, tm time.Time) {
	tb.Helper()
	testDB.Create(&domain.ServerEvent{
		ID:         uuid.New(),
		EndpointID: endpointID,
		Status:     status,
		Time:       tm,
	})
}

// ---------- BatchGetOntime ----------

func TestIntegration_BatchGetOntime_CacheMiss(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedEvent(t, 1, domain.StatusOff, oTm(2026, 6, 1, 18, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].Result) != 1 {
		t.Fatalf("len(Result) = %d, want 1", len(results[0].Result))
	}
	if results[0].Result[0].Stats <= 0 {
		t.Errorf("Stats = %f, want > 0", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_AllOn(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 0, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Result[0].Stats != 100 {
		t.Errorf("Stats = %f, want 100", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_AllOff(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOff, oTm(2026, 6, 1, 0, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Result[0].Stats != 0 {
		t.Errorf("Stats = %f, want 0", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_NoEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Result[0].Stats != 0 {
		t.Errorf("Stats = %f, want 0 (no events)", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_EmptyRequest(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	svc := newService(t)
	results, err := svc.batcher.BatchGetOntime(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("len(results) = %d, want 0", len(results))
	}
}

// ---------- BatchGetOntime with real Redis cache ----------

func TestIntegration_BatchGetOntime_CacheHit(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	testcontainers.CleanRedis(t, testRedis)

	now := oDay(2026, 6, 1)
	svc := newServiceWithRedis(t)

	key := fmt.Sprintf("ontime:%d:%s:stats", 1, now.Format("2006-01-02"))
	if err := testRedis.Set(t.Context(), key, "99.50", 0).Err(); err != nil {
		t.Fatalf("seed redis: %v", err)
	}

	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}
	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	if results[0].Result[0].Stats != 99.50 {
		t.Errorf("Stats = %f, want 99.50", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_CacheMissThenWarm(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	testcontainers.CleanRedis(t, testRedis)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))

	svc := newServiceWithRedis(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	if results[0].Result[0].Stats <= 0 {
		t.Errorf("Stats = %f, want > 0", results[0].Result[0].Stats)
	}

	key := fmt.Sprintf("ontime:%d:%s:stats", 1, now.Format("2006-01-02"))
	val, err := testRedis.Get(t.Context(), key).Result()
	if err != nil {
		t.Fatalf("Get cached key: %v", err)
	}
	if val == "" {
		t.Error("expected non-empty cached value")
	}
}

func TestIntegration_BatchGetOntime_PartialCacheHit(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)
	testcontainers.CleanRedis(t, testRedis)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedServer(t, 2, "s2", now.Add(-48*time.Hour))
	seedEndpoint(t, 2, 2)
	seedEvent(t, 2, domain.StatusOn, oTm(2026, 6, 1, 0, 0))

	key := fmt.Sprintf("ontime:%d:%s:stats", 1, now.Format("2006-01-02"))
	if err := testRedis.Set(t.Context(), key, "88.00", 0).Err(); err != nil {
		t.Fatalf("seed redis: %v", err)
	}

	svc := newServiceWithRedis(t)
	req := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: now},
		{ServerID: 2, Date: now},
	}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	for _, r := range results {
		if len(r.Result) != 1 {
			t.Fatalf("server %d: got %d results, want 1", r.ServerID, len(r.Result))
		}
	}

	for _, r := range results {
		if r.ServerID == 1 && r.Result[0].Stats != 88.00 {
			t.Errorf("server 1: Stats = %f, want 88.00", r.Result[0].Stats)
		}
		if r.ServerID == 2 && r.Result[0].Stats <= 0 {
			t.Errorf("server 2: Stats = %f, want > 0", r.Result[0].Stats)
		}
	}

	key2 := fmt.Sprintf("ontime:%d:%s:stats", 2, now.Format("2006-01-02"))
	val, err := testRedis.Get(t.Context(), key2).Result()
	if err != nil {
		t.Fatalf("Get cached key for server 2: %v", err)
	}
	if val == "" {
		t.Error("expected server 2 to be cached after miss")
	}
}

// ---------- ListServersWithOntime ----------

func TestIntegration_ListServersWithOntime(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	createdAt := oDay(2026, 6, 1).Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedServer(t, 2, "s2", createdAt)
	seedEndpoint(t, 1, 1)
	seedEndpoint(t, 2, 2)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedEvent(t, 2, domain.StatusOff, oTm(2026, 6, 2, 0, 0))

	svc := newService(t)
	results, total, _, _, err := svc.ListServersWithOntime(t.Context(), 1, 1, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	// Each server should have ontime stats for its date range
	for _, r := range results {
		if len(r.OntimeStats) == 0 {
			t.Errorf("server %s: ontime_stats is empty", r.Server.Name)
		}
		for _, stat := range r.OntimeStats {
			if stat.Stats < 0 || stat.Stats > 100 {
				t.Errorf("server %s, date %v: Stats = %f, out of range [0,100]",
					r.Server.Name, stat.Date, stat.Stats)
			}
		}
	}
}

// ---------- GetServerWithOntime ----------

func TestIntegration_GetServerWithOntime(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	createdAt := oDay(2026, 6, 1).Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))

	svc := newService(t)
	result, err := svc.GetServerWithOntime(t.Context(), 1, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Server.Name != "s1" {
		t.Errorf("Server.Name = %q, want s1", result.Server.Name)
	}
	if len(result.OntimeStats) == 0 {
		t.Error("ontime_stats is empty")
	}
}

func TestIntegration_GetServerWithOntime_NotFound(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	svc := newService(t)
	_, err := svc.GetServerWithOntime(t.Context(), 999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent server")
	}
}

func TestIntegration_GetServerWithOntime_Forbidden(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	createdAt := oDay(2026, 6, 1).Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)

	svc := newService(t)
	_, err := svc.GetServerWithOntime(t.Context(), 1, 99)
	if err == nil {
		t.Fatal("expected error for non-matching user")
	}
	if !errors.Is(err, apperrors.ErrForbidden) {
		t.Errorf("got %v, want ErrForbidden", err)
	}
}

// ---------- URL encoding regression ----------

func TestIntegration_BatchGetOntime_URLSpecialChars(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	now := oDay(2026, 6, 1)
	seedServer(t, 1, "s1", now.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)

	// Server with URL containing special characters
	seedServer(t, 2, "s2", now.Add(-48*time.Hour))
	testDB.Create(&domain.Endpoint{
		Model:    gorm.Model{ID: 2},
		ServerID: 2,
		URL:      "https://example.com/path?q=1&r=2",
		Method:   "GET",
	})

	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: now},
		{ServerID: 2, Date: now},
	}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}
}

// ---------- today vs past day ----------

func TestIntegration_BatchGetOntime_Today(t *testing.T) {
	testcontainers.SkipIfShort(t)
	testDB = initTestDB(t)

	today := oDay(time.Now().Year(), int(time.Now().Month()), time.Now().Day())
	onTime := today.Add(6 * time.Hour)
	if time.Now().Before(onTime) {
		t.Skip("event at 06:00 UTC hasn't happened yet — skip")
	}

	seedServer(t, 1, "s1", today.Add(-48*time.Hour))
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, onTime)

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: today}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}

	// Today should have partial coverage (from 06:00 to now)
	got := results[0].Result[0].Stats
	if got <= 0 {
		t.Errorf("Stats = %f, want > 0 for today", got)
	}
}
