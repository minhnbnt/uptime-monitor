package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/testcontainers"
)

const (
	benchEndpoints = 20
	benchDays      = 30

	// 30 days of pings every 30s per endpoint → ~1.7M rows, same shape as the
	// throwaway benchmark/main.go sample.
	benchSlotsPerEndpoint = 30 * 24 * 60 * 60 / 30 // 86400
)

var (
	benchOnce     sync.Once
	benchDB       *gorm.DB
	benchRequests []BatchGetOntimeRequest
)

// setupBench starts a throwaway Postgres, seeds it once, and builds the
// request fan-out. Mirrors benchmark/main.go's container + seed recipe, but
// lazy so it only runs when a benchmark actually executes.
func setupBench(b *testing.B) {
	b.Helper()

	benchOnce.Do(func() {
		ctx := context.Background()

		_, dsn := testcontainers.StartPostgres(ctx)
		db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			b.Fatalf("open bench db: %v", err)
		}
		benchDB = db

		if err := db.AutoMigrate(&domain.ServerEvent{}, &domain.ServerOwner{}); err != nil {
			b.Fatalf("migrate: %v", err)
		}

		base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		if err := seedBench(db, base); err != nil {
			b.Fatalf("seed: %v", err)
		}

		benchRequests = buildBenchRequests(base)

		payload, err := json.Marshal(benchRequests)
		if err != nil {
			b.Fatalf("marshal payload: %v", err)
		}
		explainBench(db, string(payload))
	})
}

// explainBench prints EXPLAIN (ANALYZE, BUFFERS) for the real uptime query,
// building it exactly like BatchGetUptime does (payload bound as a param, so
// the plan shows $1 instead of a giant jsonb literal).
func explainBench(db *gorm.DB, payload string) {
	request := db.Raw(windowFromJsonb, payload)
	q := "EXPLAIN (ANALYZE, BUFFERS) " + uptimeSQL

	rows, err := db.Raw(q, request).Rows()
	if err != nil {
		fmt.Printf("explain error: %v\n", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n================ EXPLAIN ANALYZE — BatchGetUptime (warm) ================")
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			fmt.Printf("explain scan error: %v\n", err)
			return
		}
		fmt.Println(line)
	}
}

func seedBench(db *gorm.DB, base time.Time) error {
	fmt.Printf("== seeding %d endpoints x %d days (%d rows) ==\n",
		benchEndpoints, benchDays, benchEndpoints*benchSlotsPerEndpoint)
	started := time.Now()

	res := db.Exec(`
		INSERT INTO server_events (id, endpoint_id, status, "time")
		SELECT gen_random_uuid(),
			ep,
			CASE
				WHEN (g.s % 19200) < 96 THEN 'UNKNOWN'
				WHEN (g.s % 9600) < 288 THEN 'OFF'
				ELSE 'ON'
			END,
			$1::timestamptz + make_interval(secs => g.s * 30)
		FROM generate_series(1, $2) AS e(ep),
		     generate_series(0, $3 - 1) AS g(s)
	`, base.Format(time.RFC3339), benchEndpoints, benchSlotsPerEndpoint)
	if res.Error != nil {
		return fmt.Errorf("seed events: %w", res.Error)
	}
	n := res.RowsAffected
	fmt.Printf("seeded %d rows in %s\n", n, time.Since(started).Round(time.Millisecond))

	// A little history before the window so carry-in probes hit real data.
	if err := db.Exec(`
		INSERT INTO server_events (id, endpoint_id, status, "time")
		SELECT gen_random_uuid(), ep, 'ON', $1::timestamptz - interval '48 hours'
		FROM generate_series(1, $2) AS e(ep)
	`, base.Format(time.RFC3339), benchEndpoints).Error; err != nil {
		return fmt.Errorf("seed history: %w", err)
	}

	// Planner stats on fresh data — autovacuum won't have run yet.
	if err := db.Exec(`ANALYZE server_events`).Error; err != nil {
		return fmt.Errorf("analyze: %w", err)
	}
	return nil
}

// buildBenchRequests covers every endpoint for every day as a half-open
// [From,To) window, matching how the production caller fans out daily uptime.
func buildBenchRequests(base time.Time) []BatchGetOntimeRequest {
	reqs := make([]BatchGetOntimeRequest, 0, benchEndpoints*benchDays)
	for ep := 1; ep <= benchEndpoints; ep++ {
		for d := 0; d < benchDays; d++ {
			from := base.AddDate(0, 0, d)
			reqs = append(reqs, BatchGetOntimeRequest{
				EndpointID: uint(ep),
				From:       from,
				To:         from.AddDate(0, 0, 1),
			})
		}
	}
	return reqs
}

// BenchmarkBatchGetUptime runs the real SQL-backed query over the seeded data.
// One warmup call shakes out the cold cache, then the timer starts.
func BenchmarkBatchGetUptime(b *testing.B) {
	setupBench(b)

	repo := NewOntimeUptimeRepository(benchDB)
	ctx := context.Background()

	// Warm up: first execution pays for plan cache + buffer fills.
	if _, err := repo.BatchGetUptime(ctx, benchRequests); err != nil {
		b.Fatalf("warmup: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rows, err := repo.BatchGetUptime(ctx, benchRequests)
		if err != nil {
			b.Fatalf("BatchGetUptime: %v", err)
		}
		b.ReportMetric(float64(len(rows)), "rows")
	}
}
