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

type Event struct {
	AnchorTime time.Time
	Status     string
	Time       time.Time
	Src        string
}

type ServerEvent struct {
	ServerID uint
	Event    Event `gorm:"embedded"`
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

func (r *OntineRepository) BatchGetOntime(ctx context.Context, req []BatchGetOntimeRequest) ([]ServerEvent, error) {

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return gorm.G[ServerEvent](r.db).Raw(dayEventSQL, string(payload)).Find(ctx)
}

func (r *OntineRepository) BatchGetOntimeRange(ctx context.Context, req []BatchGetOntimeRangeRequest) ([]ServerEvent, error) {

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	return gorm.G[ServerEvent](r.db).Raw(rangeEventSQL, string(payload)).Find(ctx)
}

const dayEventSQL = `
	WITH requested AS (
		SELECT *
		FROM jsonb_to_recordset(?::jsonb)
		AS x(server_id bigint, date date)
	),
	lowerbound AS (
		SELECT DISTINCT ON (r.server_id, r.date)
			r.server_id,
			r.date::timestamp AS anchor_time,
			se.status,
			se.time,
			se.id
		FROM requested r
		LEFT JOIN server_events se ON se.server_id = r.server_id
			AND se.time < r.date
		ORDER BY r.server_id, r.date, se.time DESC, se.id DESC
	),
	upperbound AS (
		SELECT DISTINCT ON (r.server_id, r.date)
			r.server_id,
			r.date::timestamp AS anchor_time,
			se.status,
			se.time,
			se.id
		FROM requested r
		LEFT JOIN server_events se ON se.server_id = r.server_id
			AND se.time < r.date + interval '1 day'
		ORDER BY r.server_id, r.date, se.time DESC, se.id DESC
	),
	day_events AS (
		SELECT r.server_id, r.date::timestamp AS anchor_time, se.status, se.time, se.id
		FROM requested r
		JOIN server_events se ON se.server_id = r.server_id
			AND se.time >= r.date
			AND se.time < r.date + interval '1 day'
	),
	combined AS (
		SELECT server_id, anchor_time, status, time, id, 'lowerbound' AS src FROM lowerbound WHERE status IS NOT NULL
		UNION ALL
		SELECT server_id, anchor_time, status, time, id, 'upperbound' AS src FROM upperbound WHERE status IS NOT NULL
		UNION ALL
		SELECT server_id, anchor_time, status, time, id, 'day_event' AS src FROM day_events
	)
	SELECT server_id, anchor_time, status, time, id, src
	FROM combined
	ORDER BY server_id, anchor_time, time ASC, id ASC
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
			r.from_time   AS anchor_time,
			se.status,
			r.from_time   AS time,
			se.id
		FROM requested r
		LEFT JOIN server_events se ON se.server_id = r.server_id
			AND se.time < r.from_time
		WHERE se.status IS NOT NULL
		ORDER BY r.server_id, se.time DESC, se.id DESC
	),
	range_events AS (
		SELECT r.server_id, r.from_time AS anchor_time, se.status, se.time, se.id
		FROM requested r
		JOIN server_events se ON se.server_id = r.server_id
			AND se.time >= r.from_time
			AND se.time <= r.to_time
	)
	SELECT server_id, anchor_time, status, time, id, 'lowerbound' AS src FROM lowerbound
	UNION ALL
	SELECT server_id, anchor_time, status, time, id, 'event' AS src FROM range_events
	ORDER BY server_id, time ASC, id ASC
`
