package ontime

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

// ---------- no events on the queried day, lowerbound from previous day ----------

func TestIntegration_BatchGetOntime_LowerboundON_NoDayEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)

	queryDate := oDay(2026, 6, 2)
	createdAt := queryDate.Add(-72 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedEndpoint(t, 1, 1)
	// Last event before queryDate is ON on previous day
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 18, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: queryDate}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	got := results[0].Result[0].Stats
	// Lowerbound ON + no events inside day → should be 100 (ON all day)
	if got != 100 {
		t.Errorf("Stats = %f, want 100 (lowerbound ON, no events)", got)
	}
}

func TestIntegration_BatchGetOntime_LowerboundOFF_NoDayEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)

	queryDate := oDay(2026, 6, 2)
	createdAt := queryDate.Add(-72 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedEndpoint(t, 1, 1)
	// Last event before queryDate is OFF on previous day
	seedEvent(t, 1, domain.StatusOff, oTm(2026, 6, 1, 23, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: queryDate}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	got := results[0].Result[0].Stats
	// Lowerbound OFF + no events inside day → should be 0 (OFF all day)
	if got != 0 {
		t.Errorf("Stats = %f, want 0 (lowerbound OFF, no events)", got)
	}
}

// ---------- today with single event ----------

func TestIntegration_BatchGetOntime_TodaySingleON_PrevDayOFF(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)

	today := oDay(time.Now().Year(), int(time.Now().Month()), time.Now().Day())
	yesterday := today.Add(-24 * time.Hour)
	onTime := today.Add(6 * time.Hour) // ON at 06:00 UTC
	if time.Now().Before(onTime) {
		t.Skip("event at 06:00 UTC hasn't happened yet — skip")
	}

	createdAt := today.Add(-72 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedEndpoint(t, 1, 1)
	// Yesterday OFF at 23:00 → lowerbound = OFF
	seedEvent(t, 1, domain.StatusOff, yesterday.Add(23*time.Hour))
	// Today ON at 06:00 → start at 06:00
	seedEvent(t, 1, domain.StatusOn, onTime)

	now := time.Now()

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: today}}

	results, err := svc.batcher.BatchGetOntimeUntil(t.Context(), req, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}

	// calculator: StartTime = last prev event = yesterday 23:00
	//            EndTime = now, StartStatus = OFF
	//            online = now - 06:00, coverage = now - yesterday 23:00
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
	truncateTables(t)

	today := oDay(time.Now().Year(), int(time.Now().Month()), time.Now().Day())
	offTime := today.Add(10 * time.Hour) // OFF at 10:00 UTC
	if time.Now().Before(offTime) {
		t.Skip("OFF at 10:00 UTC hasn't happened yet — skip")
	}

	createdAt := today.Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedEndpoint(t, 1, 1)
	// ON at 06:00, OFF at 10:00
	seedEvent(t, 1, domain.StatusOn, today.Add(6*time.Hour))
	seedEvent(t, 1, domain.StatusOff, offTime)

	now := time.Now()

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: today}}

	results, err := svc.batcher.BatchGetOntimeUntil(t.Context(), req, now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}

	// no prev events, StartTime = first day event = 06:00 (isToday=true)
	// StartStatus = ON, EndTime = now
	// online = 06:00→10:00 = 4h, then OFF
	onTime := today.Add(6 * time.Hour)
	online := offTime.Sub(onTime).Seconds()
	coverage := now.Sub(onTime).Seconds()
	want := online / coverage * 100

	got := results[0].Result[0].Stats
	if got != want {
		t.Errorf("Stats = %f, want %f (ON 06-10, then OFF)", got, want)
	}
}

// ---------- server with no endpoint ----------

func TestIntegration_BatchGetOntime_NoEndpoint(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)

	now := oDay(2026, 6, 1)
	createdAt := now.Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	// No endpoint for this server

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	// No events found → Stats = 0
	if results[0].Result[0].Stats != 0 {
		t.Errorf("Stats = %f, want 0 (no endpoint)", results[0].Result[0].Stats)
	}
}

// ---------- endpoint soft deleted ----------

func TestIntegration_BatchGetOntime_SoftDeletedEndpoint(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)

	now := oDay(2026, 6, 1)
	createdAt := now.Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	// Endpoint with deleted_at set (soft deleted)
	testDB.Create(&domain.Endpoint{
		Model: gorm.Model{
			ID:        1,
			DeletedAt: gorm.DeletedAt{Time: now.Add(-24 * time.Hour), Valid: true},
		},
		ServerID: 1,
		URL:      "https://example.com",
		Method:   "GET",
	})

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	// No active endpoint → no events → Stats = 0
	if results[0].Result[0].Stats != 0 {
		t.Errorf("Stats = %f, want 0 (soft-deleted endpoint)", results[0].Result[0].Stats)
	}
}

// ---------- server created today ----------

func TestIntegration_BatchGetOntime_ServerCreatedToday(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)

	today := oDay(time.Now().Year(), int(time.Now().Month()), time.Now().Day())
	eventTime := today.Add(14 * time.Hour) // ON at 14:00 UTC
	if time.Now().Before(eventTime) {
		t.Skip("event at 14:00 UTC hasn't happened yet — skip")
	}

	createdAt := today.Add(10 * time.Hour) // created at 10:00 today
	seedServer(t, 1, "s1", createdAt)
	seedEndpoint(t, 1, 1)
	seedEvent(t, 1, domain.StatusOn, eventTime)

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: today}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || len(results[0].Result) != 1 {
		t.Fatalf("unexpected result shape: %+v", results)
	}
	got := results[0].Result[0].Stats
	if got <= 0 {
		t.Errorf("Stats = %f, want > 0 (server created today)", got)
	}
}

// ---------- multiple servers with mix of data ----------

func TestIntegration_BatchGetOntime_MultipleServers(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)

	now := oDay(2026, 6, 1)
	createdAt := now.Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)
	seedServer(t, 2, "s2", createdAt)
	seedServer(t, 3, "s3", createdAt)
	seedEndpoint(t, 1, 1)
	seedEndpoint(t, 2, 2)
	seedEndpoint(t, 3, 3)

	// s1: ON all day → 100%
	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 0, 0))
	// s2: OFF all day → 0%
	seedEvent(t, 2, domain.StatusOff, oTm(2026, 6, 1, 0, 0))
	// s3: ON 6h, OFF 18h → 25%
	seedEvent(t, 3, domain.StatusOn, oTm(2026, 6, 1, 6, 0))
	seedEvent(t, 3, domain.StatusOff, oTm(2026, 6, 1, 12, 0))
	seedEvent(t, 3, domain.StatusOn, oTm(2026, 6, 1, 18, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: now},
		{ServerID: 2, Date: now},
		{ServerID: 3, Date: now},
	}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
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
		t.Errorf("s3 Stats = %f, want > 0 (ON 6h + ON 18h)", stats[3])
	}
}

// ---------- server with multiple endpoints (only active one counts) ----------

func TestIntegration_BatchGetOntime_MultipleEndpointsOneActive(t *testing.T) {
	testcontainers.SkipIfShort(t)
	truncateTables(t)

	now := oDay(2026, 6, 1)
	createdAt := now.Add(-48 * time.Hour)
	seedServer(t, 1, "s1", createdAt)

	// Active endpoint
	testDB.Create(&domain.Endpoint{
		Model:    gorm.Model{ID: 1},
		ServerID: 1,
		URL:      "https://active.example.com",
		Method:   "GET",
	})
	// Soft-deleted endpoint
	testDB.Create(&domain.Endpoint{
		Model: gorm.Model{
			ID:        2,
			DeletedAt: gorm.DeletedAt{Time: now.Add(-24 * time.Hour), Valid: true},
		},
		ServerID: 1,
		URL:      "https://deleted.example.com",
		Method:   "GET",
	})

	seedEvent(t, 1, domain.StatusOn, oTm(2026, 6, 1, 0, 0))

	svc := newService(t)
	req := []dto.BatchGetOntimeItem{{ServerID: 1, Date: now}}

	results, err := svc.batcher.BatchGetOntime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	// Should find the active endpoint and return stats
	if results[0].Result[0].Stats != 100 {
		t.Errorf("Stats = %f, want 100", results[0].Result[0].Stats)
	}
}
