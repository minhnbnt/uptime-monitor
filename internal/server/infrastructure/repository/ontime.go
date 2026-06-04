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

type RawEvent struct {
	ServerID uint
	Day      time.Time
	Status   string
	Time     time.Time
	Src      string
}

func (sr *ServerRepository) BatchGetOntime(ctx context.Context, req []BatchGetOntimeRequest) ([]BatchGetOntimeResponse, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	rows, err := sr.collectRawEvents(ctx, payload)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := truncateDay(now)

	type dayKey struct {
		ServerID uint
		Day      time.Time
	}

	groups := make(map[dayKey][]RawEvent)
	for _, row := range rows {
		key := dayKey{ServerID: row.ServerID, Day: row.Day}
		groups[key] = append(groups[key], row)
	}

	serverResults := make(map[uint][]OntimeResult)
	for _, r := range req {
		key := dayKey{ServerID: r.ServerID, Day: truncateDay(r.Date)}
		events := groups[key]
		stats := calculateDayOntime(events, today, now)
		serverResults[r.ServerID] = append(serverResults[r.ServerID], OntimeResult{
			Date:  r.Date,
			Stats: stats,
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

func (sr *ServerRepository) collectRawEvents(ctx context.Context, payload []byte) ([]RawEvent, error) {
	var rows []RawEvent

	err := sr.db.WithContext(ctx).
		Raw(rawEventSQL, string(payload)).
		Scan(&rows).Error

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func calculateDayOntime(events []RawEvent, today time.Time, now time.Time) float64 {
	if len(events) == 0 {
		return 0
	}

	day := truncateDay(events[0].Day)
	dayStart := day
	dayEnd := day.Add(24 * time.Hour)
	isToday := day.Equal(today)

	startStatus := ""
	startTime := dayStart
	var dayEvents []RawEvent

	for _, e := range events {
		if e.Time.Before(dayStart) {
			startStatus = e.Status
			if isToday {
				startTime = e.Time
			}
		} else if e.Time.Before(dayEnd) {
			dayEvents = append(dayEvents, e)
		}
	}

	if startStatus == "" {
		switch {
		case len(dayEvents) > 0:
			startStatus = dayEvents[0].Status
			if isToday {
				startTime = dayEvents[0].Time
			}
		case len(events) > 0:
			startStatus = events[0].Status
			if isToday {
				startTime = events[0].Time
			}
		}
	}

	if len(dayEvents) > 1 {
		unique := []RawEvent{dayEvents[0]}
		for i := 1; i < len(dayEvents); i++ {
			if !dayEvents[i].Time.Equal(dayEvents[i-1].Time) {
				unique = append(unique, dayEvents[i])
			}
		}
		dayEvents = unique
	}

	prevTime := startTime
	prevStatus := startStatus
	var totalOnline float64

	for _, e := range dayEvents {
		if domain.ServerStatus(prevStatus) == domain.StatusOn {
			totalOnline += e.Time.Sub(prevTime).Seconds()
		}
		prevStatus = e.Status
		prevTime = e.Time
	}

	endTime := dayEnd
	if isToday {
		endTime = now
	}

	if domain.ServerStatus(prevStatus) == domain.StatusOn {
		dur := endTime.Sub(prevTime).Seconds()
		if dur > 0 {
			totalOnline += dur
		}
	}

	var coverage float64
	if isToday {
		coverage = now.Sub(startTime).Seconds()
	} else {
		coverage = dayEnd.Sub(dayStart).Seconds()
	}

	if coverage <= 0 {
		if domain.ServerStatus(startStatus) == domain.StatusOn {
			return 100
		}
		return 0
	}

	return totalOnline / coverage * 100
}

func truncateDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
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
