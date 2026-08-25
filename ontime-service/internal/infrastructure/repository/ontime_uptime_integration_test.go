package repository

import (
	"context"
	"flag"
	"math"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/testcontainers"
)

var testDSN string

func TestMain(m *testing.M) {
	flag.Parse()

	if !testing.Short() {
		ctx := context.Background()
		container, dsn := testcontainers.StartPostgres(ctx)
		defer func() { _ = container.Terminate(ctx) }()
		testDSN = dsn
	}

	os.Exit(m.Run())
}

func uDay(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func uTm(y, m, d, h, mn int) time.Time {
	return time.Date(y, time.Month(m), d, h, mn, 0, 0, time.UTC)
}

func initUptimeTestDB(tb testing.TB) *gorm.DB {
	tb.Helper()
	return testcontainers.CreateTestDB(tb, testDSN)
}

func seedUptimeRow(tb testing.TB, db *gorm.DB, endpointID uint, status domain.ServerStatus, tm time.Time) {
	tb.Helper()

	if err := db.Create(&domain.ServerEvent{
		ID:         uuid.New(),
		EndpointID: endpointID,
		Status:     status,
		Time:       tm,
	}).Error; err != nil {
		tb.Fatalf("seed event: %v", err)
	}
}

// seedRawEvent bypasses the domain status type to insert values the DB
// schema allows but ToServerStatus rejects (unknown statuses).
func seedRawEvent(tb testing.TB, db *gorm.DB, endpointID uint, raw string, tm time.Time) {
	tb.Helper()

	if err := db.Exec(
		`INSERT INTO server_events (id, endpoint_id, status, "time") VALUES (?, ?, ?, ?)`,
		uuid.New(), endpointID, raw, tm,
	).Error; err != nil {
		tb.Fatalf("seed raw event: %v", err)
	}
}

func findUptimeRow(t *testing.T, rows []UptimeRow, endpointID uint, day string) UptimeRow {
	t.Helper()

	for _, r := range rows {
		if r.EndpointID == endpointID && r.Day.Format("2006-01-02") == day {
			return r
		}
	}

	t.Fatalf("no row for endpoint %d on %s in %+v", endpointID, day, rows)
	return UptimeRow{}
}

func assertUptime(t *testing.T, got, want float64) {
	t.Helper()

	if math.Abs(got-want) > 0.01 {
		t.Errorf("uptime = %f, want %f", got, want)
	}
}

// ---------- carry-in state (lowerbound) ----------

// Last event before the day was ON and nothing happened during the day:
// the server stayed ON for all 24h → 100%.
func TestIntegration_Uptime_CarryInON_NoDayEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 18, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, Date: uDay(2026, 6, 2)},
	}, uDay(2026, 6, 3))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1: %+v", len(rows), rows)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-02")
	assertUptime(t, row.Uptime, 100)
	if !row.HasData {
		t.Error("HasData = false, want true")
	}
}

// Same as above but OFF: server down all day → 0%.
func TestIntegration_Uptime_CarryInOFF_NoDayEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 6, 1, 23, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, Date: uDay(2026, 6, 2)},
	}, uDay(2026, 6, 3))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-02")
	assertUptime(t, row.Uptime, 0)
}

// ---------- in-day events with carry-in ----------

// Carry-in ON from previous night; inside the day ON again at 06:00 then
// OFF at 18:00. The carry-in state holds until the first day event, so the
// server was online continuously 00:00→18:00 = 18h of 24h → 75%
// (matches onlineSeconds: prevStatus starts as the carried-in status).
func TestIntegration_Uptime_ON_OFF_Day_WithCarryIn(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 5, 31, 23, 0))
	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 6, 0))
	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 6, 1, 18, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, Date: uDay(2026, 6, 1)},
	}, uDay(2026, 6, 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.Uptime, 75)
}

// No event ever happened before the queried day: state at midnight is
// unknown, so the window must shrink to the first known event. Events
// OFF@06:00, ON@12:00 → online 12h over an 18h observed window ≈ 66.67%.
func TestIntegration_Uptime_NoHistory_WindowShrinksToFirstEvent(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 6, 1, 6, 0))
	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 12, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, Date: uDay(2026, 6, 1)},
	}, uDay(2026, 6, 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.Uptime, 100.0*12.0/18.0)
}

// ---------- today clamped to `until` ----------

// Queried day is the same calendar day as `until`: window ends at `until`,
// not midnight. Carry-in OFF, ON at 06:00, until 12:00 → 6h online / 12h.
func TestIntegration_Uptime_TodayClampedToUntil(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 5, 31, 23, 0))
	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 6, 0))

	until := uTm(2026, 6, 1, 12, 0)
	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, Date: uDay(2026, 6, 1)},
	}, until)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.Uptime, 50)
}

// ---------- batching across servers and days ----------

// One call, two endpoints × two days: day 2 must inherit day 1's last state
// through its own lowerbound probe.
func TestIntegration_Uptime_MultiServer_MultiDay(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	// Server 7: ON since Jun 1 00:00, goes OFF Jun 2 12:00.
	seedUptimeRow(t, db, 7, domain.StatusOn, uTm(2026, 6, 1, 0, 0))
	seedUptimeRow(t, db, 7, domain.StatusOff, uTm(2026, 6, 2, 12, 0))
	// Server 8: no data at all.

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 7, Date: uDay(2026, 6, 1)},
		{EndpointID: 7, Date: uDay(2026, 6, 2)},
		{EndpointID: 8, Date: uDay(2026, 6, 1)},
	}, uDay(2026, 6, 3))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	d1 := findUptimeRow(t, rows, 7, "2026-06-01")
	assertUptime(t, d1.Uptime, 100)

	d2 := findUptimeRow(t, rows, 7, "2026-06-02")
	assertUptime(t, d2.Uptime, 50)

	for _, r := range rows {
		if r.EndpointID == 8 {
			t.Errorf("endpoint 8 has no events, must not appear in rows: %+v", rows)
		}
	}
}

// ---------- unknown statuses are ignored without advancing the boundary ----------

// A non-ON/OFF status row inside the day must be skipped entirely: the next
// known event still measures from the last known boundary. ON@06, WEIRD@09,
// OFF@18 → ON interval runs 06→18 = 12h over an 18h observed window ≈ 66.67%.
// Counting the weird row as a boundary would give 3h/18h instead.
func TestIntegration_Uptime_IgnoresUnknownStatus(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedRawEvent(t, db, 1, "ON", uTm(2026, 6, 1, 6, 0))
	seedRawEvent(t, db, 1, "DEGRADED", uTm(2026, 6, 1, 9, 0))
	seedRawEvent(t, db, 1, "OFF", uTm(2026, 6, 1, 18, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, Date: uDay(2026, 6, 1)},
	}, uDay(2026, 6, 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.Uptime, 100.0*12.0/18.0)
}

// ---------- no data & duplicate keys ----------

func TestIntegration_Uptime_NoEvents_ReturnsNoRow(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 42, Date: uDay(2026, 6, 1)},
	}, uDay(2026, 6, 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("len(rows) = %d, want 0: %+v", len(rows), rows)
	}
}

// The same (endpoint, day) requested twice collapses to one result row.
func TestIntegration_Uptime_DuplicateRequestKeys(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 0, 0))

	req := []BatchGetOntimeRequest{
		{EndpointID: 1, Date: uDay(2026, 6, 1)},
		{EndpointID: 1, Date: uDay(2026, 6, 1)},
	}

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), req, uDay(2026, 6, 2))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("len(rows) = %d, want 1 (duplicate keys collapse): %+v", len(rows), rows)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.Uptime, 100)
}
