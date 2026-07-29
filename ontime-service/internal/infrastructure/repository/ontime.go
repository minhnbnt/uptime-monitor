package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
)

type BatchGetOntimeRequest struct {
	ServerID uint      `json:"server_id" binding:"required"`
	Date     time.Time `json:"date" binding:"required"`
}

type BatchGetOntimeRangeRequest struct {
	ServerID uint      `json:"server_id" binding:"required"`
	From     time.Time `json:"from_time" binding:"required"`
	To       time.Time `json:"to_time" binding:"required"`
}

type RawEvent struct {
	ServerID uint
	Day      time.Time
	Status   string
	Time     time.Time
	Src      string
}

type RangeEvent struct {
	ServerID    uint
	StartStatus string
	StartTime   time.Time
	Status      string
	Time        time.Time
}

type OntineRepository struct {
	db *gorm.DB
}

func NewOntineRepository(db *gorm.DB) *OntineRepository {
	return &OntineRepository{db: db}
}

func RegisterOntineRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntineRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &OntineRepository{db: dbWrapper.GetDB()}, nil
	})
}

func (r *OntineRepository) BatchGetOntime(ctx context.Context, req []BatchGetOntimeRequest) ([]RawEvent, error) {

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return gorm.G[RawEvent](r.db).Raw(rawEventSQL, string(payload)).Find(ctx)
}

func (r *OntineRepository) BatchGetOntimeRange(ctx context.Context, req []BatchGetOntimeRangeRequest) ([]RangeEvent, error) {

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return gorm.G[RangeEvent](r.db).Raw(rangeEventSQL, string(payload)).Find(ctx)
}

const rawEventSQL = `
	WITH requested AS (
		SELECT *
		FROM jsonb_to_recordset(?::jsonb)
		AS x(server_id bigint, date date)
	),
	lowerbound AS (
		SELECT DISTINCT ON (r.server_id, r.date)
			r.server_id,
			r.date           AS day,
			se.status,
			se.time
		FROM requested r
		LEFT JOIN server_events se ON se.server_id = r.server_id
			AND se.time < r.date
		ORDER BY r.server_id, r.date, se.time DESC
	),
	upperbound AS (
		SELECT DISTINCT ON (r.server_id, r.date)
			r.server_id,
			r.date           AS day,
			se.status,
			se.time
		FROM requested r
		LEFT JOIN server_events se ON se.server_id = r.server_id
			AND se.time < r.date + interval '1 day'
		ORDER BY r.server_id, r.date, se.time DESC
	),
	day_events AS (
		SELECT r.server_id, r.date AS day, se.status, se.time
		FROM requested r
		JOIN server_events se ON se.server_id = r.server_id
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

const rangeEventSQL = `
	WITH requested AS (
		SELECT *
		FROM jsonb_to_recordset(?::jsonb)
		AS x(server_id bigint, from_time timestamp, to_time timestamp)
	),
	lowerbound AS (
		SELECT DISTINCT ON (r.server_id)
			r.server_id,
			se.status AS start_status,
			se.time   AS start_time
		FROM requested r
		LEFT JOIN server_events se ON se.server_id = r.server_id
			AND se.time < r.from_time
		ORDER BY r.server_id, se.time DESC
	),
	range_events AS (
		SELECT r.server_id, se.status, se.time
		FROM requested r
		JOIN server_events se ON se.server_id = r.server_id
			AND se.time >= r.from_time
			AND se.time <= r.to_time
	)
	SELECT
		COALESCE(re.server_id, lb.server_id) AS server_id,
		COALESCE(lb.start_status, 'unknown') AS start_status,
		COALESCE(lb.start_time, r.from_time) AS start_time,
		COALESCE(re.status, '') AS status,
		COALESCE(re.time, r.to_time) AS time
	FROM requested r
	LEFT JOIN lowerbound lb ON lb.server_id = r.server_id
	LEFT JOIN range_events re ON re.server_id = r.server_id
	ORDER BY r.server_id, re.time ASC
`
