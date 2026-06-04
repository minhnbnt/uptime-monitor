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

type Timeline struct {
	Day         time.Time
	StartTime   time.Time
	EndTime     time.Time
	StartStatus string
	Events      []RawEvent
}

func calculateDayOntime(events []RawEvent, today time.Time, now time.Time) float64 {

	if len(events) == 0 {
		return 0
	}

	t := buildTimeline(events, today, now)
	online := calculateOnlineDuration(t)
	coverage := t.EndTime.Sub(t.StartTime).Seconds()

	if coverage > 0 {
		return online / coverage * 100
	}

	if domain.ServerStatus(t.StartStatus) == domain.StatusOn {
		return 100
	}
	return 0
}

func buildTimeline(events []RawEvent, today time.Time, now time.Time) Timeline {
	day := truncateDay(events[0].Day)

	t := Timeline{
		Day:       day,
		StartTime: day,
		EndTime:   day.Add(24 * time.Hour),
	}

	if day.Equal(today) {
		t.EndTime = now
	}

	prevEvents, dayEvents := splitByDayBoundary(events, day)
	applyStartState(&t, prevEvents, dayEvents, events, day.Equal(today))
	t.Events = dedupEvents(dayEvents)

	return t
}

func splitByDayBoundary(events []RawEvent, day time.Time) (prev, inside []RawEvent) {
	dayEnd := day.Add(24 * time.Hour)

	for _, e := range events {
		if e.Time.Before(day) {
			prev = append(prev, e)
		} else if e.Time.Before(dayEnd) {
			inside = append(inside, e)
		}
	}

	return
}

func applyStartState(t *Timeline, prevEvents, dayEvents, allEvents []RawEvent, isToday bool) {

	if len(prevEvents) > 0 {

		last := prevEvents[len(prevEvents)-1]

		t.StartStatus = last.Status
		if isToday {
			t.StartTime = last.Time
		}

		return
	}

	if len(allEvents) == 0 {
		return
	}

	fallback := allEvents[0]

	switch {
	case len(dayEvents) > 0:
		fallback = dayEvents[0]
	}

	t.StartStatus = fallback.Status
	if isToday {
		t.StartTime = fallback.Time
	}
}

func dedupEvents(events []RawEvent) []RawEvent {

	if len(events) <= 1 {
		return events
	}

	unique := []RawEvent{events[0]}
	for i := 1; i < len(events); i++ {
		if !events[i].Time.Equal(events[i-1].Time) {
			unique = append(unique, events[i])
		}
	}

	return unique
}

func calculateOnlineDuration(t Timeline) float64 {

	prevTime := t.StartTime
	prevStatus := t.StartStatus
	var total float64

	for _, e := range t.Events {
		if domain.ServerStatus(prevStatus) == domain.StatusOn {
			total += e.Time.Sub(prevTime).Seconds()
		}
		prevStatus = e.Status
		prevTime = e.Time
	}

	if domain.ServerStatus(prevStatus) == domain.StatusOn {
		dur := t.EndTime.Sub(prevTime).Seconds()
		if dur > 0 {
			total += dur
		}
	}

	return total
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
