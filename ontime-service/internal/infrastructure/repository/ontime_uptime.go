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
// the SQL measures, Go presents.
type UptimeRow struct {
	EndpointID    uint
	From          time.Time
	To            time.Time
	HasData       bool
	ObservedFrom  time.Time
	ObservedTo    time.Time
	OnlineSeconds float64
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
// requested (endpoint, window) that has any known ON/OFF event in or before
// it. Windows with no data produce no row — callers pre-seed their own zero
// results for those.
//
// Semantics: state at window start carries in from the last prior event;
// a window whose state was never known shrinks its start to the first known
// event inside it.
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

	return gorm.G[UptimeRow](r.db).Raw(uptimeSQL, string(payload)).Find(ctx)
}

const uptimeSQL = `
	WITH
	-- 1. Parse the batch request payload into (endpoint, window) pairs.
	requested AS (
		SELECT x.endpoint_id, x.from_ts, x.to_ts
		FROM jsonb_to_recordset(?::jsonb)
		AS x(endpoint_id bigint, from_ts timestamptz, to_ts timestamptz)
	),

	-- 2. Ordered so the window LEAD below can incremental-sort instead of
	--    spilling a global sort to disk.
	windows AS (
		SELECT endpoint_id,
			from_ts AS win_start,
			to_ts AS win_end
		FROM requested
		ORDER BY endpoint_id, from_ts
	),

	-- 3. Single status gate: only real ON/OFF states drive uptime.
	--    Anything else is skipped WITHOUT advancing the boundary,
	--    same as the historical ToServerStatus gate in onlineSeconds.
	--    NOT MATERIALIZED is load-bearing: materialized (default when a CTE
	--    is referenced twice), every per-window probe would scan the
	--    tuplestore instead of hitting idx_endpoint_time — measured 51s vs 1.7s.
	known_events AS NOT MATERIALIZED (
		SELECT endpoint_id, status, time
		FROM server_events
		WHERE status IN ('ON', 'OFF')
	),

	-- 4. One ordered stream: for each window, its starting state (last known
	--    event strictly before win_start — the lowerbound probe) plus every
	--    known event inside the window itself, both via index lookups.
	timeline AS (
		SELECT w.endpoint_id, w.win_start, w.win_end, tl.status, tl.time, tl.src
		FROM windows w
		CROSS JOIN LATERAL (
			(
				SELECT ke.status, ke.time, 'carryin'::text AS src
				FROM known_events ke
				WHERE ke.endpoint_id = w.endpoint_id
					AND ke.time < w.win_start
				ORDER BY ke.time DESC
				LIMIT 1
			)
			UNION ALL
			(
				SELECT ke.status, ke.time, 'dayev'::text AS src
				FROM known_events ke
				WHERE ke.endpoint_id = w.endpoint_id
					AND ke.time >= w.win_start
					AND ke.time < w.win_end
			)
		) tl
		ORDER BY endpoint_id, win_start, time
	),

	-- 5. Raw lookahead only: when does this state stop being true?
	--    The last row of a partition has no successor → NULL, patched next
	--    step. Partitioned by the full window triple: two windows sharing a
	--    start but not an end must never share a LEAD chain.
	next_times AS (
		SELECT
			*,
			LEAD(time)
			OVER (PARTITION BY endpoint_id, win_start, win_end ORDER BY time) AS next_time
		FROM timeline
	),

	-- 6. Turn rows into half-open [seg_from, seg_to) intervals clamped to the
	--    window: carry-in starts at win_start, nothing runs past win_end.
	segments AS (
		SELECT endpoint_id,
			win_start, win_end,
			status, src,
			GREATEST(time, win_start) AS seg_from,
			LEAST(COALESCE(next_time, win_end), win_end) AS seg_to
		FROM next_times
	)

	-- 7. Measure per window: seconds spent ON plus where observation really
	--     began and ended. Windows whose starting state was known cover the
	--     full span; otherwise coverage starts at the first known event.
	SELECT endpoint_id,
		win_start AS "from",
		win_end AS "to",
		true AS has_data,
		CASE WHEN bool_or(src = 'carryin')
			THEN MIN(win_start)
			ELSE MIN(seg_from)
		END AS observed_from,
		MAX(win_end) AS observed_to,
		COALESCE(
			SUM(EXTRACT(EPOCH FROM (seg_to - seg_from))) FILTER (WHERE status = 'ON'),
			0
		) AS online_seconds
	FROM segments
	GROUP BY endpoint_id, win_start, win_end
`
