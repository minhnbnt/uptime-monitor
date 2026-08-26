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
		if r.EndpointID == endpointID && r.From.Format("2006-01-02") == day {
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

// Last event before the window was ON and nothing happened during it:
// the server stayed ON for the full 24h → 100%.
func TestIntegration_Uptime_CarryInON_NoDayEvents(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 18, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, From: uDay(2026, 6, 2), To: uDay(2026, 6, 3)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1: %+v", len(rows), rows)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-02")
	assertUptime(t, row.UptimePercent(), 100)
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
		{EndpointID: 1, From: uDay(2026, 6, 2), To: uDay(2026, 6, 3)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-02")
	assertUptime(t, row.UptimePercent(), 0)
}

// ---------- in-day events with carry-in ----------

// Carry-in ON from previous night; inside the window ON again at 06:00 then
// OFF at 18:00. The carry-in state holds until the first window event, so
// the server was online continuously 00:00→18:00 = 18h of 24h → 75%.
func TestIntegration_Uptime_ON_OFF_Day_WithCarryIn(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 5, 31, 23, 0))
	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 6, 0))
	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 6, 1, 18, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.UptimePercent(), 75)
}

// No event ever happened before the queried window: state at the start is
// unknown, so the window must shrink to the first known event. Events
// OFF@06:00, ON@12:00 → online 12h over an 18h observed window ≈ 66.67%.
func TestIntegration_Uptime_NoHistory_WindowShrinksToFirstEvent(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 6, 1, 6, 0))
	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 12, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.UptimePercent(), 100.0*12.0/18.0)
}

// ---------- windows not aligned to midnight ----------

// A window ending mid-day: carry-in OFF, ON at 06:00, window ends 12:00 →
// 6h online of a 12h observed window → 50%. This used to require special
// "today + until" handling; now it is just an ordinary From/To.
func TestIntegration_Uptime_WindowEndMidDay(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 5, 31, 23, 0))
	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 6, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, From: uDay(2026, 6, 1), To: uTm(2026, 6, 1, 12, 0)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.UptimePercent(), 50)
}

// A window starting and ending mid-day proves no midnight alignment is
// assumed anywhere. Carry-in OFF, ON@10:00, OFF@20:00 over [09:00, 21:00):
// online 10h of 12h ≈ 83.33%.
func TestIntegration_Uptime_ArbitraryMidDayRange(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 5, 31, 23, 0))
	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 10, 0))
	seedUptimeRow(t, db, 1, domain.StatusOff, uTm(2026, 6, 1, 20, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, From: uTm(2026, 6, 1, 9, 0), To: uTm(2026, 6, 1, 21, 0)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.UptimePercent(), 100.0*10.0/12.0)
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
		{EndpointID: 7, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
		{EndpointID: 7, From: uDay(2026, 6, 2), To: uDay(2026, 6, 3)},
		{EndpointID: 8, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	d1 := findUptimeRow(t, rows, 7, "2026-06-01")
	assertUptime(t, d1.UptimePercent(), 100)

	d2 := findUptimeRow(t, rows, 7, "2026-06-02")
	assertUptime(t, d2.UptimePercent(), 50)

	for _, r := range rows {
		if r.EndpointID == 8 {
			t.Errorf("endpoint 8 has no events, must not appear in rows: %+v", rows)
		}
	}
}

// ---------- unknown statuses are ignored without advancing the boundary ----------

// A non-ON/OFF status row inside the window must be skipped entirely: the
// next known event still measures from the last known boundary. ON@06,
// WEIRD@09, OFF@18 → ON interval runs 06→18 = 12h over an 18h observed
// window ≈ 66.67%. Counting the weird row as a boundary would give 3h/18h.
func TestIntegration_Uptime_IgnoresUnknownStatus(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedRawEvent(t, db, 1, "ON", uTm(2026, 6, 1, 6, 0))
	seedRawEvent(t, db, 1, "DEGRADED", uTm(2026, 6, 1, 9, 0))
	seedRawEvent(t, db, 1, "OFF", uTm(2026, 6, 1, 18, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.UptimePercent(), 100.0*12.0/18.0)
}

// ---------- no data & duplicate keys & invalid input ----------

func TestIntegration_Uptime_NoEvents_ReturnsNoRow(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 42, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Errorf("len(rows) = %d, want 0: %+v", len(rows), rows)
	}
}

// The same window requested twice collapses to one result row.
func TestIntegration_Uptime_DuplicateRequestKeys(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 0, 0))

	req := []BatchGetOntimeRequest{
		{EndpointID: 1, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
		{EndpointID: 1, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	}

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 {
		t.Errorf("len(rows) = %d, want 1 (duplicate keys collapse): %+v", len(rows), rows)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.UptimePercent(), 100)
}

func TestIntegration_Uptime_InvalidWindow_ReturnsError(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	repo := NewOntimeUptimeRepository(db)

	cases := map[string]BatchGetOntimeRequest{
		"to equals from": {EndpointID: 1, From: uDay(2026, 6, 1), To: uDay(2026, 6, 1)},
		"to before from": {EndpointID: 1, From: uDay(2026, 6, 2), To: uDay(2026, 6, 1)},
	}

	for name, req := range cases {
		if _, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{req}); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

// ---------- unknown segments ----------

// ON for the first half of the day, UNKNOWN from noon onward: online and
// unknown split the window exactly, and the window still has data.
func TestIntegration_Uptime_UnknownSplitsWindow(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 1, domain.StatusOn, uTm(2026, 6, 1, 0, 0))
	seedUptimeRow(t, db, 1, domain.StatusUnknown, uTm(2026, 6, 1, 12, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 1, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 1, "2026-06-01")
	assertUptime(t, row.OnlineSeconds, 12*3600)
	assertUptime(t, row.UnknownSeconds, 12*3600)
	if !row.HasData {
		t.Error("partially unknown window must still report has_data")
	}
}

// OFF after a mid-day unknown: the unknown gap must not swallow either side.
func TestIntegration_Uptime_UnknownBetweenKnownStates(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 2, domain.StatusOn, uTm(2026, 6, 1, 0, 0))
	seedUptimeRow(t, db, 2, domain.StatusUnknown, uTm(2026, 6, 1, 6, 0))
	seedUptimeRow(t, db, 2, domain.StatusOff, uTm(2026, 6, 1, 12, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 2, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 2, "2026-06-01")
	assertUptime(t, row.OnlineSeconds, 6*3600)
	assertUptime(t, row.UnknownSeconds, 6*3600)
	if !row.HasData {
		t.Error("window with known states around the gap must report has_data")
	}
}

// The last known event before the window is followed by an UNKNOWN: the
// carry-in itself is UNKNOWN, so the whole observed span is unknown and the
// window reports no usable data instead of inheriting the stale ON state.
func TestIntegration_Uptime_FullyUnknownWindow_HasNoData(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initUptimeTestDB(t)

	seedUptimeRow(t, db, 3, domain.StatusOn, uTm(2026, 5, 31, 18, 0))
	seedUptimeRow(t, db, 3, domain.StatusUnknown, uTm(2026, 5, 31, 23, 0))

	repo := NewOntimeUptimeRepository(db)
	rows, err := repo.BatchGetUptime(t.Context(), []BatchGetOntimeRequest{
		{EndpointID: 3, From: uDay(2026, 6, 1), To: uDay(2026, 6, 2)},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	row := findUptimeRow(t, rows, 3, "2026-06-01")
	assertUptime(t, row.OnlineSeconds, 0)
	assertUptime(t, row.UnknownSeconds, 24*3600)
	if row.HasData {
		t.Error("fully unknown window must report has_data=false")
	}
}
