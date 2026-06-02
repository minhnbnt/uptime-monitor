package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type BatchGetOntimeRequest struct {
	ServerID uint      `json:"server_id" binding:"required"`
	Date     time.Time `json:"date" binding:"required"`
}

type OntimeResult struct {
	Date  time.Time
	Stats float64
}

type BatchGetOntimeResponse struct {
	ServerID uint
	Result   []OntimeResult
}

func (sr *ServerRepository) BatchGetOntime(ctx context.Context, req []BatchGetOntimeRequest) ([]BatchGetOntimeResponse, error) {

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	rows, err := sr.collectEventRows(ctx, payload)
	if err != nil {
		return nil, err
	}

	uptime, coverage := accumulate(rows)
	ensureAllDays(uptime, coverage, req)

	const oneDay = 24 * 60 * 60

	serverResults := make(map[uint][]OntimeResult)
	for key, upt := range uptime {

		cov, ok := coverage[key]
		if !ok {
			cov = oneDay
		}

		serverResults[key.ServerID] = append(serverResults[key.ServerID], OntimeResult{
			Date:  key.Day,
			Stats: upt / cov * 100,
		})
	}

	results := make([]BatchGetOntimeResponse, 0, len(serverResults))
	for serverID, ontimeResults := range serverResults {
		results = append(results, BatchGetOntimeResponse{
			ServerID: serverID,
			Result:   ontimeResults,
		})
	}
	return results, nil
}

func (sr *ServerRepository) collectEventRows(ctx context.Context, payload []byte) ([]eventRow, error) {

	var rows []eventRow

	err := sr.db.WithContext(ctx).
		Raw(sql, string(payload)).
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func accumulate(rows []eventRow) (map[resultKey]float64, map[resultKey]float64) {

	uptime := make(map[resultKey]float64)
	coverage := make(map[resultKey]float64)

	for _, row := range rows {

		key := resultKey{ServerID: row.ServerID, Day: row.Day}
		coverage[key] = row.CoverageSeconds

		if domain.ServerStatus(row.Status) == domain.StatusOn {
			uptime[key] += row.DurationSeconds
		}
	}

	return uptime, coverage
}

func ensureAllDays(uptime, coverage map[resultKey]float64, req []BatchGetOntimeRequest) {
	for _, r := range req {
		day := time.Date(
			r.Date.Year(),
			r.Date.Month(),
			r.Date.Day(),
			0, 0, 0, 0,
			r.Date.Location(),
		)

		key := resultKey{ServerID: r.ServerID, Day: day}
		if _, ok := uptime[key]; ok {
			continue
		}

		uptime[key] = 0
		if _, hasCoverage := coverage[key]; !hasCoverage {
			coverage[key] = 86400
		}
	}
}

// this's ok but we have make it look prettier
const sql = `
	WITH requested AS (
		SELECT *
		FROM jsonb_to_recordset(?::jsonb)
		AS x(server_id bigint, date date)
	),
	endpoint_map AS (
		SELECT id AS endpoint_id, server_id
		FROM endpoints
		WHERE server_id IN (SELECT server_id FROM requested)
		  AND deleted_at IS NULL
	),
	prev_events AS (
		SELECT DISTINCT ON (em.server_id, r.date)
			em.server_id,
			r.date AS day,
			se.status,
			r.date AS occurred_at
		FROM requested r
		JOIN endpoint_map em ON em.server_id = r.server_id
		LEFT JOIN server_events se ON se.endpoint_id = em.endpoint_id
			AND se.time < r.date
		ORDER BY em.server_id, r.date, se.time DESC
	),
	day_events AS (
		SELECT em.server_id, r.date AS day, se.status, se.time AS occurred_at
		FROM requested r
		JOIN endpoint_map em ON em.server_id = r.server_id
		JOIN server_events se ON se.endpoint_id = em.endpoint_id
			AND se.time >= r.date
			AND se.time < r.date + interval '1 day'
	),
	combined AS (
		SELECT server_id, day, status, occurred_at, true AS from_prev
		FROM prev_events
		WHERE status IS NOT NULL
		UNION ALL
		SELECT server_id, day, status, occurred_at, false AS from_prev
		FROM day_events
	),
	day_bounds AS (
		SELECT server_id, day,
			CASE
				WHEN bool_or(from_prev) THEN 86400
				ELSE EXTRACT(EPOCH FROM (day + interval '1 day' - MIN(occurred_at)))
			END AS coverage_seconds
		FROM combined
		GROUP BY server_id, day
	),
	ordered AS (
		SELECT
			c.server_id,
			c.day,
			c.status,
			c.occurred_at,
			LEAD(c.occurred_at, 1, c.day + interval '1 day') OVER (
				PARTITION BY c.server_id, c.day
				ORDER BY c.occurred_at ASC
			) AS next_occurred_at,
			db.coverage_seconds
		FROM combined c
		JOIN day_bounds db ON db.server_id = c.server_id AND db.day = c.day
	)
	SELECT
		server_id,
		day,
		status,
		occurred_at,
		COALESCE(EXTRACT(EPOCH FROM (next_occurred_at - occurred_at)), 0) AS duration_seconds,
		coverage_seconds
	FROM ordered
	WHERE next_occurred_at > occurred_at
	ORDER BY server_id, day, occurred_at ASC
`

type eventRow struct {
	ServerID        uint
	Day             time.Time
	Status          string
	OccurredAt      time.Time
	DurationSeconds float64
	CoverageSeconds float64
}

type resultKey struct {
	ServerID uint
	Day      time.Time
}
