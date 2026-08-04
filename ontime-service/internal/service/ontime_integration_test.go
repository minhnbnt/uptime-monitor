package service

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/testcontainers"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/logger"
)

var testRedisAddr string
var testDSN string

func TestMain(m *testing.M) {
	flag.Parse()

	if !testing.Short() {
		ctx := context.Background()

		redisContainer, addr := testcontainers.StartRedisAddr(ctx)
		defer func() { _ = redisContainer.Terminate(ctx) }()
		testRedisAddr = addr

		container, dsn := testcontainers.StartPostgres(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testDSN = dsn
	}

	os.Exit(m.Run())
}

func initTestDB(tb testing.TB) *gorm.DB {
	tb.Helper()
	db := testcontainers.CreateTestDB(tb, testDSN)
	return db
}

func newBatcher(tb testing.TB, db *gorm.DB) *Batcher {
	tb.Helper()
	testcontainers.SkipIfShort(tb)

	return NewBatcher(
		ontimerepo.NewOntineRepository(db),
		nil,
		logger.NewMockLogger(),
	)
}

func newBatcherWithRedis(tb testing.TB, db *gorm.DB, redisClient *redis.Client) *Batcher {
	tb.Helper()
	testcontainers.SkipIfShort(tb)

	return NewBatcher(
		ontimerepo.NewOntineRepository(db),
		ontimerepo.NewOntimeCacheRepository(redisClient),
		logger.NewMockLogger(),
	)
}

func seedEvent(tb testing.TB, db *gorm.DB, endpointID uint, status domain.ServerStatus, tm time.Time) {
	tb.Helper()
	db.Create(&domain.ServerEvent{
		ID:       uuid.New(),
		ServerID: endpointID,
		Status:   status,
		Time:     tm,
	})
}

// ---------- BatchGetOntime ----------

func TestIntegration_BatchGetOntime_CacheMiss(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	now := oDay(2026, 6, 1)
	seedEvent(t, db, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedEvent(t, db, 1, domain.StatusOff, oTm(2026, 6, 1, 18, 0))

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if len(results[0].DayStats) != 1 {
		t.Fatalf("len(Result) = %d, want 1", len(results[0].DayStats))
	}
	if results[0].DayStats[0].Result.Uptime <= 0 {
		t.Errorf("Stats = %f, want > 0", results[0].DayStats[0].Result.Uptime)
	}
	if !results[0].DayStats[0].Result.HasData {
		t.Error("HasData = false, want true (events exist for this day)")
	}
}

func TestIntegration_BatchGetOntime_NoData(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	// No events for server 1 at all → day carries no data, not 0%.
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: oDay(2026, 6, 1)}}

	b := newBatcher(t, db)
	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].DayStats) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	if results[0].DayStats[0].Result.HasData {
		t.Error("HasData = true, want false (no data for this server)")
	}
}

func TestIntegration_BatchGetOntime_AllOn(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	now := oDay(2026, 6, 1)
	seedEvent(t, db, 1, domain.StatusOn, oTm(2026, 6, 1, 0, 0))

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].DayStats[0].Result.Uptime != 100 {
		t.Errorf("Stats = %f, want 100", results[0].DayStats[0].Result.Uptime)
	}
}

func TestIntegration_BatchGetOntime_AllOff(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	now := oDay(2026, 6, 1)
	seedEvent(t, db, 1, domain.StatusOff, oTm(2026, 6, 1, 0, 0))

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].DayStats[0].Result.Uptime != 0 {
		t.Errorf("Stats = %f, want 0", results[0].DayStats[0].Result.Uptime)
	}
}

func TestIntegration_BatchGetOntime_NoEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	now := oDay(2026, 6, 1)
	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].DayStats[0].Result.Uptime != 0 {
		t.Errorf("Stats = %f, want 0 (no events)", results[0].DayStats[0].Result.Uptime)
	}
}

func TestIntegration_BatchGetOntime_EmptyRequest(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	b := newBatcher(t, db)
	results, err := b.BatchGetOntime(t.Context(), nil)
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
	db := initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	now := oDay(2026, 6, 1)
	b := newBatcherWithRedis(t, db, redisClient)

	key := fmt.Sprintf("ontime:%d:%s:stats", 1, now.Format("2006-01-02"))
	if err := redisClient.Set(t.Context(), key, "99.50", 0).Err(); err != nil {
		t.Fatalf("seed redis: %v", err)
	}

	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}
	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].DayStats) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	if results[0].DayStats[0].Result.Uptime != 99.50 {
		t.Errorf("Stats = %f, want 99.50", results[0].DayStats[0].Result.Uptime)
	}
}

func TestIntegration_BatchGetOntime_CacheMissThenWarm(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	now := oDay(2026, 6, 1)
	seedEvent(t, db, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))

	b := newBatcherWithRedis(t, db, redisClient)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].DayStats) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	if results[0].DayStats[0].Result.Uptime <= 0 {
		t.Errorf("Stats = %f, want > 0", results[0].DayStats[0].Result.Uptime)
	}

	key := fmt.Sprintf("ontime:%d:%s:stats", 1, now.Format("2006-01-02"))
	val, err := redisClient.Get(t.Context(), key).Result()
	if err != nil {
		t.Fatalf("Get cached key: %v", err)
	}
	if val == "" {
		t.Error("expected non-empty cached value")
	}
}

func TestIntegration_BatchGetOntime_PartialCacheHit(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)
	redisClient := testcontainers.NewTestRedis(t, testRedisAddr)

	now := oDay(2026, 6, 1)
	seedEvent(t, db, 1, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedEvent(t, db, 2, domain.StatusOn, oTm(2026, 6, 1, 0, 0))

	key := fmt.Sprintf("ontime:%d:%s:stats", 1, now.Format("2006-01-02"))
	if err := redisClient.Set(t.Context(), key, "88.00", 0).Err(); err != nil {
		t.Fatalf("seed redis: %v", err)
	}

	b := newBatcherWithRedis(t, db, redisClient)
	req := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: now},
		{ServerID: 2, Date: now},
	}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d, want 2", len(results))
	}

	for _, r := range results {
		if len(r.DayStats) != 1 {
			t.Fatalf("server %d: got %d results, want 1", r.ServerID, len(r.DayStats))
		}
	}

	for _, r := range results {
		if r.ServerID == 1 && r.DayStats[0].Result.Uptime != 88.00 {
			t.Errorf("server 1: Stats = %f, want 88.00", r.DayStats[0].Result.Uptime)
		}
		if r.ServerID == 2 && r.DayStats[0].Result.Uptime <= 0 {
			t.Errorf("server 2: Stats = %f, want > 0", r.DayStats[0].Result.Uptime)
		}
	}

	key2 := fmt.Sprintf("ontime:%d:%s:stats", 2, now.Format("2006-01-02"))
	val, err := redisClient.Get(t.Context(), key2).Result()
	if err != nil {
		t.Fatalf("Get cached key for server 2: %v", err)
	}
	if val == "" {
		t.Error("expected server 2 to be cached after miss")
	}
}

// ---------- today vs past day ----------

func TestIntegration_BatchGetOntime_Today(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	today := oDay(time.Now().Year(), int(time.Now().Month()), time.Now().Day())
	onTime := today.Add(6 * time.Hour)
	if time.Now().Before(onTime) {
		t.Skip("event at 06:00 UTC hasn't happened yet — skip")
	}

	seedEvent(t, db, 1, domain.StatusOn, onTime)

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: today}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].DayStats) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}

	got := results[0].DayStats[0].Result.Uptime
	if got <= 0 {
		t.Errorf("Stats = %f, want > 0 for today", got)
	}
}
