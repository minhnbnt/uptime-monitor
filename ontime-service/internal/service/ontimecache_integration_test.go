package service

import (
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/testcontainers"
)

func TestIntegration_OntimeCache_SentinelAndFloat(t *testing.T) {
	testcontainers.SkipIfShort(t)

	client := testcontainers.NewTestRedis(t, testRedisAddr)
	repo := ontimerepo.NewOntimeCacheRepository(client)

	day := time.Date(2026, 6, 4, 0, 0, 0, 0, time.UTC)
	noDataKey := dto.BatchGetOntimeItem{EndpointID: 1, Date: day}
	dataKey := dto.BatchGetOntimeItem{EndpointID: 2, Date: day}

	if err := repo.MSet(t.Context(), map[dto.BatchGetOntimeItem]dto.DayResult{
		noDataKey: {HasData: false},
		dataKey:   {HasData: true, Uptime: 99.5},
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
	if noData.Uptime != 0 {
		t.Errorf("noData key: Uptime = %v, want 0", noData.Uptime)
	}

	withData := got[dataKey]
	if !withData.HasData {
		t.Fatal("data key: HasData = false, want true")
	}
	if withData.Uptime != 99.5 {
		t.Errorf("data key: Uptime = %v, want 99.5", withData.Uptime)
	}
}
