package repository

import (
	"context"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
)

type CurrentStatus struct {
	EndpointID uint
	Status     string
}

type EventRepository struct {
	db *gorm.DB
}

func NewEventRepository(db *gorm.DB) *EventRepository {
	return &EventRepository{db: db}
}

func RegisterEventRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &EventRepository{db: dbWrapper.GetDB()}, nil
	})
}

func (r *EventRepository) GetCurrentStatuses(
	ctx context.Context, endpointIDs []uint,
) ([]CurrentStatus, error) {

	if len(endpointIDs) == 0 {
		return nil, nil
	}

	latest := r.db.WithContext(ctx).
		Select("DISTINCT ON (endpoint_id) endpoint_id, status").
		Table("server_events").
		Where("endpoint_id IN ?", endpointIDs).
		Order("endpoint_id, time DESC")

	rows, err := gorm.G[CurrentStatus](r.db).
		Table("(?) AS latest", latest).
		Find(ctx)

	if err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *EventRepository) CountByStatus(
	ctx context.Context, endpointIDs []uint,
) (online, offline int64, err error) {

	if len(endpointIDs) == 0 {
		return 0, 0, nil
	}

	latest := r.db.WithContext(ctx).
		Select("DISTINCT ON (endpoint_id) endpoint_id, status").
		Table("server_events").
		Where("endpoint_id IN ?", endpointIDs).
		Order("endpoint_id, time DESC")

	type counts struct {
		Online  int64 `gorm:"column:online"`
		Offline int64 `gorm:"column:offline"`
	}

	c := counts{}
	err = gorm.G[counts](r.db).
		Select(`
			COUNT(*) FILTER (WHERE status = 'ON') AS online,
			COUNT(*) FILTER (WHERE status = 'OFF') AS offline
		`).
		Table("(?) AS latest", latest).
		Scan(ctx, &c)

	if err != nil {
		return 0, 0, err
	}

	return c.Online, c.Offline, nil
}

func (r *EventRepository) CountByStatusByUserID(
	ctx context.Context, userID uint,
) (online, offline int64, err error) {

	type counts struct {
		Online  int64 `gorm:"column:online"`
		Offline int64 `gorm:"column:offline"`
	}

	c := counts{}
	err = gorm.G[counts](r.db).
		Select(`
			COUNT(*) FILTER (WHERE latest.status = 'ON') AS online,
			COUNT(*) FILTER (WHERE latest.status = 'OFF') AS offline
		`).
		Table(`(
			SELECT DISTINCT ON (se.endpoint_id) se.endpoint_id, se.status
			FROM server_events se
			JOIN server_owners so ON so.server_id = se.endpoint_id
			WHERE so.user_id = ? AND so.deleted_at IS NULL
			ORDER BY se.endpoint_id, se.time DESC
		) AS latest`, userID).
		Scan(ctx, &c)

	if err != nil {
		return 0, 0, err
	}

	return c.Online, c.Offline, nil
}
