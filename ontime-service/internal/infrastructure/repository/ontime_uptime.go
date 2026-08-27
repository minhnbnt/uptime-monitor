package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
)

// BatchGetOntimeRequest is one half-open observation window [From, To) for
// one endpoint. Windows may be arbitrary — whole days, mid-day slices, or
// multi-day spans; clamping to "now" is the caller's job.
type BatchGetOntimeRequest struct {
	EndpointID uint      `json:"endpoint_id" binding:"required"`
	From       time.Time `json:"from_ts" binding:"required"`
	To         time.Time `json:"to_ts" binding:"required"`
}

// UptimeRow is one fully-computed per-window measurement returned by
// Postgres. From/To echo the requested window; Observed* report where data
// actually began and ended (a window with no prior history shrinks its
// start forward). Callers derive the display percentage via UptimePercent —
// the SQL measures, Go presents. HasData is derived in Go after scanning:
// a window whose whole observed span is UNKNOWN counts as having no usable
// data rather than silently inheriting a stale state.
type UptimeRow struct {
	EndpointID     uint
	From           time.Time
	To             time.Time
	HasData        bool
	ObservedFrom   time.Time
	ObservedTo     time.Time
	OnlineSeconds  float64
	UnknownSeconds float64
}

// UptimePercent turns the measured seconds into the familiar 0–100 figure.
// A non-positive observation window yields 0 rather than NaN, mirroring the
// historical calcUptimePercent guard.
func (r UptimeRow) UptimePercent() float64 {

	total := r.ObservedTo.Sub(r.ObservedFrom).Seconds()
	if total <= 0 {
		return 0
	}

	return r.OnlineSeconds / total * 100
}

type OntimeUptimeRepository struct {
	db *gorm.DB
}

func NewOntimeUptimeRepository(db *gorm.DB) *OntimeUptimeRepository {
	return &OntimeUptimeRepository{db: db}
}

func RegisterOntimeUptimeRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeUptimeRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &OntimeUptimeRepository{db: dbWrapper.GetDB()}, nil
	})
}

// BatchGetUptime computes uptime entirely in SQL and returns one row per
// requested (endpoint, window) that has any event in or before it. Windows
// with no data produce no row — callers pre-seed their own zero results for
// those.
//
// Semantics: state at window start carries in from the last prior event —
// including UNKNOWN, so a window opening after a silence gap starts as
// "unknown" instead of inheriting a stale ON/OFF. A window whose observed
// span is entirely UNKNOWN is reported by Go as having no usable data.
func (r *OntimeUptimeRepository) BatchGetUptime(ctx context.Context, requests []BatchGetOntimeRequest) ([]UptimeRow, error) {

	invalidIndex := slices.IndexFunc(requests, func(req BatchGetOntimeRequest) bool {
		return !req.To.After(req.From)
	})

	if invalidIndex >= 0 {
		target := requests[invalidIndex]
		return nil, fmt.Errorf(
			"invalid window for endpoint %d: to %v must be after from %v",
			target.EndpointID, target.To, target.From,
		)
	}

	payload, err := json.Marshal(requests)
	if err != nil {
		return nil, err
	}

	request := r.db.Raw(windowFromJsonb, string(payload))
	rows, err := gorm.G[UptimeRow](r.db).Raw(uptimeSQL, request).Find(ctx)
	if err != nil {
		return nil, err
	}

	for i := range rows {
		rows[i].HasData = deriveHasData(rows[i])
	}

	return rows, nil
}

// freshnessEpsilon absorbs float wobble from EXTRACT(EPOCH) when deciding
// whether unknown time covers the whole observed span.
const freshnessEpsilon = 1e-3

func deriveHasData(row UptimeRow) bool {

	total := row.ObservedTo.Sub(row.ObservedFrom).Seconds()
	if total <= 0 {
		return false
	}

	return row.UnknownSeconds < total-freshnessEpsilon
}

const windowFromJsonb = `
	SELECT request_item.endpoint_id, request_item.from_ts, request_item.to_ts
	FROM jsonb_to_recordset(?::jsonb)
	AS request_item(endpoint_id bigint, from_ts timestamptz, to_ts timestamptz)
`

const uptimeSQL = `
	-- 1. Ordered so the window LEAD below can incremental-sort instead of
	--    spilling a global sort to disk.
	WITH windows AS (
		SELECT
			request_item.endpoint_id,
			request_item.from_ts AS window_start,
			request_item.to_ts AS window_end
		FROM (?) request_item
		ORDER BY request_item.endpoint_id, request_item.from_ts
	),

	-- 2. Single status gate: every real state drives the timeline, including
	--    UNKNOWN so silences split segments instead of being bridged by a
	--    stale carry-in. NOT MATERIALIZED is load-bearing: materialized
	--    (default when a CTE is referenced twice), every per-window probe
	--    would scan the tuplestore instead of hitting idx_endpoint_time —
	--    measured 51s vs 1.7s.
	known_events AS NOT MATERIALIZED (
		SELECT endpoint_id, status, time
		FROM server_events
		WHERE status IN ('ON', 'OFF', 'UNKNOWN')
	),

	-- 3. One ordered stream: for each window, its starting state (last known
	--    event strictly before window_start — the lowerbound probe) plus
	--    every known event inside the window itself, both via index lookups.
	timeline AS (
		SELECT
			observation_window.endpoint_id,
			observation_window.window_start,
			observation_window.window_end,
			event_stream.status,
			event_stream.time,
			event_stream.source
		FROM windows observation_window
		CROSS JOIN LATERAL (
			(
				SELECT known_event.status, known_event.time, 'carryin'::text AS source
				FROM known_events known_event
				WHERE known_event.endpoint_id = observation_window.endpoint_id
					AND known_event.time < observation_window.window_start
				ORDER BY known_event.time DESC
				LIMIT 1
			)
			UNION ALL
			(
				SELECT known_event.status, known_event.time, 'dayev'::text AS source
				FROM known_events known_event
				WHERE known_event.endpoint_id = observation_window.endpoint_id
					AND known_event.time >= observation_window.window_start
					AND known_event.time < observation_window.window_end
			)
		) event_stream
		ORDER BY endpoint_id, window_start, time
	),

	-- 4. Raw lookahead only: when does this state stop being true?
	--    The last row of a partition has no successor → NULL, patched next
	--    step. Partitioned by the full window triple: two windows sharing a
	--    start but not an end must never share a LEAD chain.
	next_times AS (
		SELECT
			*,
			LEAD(time) OVER (
				PARTITION BY endpoint_id, window_start, window_end
				ORDER BY time
			) AS next_event_time
		FROM timeline
	),

	-- 5. Turn rows into half-open [segment_start, segment_end) intervals
	--    clamped to the window: carry-in starts at window_start, nothing
	--    runs past window_end.
	segments AS (
		SELECT
			endpoint_id,
			window_start, window_end,
			status,
			source,
			GREATEST(time, window_start) AS segment_start,
			LEAST(next_event_time, window_end) AS segment_end
		FROM next_times
	)

	-- 6. Measure per window: seconds spent ON, seconds UNKNOWN, plus where
	--     observation really began and ended. Windows whose starting state
	--     was known cover the full span; otherwise coverage starts at the
	--     first known event. HasData is derived in Go, not fetched.
	SELECT
	    endpoint_id,
	    window_start AS "from", window_end AS "to",

	    CASE
	        WHEN bool_or(source = 'carryin')
	            THEN window_start
	        ELSE MIN(segment_start)
	    END AS observed_from,

	    window_end AS observed_to,

	    COALESCE(
	        SUM(EXTRACT(EPOCH FROM (segment_end - segment_start)))
	            FILTER (WHERE status = 'ON'),
	        0
	    ) AS online_seconds,

	    COALESCE(
	        SUM(EXTRACT(EPOCH FROM (segment_end - segment_start)))
	            FILTER (WHERE status = 'UNKNOWN'),
	        0
	    ) AS unknown_seconds
	FROM segments
	GROUP BY endpoint_id, window_start, window_end
`
