package repository

import (
	"context"
	"encoding/json"
	"time"
)

type BatchGetOntimeRequest struct {
	ServerID uint      `json:"server_id" binding:"required"`
	Date     time.Time `json:"date" binding:"required"`
}

type RawEvent struct {
	ServerID uint
	Day      time.Time
	Status   string
	Time     time.Time
	Src      string
}

func (sr *ServerRepository) BatchGetOntime(ctx context.Context, req []BatchGetOntimeRequest) ([]RawEvent, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	var rows []RawEvent

	err = sr.db.WithContext(ctx).
		Raw(rawEventSQL, string(payload)).
		Scan(&rows).Error

	return rows, err
}

const rawEventSQL = `
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
	lowerbound AS (
		SELECT DISTINCT ON (em.server_id, r.date)
			em.server_id,
			r.date           AS day,
			se.status,
			se.time
		FROM requested r
		JOIN endpoint_map em ON em.server_id = r.server_id
		LEFT JOIN server_events se ON se.endpoint_id = em.endpoint_id
			AND se.time < r.date
		ORDER BY em.server_id, r.date, se.time DESC
	),
	upperbound AS (
		SELECT DISTINCT ON (em.server_id, r.date)
			em.server_id,
			r.date           AS day,
			se.status,
			se.time
		FROM requested r
		JOIN endpoint_map em ON em.server_id = r.server_id
		LEFT JOIN server_events se ON se.endpoint_id = em.endpoint_id
			AND se.time < r.date + interval '1 day'
		ORDER BY em.server_id, r.date, se.time DESC
	),
	day_events AS (
		SELECT em.server_id, r.date AS day, se.status, se.time
		FROM requested r
		JOIN endpoint_map em ON em.server_id = r.server_id
		JOIN server_events se ON se.endpoint_id = em.endpoint_id
			AND se.time >= r.date
			AND se.time < r.date + interval '1 day'
	),
	combined AS (
		SELECT server_id, day, status, time, 'lowerbound' AS src FROM lowerbound WHERE status IS NOT NULL
		UNION ALL
		SELECT server_id, day, status, time, 'upperbound' AS src FROM upperbound WHERE status IS NOT NULL
		UNION ALL
		SELECT server_id, day, status, time, 'day_event' AS src FROM day_events
	)
	SELECT server_id, day, status, time, src
	FROM combined
	ORDER BY server_id, day, time ASC
`
