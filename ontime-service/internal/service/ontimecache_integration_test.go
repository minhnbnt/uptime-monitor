package service

import (
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/testcontainers"
)

func TestIntegration_OntimeCache_HashRoundTrip(t *testing.T) {
	testcontainers.SkipIfShort(t)

	client := testcontainers.NewTestRedis(t, testRedisAddr)
	repo := ontimerepo.NewOntimeCacheRepository(client)

	day := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)
	noDataKey := dto.BatchGetOntimeItem{EndpointID: 1, Date: day}
	dataKey := dto.BatchGetOntimeItem{EndpointID: 2, Date: day}

	if err := repo.MSet(t.Context(), map[dto.BatchGetOntimeItem]dto.DayResult{
		noDataKey: {HasData: false, Unknown: 12 * 3600},
		dataKey:   {HasData: true, Uptime: 99.123456789, Unknown: 600},
	}); err != nil {
		t.Fatalf("MSet: %v", err)
	}

	got, err := repo.MGet(t.Context(), []dto.BatchGetOntimeItem{noDataKey, dataKey})
	if err != nil {
		t.Fatalf("MGet: %v", err)
	}

	noData := got[noDataKey]
	if noData.HasData {
		t.Errorf("noData key: HasData = true, want false")
	}
	if noData.Unknown != 12*3600 {
		t.Errorf("noData key: Unknown = %v, want %v", noData.Unknown, 12*3600)
	}

	withData := got[dataKey]
	if !withData.HasData {
		t.Fatal("data key: HasData = false, want true")
	}
	if withData.Uptime != 99.123456789 {
		t.Errorf("data key: Uptime = %v, want full precision 99.123456789", withData.Uptime)
	}
	if withData.Unknown != 600 {
		t.Errorf("data key: Unknown = %v, want 600", withData.Unknown)
	}
}

// Legacy v1 entries were plain strings under the ":stats" suffix. The v2
// suffix must make them invisible reads (misses) while they expire on their
// own, never WRONGTYPE failures.
func TestIntegration_OntimeCache_LegacyStringEntryIsMiss(t *testing.T) {
	testcontainers.SkipIfShort(t)

	client := testcontainers.NewTestRedis(t, testRedisAddr)
	repo := ontimerepo.NewOntimeCacheRepository(client)

	day := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)
	key := dto.BatchGetOntimeItem{EndpointID: 7, Date: day}

	legacyKey := "ontime:7:2026-06-04:stats"
	if err := client.Set(t.Context(), legacyKey, "__NULL__", time.Hour).Err(); err != nil {
		t.Fatalf("seed legacy entry: %v", err)
	}

	got, err := repo.MGet(t.Context(), []dto.BatchGetOntimeItem{key})
	if err != nil {
		t.Fatalf("MGet must not fail on legacy entries: %v", err)
	}
	if _, hit := got[key]; hit {
		t.Error("legacy string entry must be a cache miss")
	}

	if err := repo.MSet(t.Context(), map[dto.BatchGetOntimeItem]dto.DayResult{
		key: {HasData: true, Uptime: 50, Unknown: 60},
	}); err != nil {
		t.Fatalf("MSet after legacy entry: %v", err)
	}

	got, err = repo.MGet(t.Context(), []dto.BatchGetOntimeItem{key})
	if err != nil {
		t.Fatalf("MGet after MSet: %v", err)
	}
	if !got[key].HasData || got[key].Uptime != 50 {
		t.Errorf("v2 entry = %+v, want HasData=true Uptime=50", got[key])
	}
}
