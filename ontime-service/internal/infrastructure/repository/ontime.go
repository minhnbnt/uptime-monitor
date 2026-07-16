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
	EndpointID uint      `json:"endpoint_id" binding:"required"`
	Date       time.Time `json:"date" binding:"required"`
}

type RawEvent struct {
	EndpointID uint
	Day        time.Time
	Status     string
	Time       time.Time
	Src        string
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

const rawEventSQL = `
	WITH requested AS (
		SELECT *
		FROM jsonb_to_recordset(?::jsonb)
		AS x(endpoint_id bigint, date date)
	),
	lowerbound AS (
		SELECT DISTINCT ON (r.endpoint_id, r.date)
			r.endpoint_id,
			r.date           AS day,
			se.status,
			se.time
		FROM requested r
		LEFT JOIN server_events se ON se.endpoint_id = r.endpoint_id
			AND se.time < r.date
		ORDER BY r.endpoint_id, r.date, se.time DESC
	),
	upperbound AS (
		SELECT DISTINCT ON (r.endpoint_id, r.date)
			r.endpoint_id,
			r.date           AS day,
			se.status,
			se.time
		FROM requested r
		LEFT JOIN server_events se ON se.endpoint_id = r.endpoint_id
			AND se.time < r.date + interval '1 day'
		ORDER BY r.endpoint_id, r.date, se.time DESC
	),
	day_events AS (
		SELECT r.endpoint_id, r.date AS day, se.status, se.time
		FROM requested r
		JOIN server_events se ON se.endpoint_id = r.endpoint_id
			AND se.time >= r.date
			AND se.time < r.date + interval '1 day'
	),
	combined AS (
		SELECT endpoint_id, day, status, time, 'lowerbound' AS src FROM lowerbound WHERE status IS NOT NULL
		UNION ALL
		SELECT endpoint_id, day, status, time, 'upperbound' AS src FROM upperbound WHERE status IS NOT NULL
		UNION ALL
		SELECT endpoint_id, day, status, time, 'day_event' AS src FROM day_events
	)
	SELECT endpoint_id, day, status, time, src
	FROM combined
	ORDER BY endpoint_id, day, time ASC
`
