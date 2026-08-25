// Throwaway benchmark: seeds ~1.7M server_events (20 endpoints x 30 days x
// one ping / 30s) into a throwaway Postgres, then EXPLAIN ANALYZE's the
// LEAD-based uptime query with the baseline index vs a covering index.
// Not part of any test suite — run manually: go run .
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	endpoints = 20
	days      = 30

	// 30 days of pings every 30s per endpoint.
	slotsPerEndpoint = 30 * 24 * 60 * 60 / 30 // 86400
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "bench:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	dsn, stopContainer, err := startPostgres(ctx)
	if err != nil {
		return err
	}
	defer stopContainer()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	if err := createSchema(db); err != nil {
		return err
	}

	base := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	until := base.AddDate(0, 0, days)

	if err := seed(db, base); err != nil {
		return err
	}

	payload, err := buildPayload(base)
	if err != nil {
		return err
	}

	fmt.Println("== smoke run (real execution) ==")
	started := time.Now()
	rows, err := execUptime(db, payload, until)
	if err != nil {
		return fmt.Errorf("smoke run: %w", err)
	}
	fmt.Printf("returned %d rows in %s\n", len(rows), time.Since(started).Round(time.Millisecond))
	printSample(rows)

	for _, label := range []string{
		"BASELINE index (endpoint_id,time) — cold",
		"BASELINE index (endpoint_id,time) — warm",
	} {
		if err := printExplain(db, label, payload, until); err != nil {
			return err
		}
	}

	return nil
}

func startPostgres(ctx context.Context) (string, func(), error) {

	req := tc.ContainerRequest{
		Image:        "postgres:18-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "bench",
			"POSTGRES_PASSWORD": "bench",
			"POSTGRES_DB":       "benchdb",
			"PGTZ":              "UTC",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(120 * time.Second),
	}

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return "", nil, fmt.Errorf("start postgres: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return "", nil, err
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		return "", nil, err
	}

	dsn := fmt.Sprintf("postgres://bench:bench@%s:%s/benchdb?sslmode=disable", host, port.Port())
	return dsn, func() { _ = container.Terminate(ctx) }, nil
}

// Mirrors domain.ServerEvent + gorm's idx_endpoint_time.
func createSchema(db *sql.DB) error {

	stmts := []string{
		`CREATE TABLE server_events (
			id uuid PRIMARY KEY,
			endpoint_id bigint NOT NULL,
			status varchar(20) NOT NULL,
			"time" timestamptz NOT NULL
		)`,
		`CREATE INDEX idx_endpoint_time ON server_events (endpoint_id, "time")`,
	}

	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("schema: %w", err)
		}
	}
	return nil
}

func seed(db *sql.DB, base time.Time) error {

	fmt.Printf("== seeding %d endpoints x %d days (%d rows) ==\n",
		endpoints, days, endpoints*slotsPerEndpoint)
	started := time.Now()

	// One event every 30s; every 9600th slot opens a 288-slot (~2.4h)
	// OFF window so the ON/OFF mix is realistic-ish. Deterministic.
	res, err := db.Exec(`
		INSERT INTO server_events (id, endpoint_id, status, "time")
		SELECT gen_random_uuid(),
			ep,
			CASE WHEN (g.s % 9600) < 288 THEN 'OFF' ELSE 'ON' END,
			$1::timestamptz + make_interval(secs => g.s * 30)
		FROM generate_series(1, $2) AS e(ep),
		     generate_series(0, $3 - 1) AS g(s)
	`, base.Format(time.RFC3339), endpoints, slotsPerEndpoint)
	if err != nil {
		return fmt.Errorf("seed events: %w", err)
	}
	n, _ := res.RowsAffected()
	fmt.Printf("seeded %d rows in %s\n", n, time.Since(started).Round(time.Millisecond))

	// A little history before the window so carry-in probes hit real data.
	if _, err := db.Exec(`
		INSERT INTO server_events (id, endpoint_id, status, "time")
		SELECT gen_random_uuid(), ep, 'ON', $1::timestamptz - interval '48 hours'
		FROM generate_series(1, $2) AS e(ep)
	`, base.Format(time.RFC3339), endpoints); err != nil {
		return fmt.Errorf("seed history: %w", err)
	}

	var total int
	if err := db.QueryRow(`SELECT count(*) FROM server_events`).Scan(&total); err != nil {
		return err
	}
	want := endpoints*slotsPerEndpoint + endpoints
	if total != want {
		return fmt.Errorf("row count = %d, want %d", total, want)
	}

	// Planner stats on fresh data — autovacuum won't have run yet.
	if _, err := db.Exec(`ANALYZE server_events`); err != nil {
		return fmt.Errorf("analyze: %w", err)
	}

	return nil
}

type pair struct {
	EndpointID uint   `json:"endpoint_id"`
	Date       string `json:"date"`
}

func buildPayload(base time.Time) (string, error) {

	reqs := make([]pair, 0, endpoints*days)
	for ep := 1; ep <= endpoints; ep++ {
		for d := 0; d < days; d++ {
			reqs = append(reqs, pair{
				EndpointID: uint(ep),
				Date:       base.AddDate(0, 0, d).Format("2006-01-02"),
			})
		}
	}

	b, err := json.Marshal(reqs)
	return string(b), err
}

type benchRow struct {
	EndpointID    uint
	Day           time.Time
	HasData       bool
	OnlineSeconds float64
	ObservedFrom  time.Time
	ObservedTo    time.Time
}

func execUptime(db *sql.DB, payload string, until time.Time) ([]benchRow, error) {

	rs, err := db.Query(uptimeSQL, payload, until, until)
	if err != nil {
		return nil, err
	}
	defer rs.Close()

	var out []benchRow
	for rs.Next() {
		var r benchRow
		if err := rs.Scan(&r.EndpointID, &r.Day, &r.HasData, &r.ObservedFrom, &r.ObservedTo, &r.OnlineSeconds); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rs.Err()
}

func printSample(rows []benchRow) {

	online := 0
	for _, r := range rows {
		if r.OnlineSeconds > 0 {
			online++
		}
	}
	if len(rows) > 0 {
		r := rows[len(rows)/2]
		pct := 0.0
		if secs := r.ObservedTo.Sub(r.ObservedFrom).Seconds(); secs > 0 {
			pct = r.OnlineSeconds / secs * 100
		}
		fmt.Printf("sample: endpoint=%d day=%s online=%.0fs observed=[%s → %s] = %.2f%%\n",
			r.EndpointID, r.Day.Format("2006-01-02"), r.OnlineSeconds,
			r.ObservedFrom.Format(time.RFC3339), r.ObservedTo.Format(time.RFC3339), pct)
	}
	fmt.Printf("rows with nonzero online seconds: %d/%d\n\n", online, len(rows))
}

func printExplain(db *sql.DB, label, payload string, until time.Time) error {
	return printExplainConn(context.Background(), db, label, payload, until)
}

func printExplainConn(ctx context.Context, qx interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}, label, payload string, until time.Time) error {

	litPayload := "'" + strings.ReplaceAll(payload, "'", "''") + "'"
	litUntil := "timestamptz '" + until.Format("2006-01-02 15:04:05+00") + "'"

	q := uptimeSQL
	q = strings.Replace(q, "$1::jsonb", litPayload+"::jsonb", 1)
	q = strings.Replace(q, "$2::timestamptz AT TIME ZONE 'UTC'", litUntil+" AT TIME ZONE 'UTC'", 1)
	q = strings.Replace(q, "$3::timestamptz", litUntil, 1)

	fmt.Printf("\n================ EXPLAIN ANALYZE — %s ================\n", label)

	started := time.Now()
	rs, err := qx.QueryContext(ctx, "EXPLAIN (ANALYZE, BUFFERS) "+q)
	if err != nil {
		return fmt.Errorf("explain (%s): %w", label, err)
	}
	defer rs.Close()

	for rs.Next() {
		var line string
		if err := rs.Scan(&line); err != nil {
			return err
		}
		fmt.Println(line)
	}
	if err := rs.Err(); err != nil {
		return err
	}
	fmt.Printf("(explain wall time incl. execution: %s)\n", time.Since(started).Round(time.Millisecond))
	return nil
}

const uptimeSQL = `
	WITH
	requested AS (
		SELECT x.endpoint_id, x.date AS day
		FROM jsonb_to_recordset($1::jsonb)
		AS x(endpoint_id bigint, date date)
	),
	params AS (
		SELECT ($2::timestamptz AT TIME ZONE 'UTC')::date AS today
	),
	windows AS (
		SELECT r.endpoint_id,
			r.day,
			r.day::timestamp AS day_start,
			CASE
				WHEN r.day = p.today THEN $3::timestamptz
				ELSE r.day::timestamp + interval '1 day'
			END AS day_end
		FROM requested r
		CROSS JOIN params p
		ORDER BY r.endpoint_id, r.day
	),
	known_events AS NOT MATERIALIZED (
		SELECT endpoint_id, status, time
		FROM server_events
		WHERE status IN ('ON', 'OFF')
	),
	timeline AS (
		SELECT w.endpoint_id, w.day, w.day_end, tl.status, tl.time, tl.src
		FROM windows w
		CROSS JOIN LATERAL (
			(
				SELECT ke.status, ke.time, 'carryin'::text AS src
				FROM known_events ke
				WHERE ke.endpoint_id = w.endpoint_id
					AND ke.time < w.day
				ORDER BY ke.time DESC
				LIMIT 1
			)
			UNION ALL
			(
				SELECT ke.status, ke.time, 'dayev'::text AS src
				FROM known_events ke
				WHERE ke.endpoint_id = w.endpoint_id
					AND ke.time >= w.day
					AND ke.time < w.day_end
			)
		) tl
		ORDER BY endpoint_id, day, time
	),
	next_times AS (
		SELECT *,
			LEAD(time) OVER (PARTITION BY endpoint_id, day ORDER BY time) AS next_time
		FROM timeline
	),
	segments AS (
		SELECT endpoint_id,
			day,
			day_end,
			status,
			src,
			GREATEST(time, day) AS seg_from,
			LEAST(COALESCE(next_time, day_end), day_end) AS seg_to
		FROM next_times
	),
	per_day AS (
		SELECT endpoint_id,
			day,
			MAX(day_end) AS observed_to,
			CASE WHEN bool_or(src = 'carryin')
				THEN MIN(day)::timestamp
				ELSE MIN(seg_from)
			END AS observed_from,
			COALESCE(
				SUM(EXTRACT(EPOCH FROM (seg_to - seg_from))) FILTER (WHERE status = 'ON'),
				0
			) AS online_seconds
		FROM segments
		GROUP BY endpoint_id, day
	)
	SELECT endpoint_id,
		day,
		true AS has_data,
		observed_from,
		observed_to,
		online_seconds
	FROM per_day
`
