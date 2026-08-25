package repository

import (
	"context"
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// UptimeRow is one fully-computed per-(endpoint, day) uptime result.
type UptimeRow struct {
	EndpointID uint
	Day        time.Time
	HasData    bool
	Uptime     float64
}

type OntimeUptimeRepository struct {
	db *gorm.DB
}

func NewOntimeUptimeRepository(db *gorm.DB) *OntimeUptimeRepository {
	return &OntimeUptimeRepository{db: db}
}

// BatchGetUptime computes uptime entirely in SQL and returns one row per
// requested (endpoint, date) pair that has any known ON/OFF event in or
// before its window. Pairs with no data produce no row — callers pre-seed
// their own zero results for those.
//
// Semantics match OntimeCalculator: state at midnight carries in from the
// last prior event; a day whose state was never known shrinks its window to
// the first known event; windows ending "today" are clamped to until.
func (r *OntimeUptimeRepository) BatchGetUptime(ctx context.Context, req []BatchGetOntimeRequest, until time.Time) ([]UptimeRow, error) {

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var rows []UptimeRow
	if err := r.db.WithContext(ctx).Raw(uptimeSQL, string(payload), until, until).Scan(&rows).Error; err != nil {
		return nil, err
	}

	return rows, nil
}

const uptimeSQL = `
	WITH
	-- 1. Parse the batch request payload into (endpoint, day) pairs.
	requested AS (
		SELECT x.endpoint_id, x.date AS day
		FROM jsonb_to_recordset(?::jsonb)
		AS x(endpoint_id bigint, date date)
	),

	-- 2. Which calendar day counts as "today", derived from until.
	--    Matches CalculateDayOntime: today.Equal(utils.TruncateDay(now)).
	params AS (
		SELECT (?::timestamptz AT TIME ZONE 'UTC')::date AS today
	),

	-- 3. Every requested day gets its observation window:
	--    past days end at midnight, today ends at until.
	--    Ordered so the window LEAD below can incremental-sort instead of
	--    spilling a global sort to disk.
	windows AS (
		SELECT r.endpoint_id,
			r.day,
			r.day::timestamp AS day_start,
			CASE
				WHEN r.day = p.today THEN ?::timestamptz
				ELSE r.day::timestamp + interval '1 day'
			END AS day_end
		FROM requested r
		CROSS JOIN params p
		ORDER BY r.endpoint_id, r.day
	),

	-- 4. Single status gate: only real ON/OFF states drive uptime.
	--    Anything else is skipped WITHOUT advancing the boundary,
	--    same as ToServerStatus returning unknown in onlineSeconds.
	--    NOT MATERIALIZED is load-bearing: materialized (default when a CTE
	--    is referenced twice), every per-day probe would scan the tuplestore
	--    instead of hitting idx_endpoint_time — measured 51s vs 1.7s.
	known_events AS NOT MATERIALIZED (
		SELECT endpoint_id, status, time
		FROM server_events
		WHERE status IN ('ON', 'OFF')
	),

	-- 5. One ordered stream: for each window, its midnight state (last known
	--    event strictly before day_start — the lowerbound probe) plus every
	--    known event inside the window itself, both via index lookups.
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
					AND ke.time >= w.day_start
					AND ke.time < w.day_end
			)
		) tl
		ORDER BY endpoint_id, day, time
	),

	-- 6. Raw lookahead only: when does this state stop being true?
	--    The last row of a group has no successor → NULL, patched next step.
	next_times AS (
		SELECT *,
			LEAD(time) OVER (PARTITION BY endpoint_id, day ORDER BY time) AS next_time
		FROM timeline
	),

	-- 7. Turn rows into half-open [seg_from, seg_to) intervals clamped to the
	--    window: carry-in starts at midnight, nothing runs past day_end.
	--    Replaces EndTime - prevTime at the tail of onlineSeconds.
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

	-- 8. Measure per day: seconds spent ON, whether midnight state was
	--     known, and where observation really began (first event if the
	--     state at midnight was unknown — newTimeline's shrunk window).
	per_day AS (
		SELECT endpoint_id,
			day,
			day_end,
			bool_or(src = 'carryin') AS has_carryin,
			MIN(seg_from) AS first_seen,
			COALESCE(
				SUM(EXTRACT(EPOCH FROM (seg_to - seg_from))) FILTER (WHERE status = 'ON'),
				0
			) AS online_seconds
		FROM segments
		GROUP BY endpoint_id, day, day_end
	)

	-- 9. Uptime = online seconds over the observed window. Days whose
	--     midnight state was known cover the full window; otherwise coverage
	--     starts at the first known event (calcUptimePercent guards /0).
	SELECT endpoint_id,
		day,
		true AS has_data,
		COALESCE(
			100.0 * online_seconds
			/ NULLIF(EXTRACT(EPOCH FROM (
				day_end - CASE WHEN has_carryin THEN day::timestamp ELSE first_seen END
			)), 0)
		, 0) AS uptime
	FROM per_day
`
