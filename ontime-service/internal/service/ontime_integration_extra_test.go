package service

import (
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/testcontainers"
)

func TestIntegration_BatchGetOntime_LowerboundON_NoDayEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	queryDate := oDay(2026, 6, 2)
	seedEndpoint(t, db, 1, 1)
	seedEvent(t, db, 1, domain.StatusOn, oTm(2026, 6, 1, 18, 0))

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: queryDate}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	got := results[0].Result[0].Stats
	if got != 100 {
		t.Errorf("Stats = %f, want 100 (lowerbound ON, no events)", got)
	}
}

func TestIntegration_BatchGetOntime_LowerboundOFF_NoDayEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	queryDate := oDay(2026, 6, 2)
	seedEndpoint(t, db, 1, 1)
	seedEvent(t, db, 1, domain.StatusOff, oTm(2026, 6, 1, 23, 0))

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: queryDate}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	got := results[0].Result[0].Stats
	if got != 0 {
		t.Errorf("Stats = %f, want 0 (lowerbound OFF, no events)", got)
	}
}

func TestIntegration_BatchGetOntime_TodaySingleON_PrevDayOFF(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	today := oDay(time.Now().Year(), int(time.Now().Month()), time.Now().Day())
	yesterday := today.Add(-24 * time.Hour)
	onTime := today.Add(6 * time.Hour)
	if time.Now().Before(onTime) {
		t.Skip("event at 06:00 UTC hasn't happened yet — skip")
	}

	seedEndpoint(t, db, 1, 1)
	seedEvent(t, db, 1, domain.StatusOff, yesterday.Add(23*time.Hour))
	seedEvent(t, db, 1, domain.StatusOn, onTime)

	now := time.Now()
	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: today}}

	results, err := b.BatchGetOntimeUntil(t.Context(), req, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}

	startTime := yesterday.Add(23 * time.Hour)
	online := now.Sub(onTime).Seconds()
	coverage := now.Sub(startTime).Seconds()
	want := online / coverage * 100

	got := results[0].Result[0].Stats
	if got != want {
		t.Errorf("Stats = %f, want %f (prev day OFF, today ON at 06:00)", got, want)
	}
}

func TestIntegration_BatchGetOntime_Today_ON_to_OFF(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	today := oDay(time.Now().Year(), int(time.Now().Month()), time.Now().Day())
	offTime := today.Add(10 * time.Hour)
	if time.Now().Before(offTime) {
		t.Skip("OFF at 10:00 UTC hasn't happened yet — skip")
	}

	seedEndpoint(t, db, 1, 1)
	seedEvent(t, db, 1, domain.StatusOn, today.Add(6*time.Hour))
	seedEvent(t, db, 1, domain.StatusOff, offTime)

	now := time.Now()
	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: today}}

	results, err := b.BatchGetOntimeUntil(t.Context(), req, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}

	onTime := today.Add(6 * time.Hour)
	online := offTime.Sub(onTime).Seconds()
	coverage := now.Sub(onTime).Seconds()
	want := online / coverage * 100

	got := results[0].Result[0].Stats
	if got != want {
		t.Errorf("Stats = %f, want %f (ON 06-10, then OFF)", got, want)
	}
}

func TestIntegration_BatchGetOntime_NoEndpoint(t *testing.T) {
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
	if results[0].Result[0].Stats != 0 {
		t.Errorf("Stats = %f, want 0 (no endpoint)", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_SoftDeletedEndpoint(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	now := oDay(2026, 6, 1)
	// Endpoint with deleted_at set (soft deleted)
	db.Exec(`INSERT INTO endpoints (id, server_id, deleted_at) VALUES (1, 1, ?)`, now.Add(-24*time.Hour))

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Result[0].Stats != 0 {
		t.Errorf("Stats = %f, want 0 (soft-deleted endpoint)", results[0].Result[0].Stats)
	}
}

func TestIntegration_BatchGetOntime_MultipleServers(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	now := oDay(2026, 6, 1)
	seedEndpoint(t, db, 1, 1)
	seedEndpoint(t, db, 2, 2)
	seedEndpoint(t, db, 3, 3)

	seedEvent(t, db, 1, domain.StatusOn, oTm(2026, 6, 1, 0, 0))
	seedEvent(t, db, 2, domain.StatusOff, oTm(2026, 6, 1, 0, 0))
	seedEvent(t, db, 3, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedEvent(t, db, 3, domain.StatusOff, oTm(2026, 6, 1, 12, 0))
	seedEvent(t, db, 3, domain.StatusOn, oTm(2026, 6, 1, 18, 0))

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: now},
		{ServerID: 2, Date: now},
		{ServerID: 3, Date: now},
	}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}

	stats := map[uint]float64{}
	for _, r := range results {
		if len(r.Result) == 1 {
			stats[r.ServerID] = r.Result[0].Stats
		}
	}
	if stats[1] != 100 {
		t.Errorf("s1 Stats = %f, want 100", stats[1])
	}
	if stats[2] != 0 {
		t.Errorf("s2 Stats = %f, want 0", stats[2])
	}
	if stats[3] <= 0 {
		t.Errorf("s3 Stats = %f, want > 0", stats[3])
	}
}

func TestIntegration_BatchGetOntime_MultipleEndpointsOneActive(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	now := oDay(2026, 6, 1)

	// Active endpoint
	db.Exec(`INSERT INTO endpoints (id, server_id) VALUES (1, 1)`)
	// Soft-deleted endpoint
	db.Exec(`INSERT INTO endpoints (id, server_id, deleted_at) VALUES (2, 1, ?)`, now.Add(-24*time.Hour))

	seedEvent(t, db, 1, domain.StatusOn, oTm(2026, 6, 1, 0, 0))

	b := newBatcher(t, db)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := b.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Result[0].Stats != 100 {
		t.Errorf("Stats = %f, want 100", results[0].Result[0].Stats)
	}
}
