package repository

import (
	"context"
	"flag"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var testRedis *redis.Client

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()
		container, client := testcontainers.StartRedis(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testRedis = client
	}
	os.Exit(m.Run())
}

func newCacheRepository(tb testing.TB) *OntimeCacheRepository {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	return &OntimeCacheRepository{client: testRedis}
}

func TestIntegration_RedisKey(t *testing.T) {
	tests := []struct {
		serverID uint
		day      time.Time
		want     string
	}{
		{1, time.Date(2026, 6, 25, 0, 0, 0, 0, time.UTC), "ontime:1:2026-06-25:stats"},
		{42, time.Date(2026, 1, 1, 12, 30, 0, 0, time.UTC), "ontime:42:2026-01-01:stats"},
		{0, time.Date(2026, 12, 31, 23, 59, 59, 0, time.UTC), "ontime:0:2026-12-31:stats"},
	}

	for _, tt := range tests {
		got := redisKey(tt.serverID, tt.day)
		if got != tt.want {
			t.Errorf("redisKey(%d, %v) = %q, want %q", tt.serverID, tt.day, got, tt.want)
		}
	}
}

func TestIntegration_MGet_Empty(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)
	repo := newCacheRepository(t)

	result, err := repo.MGet(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("result = %v, want nil", result)
	}
}

func TestIntegration_MGet_AllHit(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	yesterday := time.Now().AddDate(0, 0, -1)
	key1 := redisKey(1, yesterday)
	key2 := redisKey(2, yesterday)

	if err := testRedis.Set(t.Context(), key1, "99.50", 0).Err(); err != nil {
		t.Fatalf("seed key1: %v", err)
	}
	if err := testRedis.Set(t.Context(), key2, "50.00", 0).Err(); err != nil {
		t.Fatalf("seed key2: %v", err)
	}

	repo := newCacheRepository(t)
	items := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: yesterday},
		{ServerID: 2, Date: yesterday},
	}

	result, err := repo.MGet(t.Context(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("len(result) = %d, want 2", len(result))
	}
	if result[items[0]] != 99.50 {
		t.Errorf("result[1] = %f, want 99.50", result[items[0]])
	}
	if result[items[1]] != 50.00 {
		t.Errorf("result[2] = %f, want 50.00", result[items[1]])
	}
}

func TestIntegration_MGet_PartialHit(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	yesterday := time.Now().AddDate(0, 0, -1)
	key1 := redisKey(1, yesterday)

	if err := testRedis.Set(t.Context(), key1, "80.00", 0).Err(); err != nil {
		t.Fatalf("seed key1: %v", err)
	}

	repo := newCacheRepository(t)
	items := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: yesterday},
		{ServerID: 2, Date: yesterday},
	}

	result, err := repo.MGet(t.Context(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}
	if result[items[0]] != 80.00 {
		t.Errorf("result[1] = %f, want 80.00", result[items[0]])
	}
	_, hit := result[items[1]]
	if hit {
		t.Error("item 2 should not be in result")
	}
}

func TestIntegration_MGet_AllMiss(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	repo := newCacheRepository(t)
	items := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: time.Now().AddDate(0, 0, -1)},
		{ServerID: 2, Date: time.Now().AddDate(0, 0, -2)},
	}

	result, err := repo.MGet(t.Context(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0", len(result))
	}
}

func TestIntegration_MGet_TypeError(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	yesterday := time.Now().AddDate(0, 0, -1)
	key := redisKey(1, yesterday)

	if err := testRedis.HSet(t.Context(), key, "field", "value").Err(); err != nil {
		t.Fatalf("seed hash key: %v", err)
	}

	repo := newCacheRepository(t)
	items := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: yesterday},
	}

	result, err := repo.MGet(t.Context(), items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("len(result) = %d, want 0 (hash value should be skipped)", len(result))
	}
}

func TestIntegration_MSet_Empty(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	repo := newCacheRepository(t)
	err := repo.MSet(t.Context(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIntegration_MSet_PastDate(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	repo := newCacheRepository(t)
	yesterday := time.Now().AddDate(0, 0, -1)
	items := map[dto.BatchGetOntimeItem]float64{
		{ServerID: 1, Date: yesterday}: 85.75,
	}

	if err := repo.MSet(t.Context(), items); err != nil {
		t.Fatalf("MSet error: %v", err)
	}

	key := redisKey(1, yesterday)
	val, err := testRedis.Get(t.Context(), key).Result()
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if val != "85.75" {
		t.Errorf("got %q, want 85.75", val)
	}

	ttl, err := testRedis.TTL(t.Context(), key).Result()
	if err != nil {
		t.Fatalf("TTL error: %v", err)
	}
	if ttl < 50*time.Minute || ttl > 1*time.Hour {
		t.Errorf("TTL = %v, want ~1h", ttl)
	}
}

func TestIntegration_MSet_MultipleItems(t *testing.T) {
	testcontainers.CleanRedis(t, testRedis)

	repo := newCacheRepository(t)
	yesterday := time.Now().AddDate(0, 0, -1)
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	items := map[dto.BatchGetOntimeItem]float64{
		{ServerID: 1, Date: yesterday}:  90.00,
		{ServerID: 2, Date: twoDaysAgo}: 45.50,
	}

	if err := repo.MSet(t.Context(), items); err != nil {
		t.Fatalf("MSet error: %v", err)
	}

	key1 := redisKey(1, yesterday)
	key2 := redisKey(2, twoDaysAgo)

	val1, err := testRedis.Get(t.Context(), key1).Result()
	if err != nil {
		t.Fatalf("Get key1 error: %v", err)
	}
	if val1 != "90.00" {
		t.Errorf("val1 = %q, want 90.00", val1)
	}

	val2, err := testRedis.Get(t.Context(), key2).Result()
	if err != nil {
		t.Fatalf("Get key2 error: %v", err)
	}
	if val2 != "45.50" {
		t.Errorf("val2 = %q, want 45.50", val2)
	}
}
