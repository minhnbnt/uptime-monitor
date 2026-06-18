package ontime

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/features/server/repository"
	featservice "github.com/minhnbnt/uptime-monitor/internal/features/server/service"
	service "github.com/minhnbnt/uptime-monitor/internal/server/service"
)

func gormModel(id uint, t time.Time) gorm.Model {
	return gorm.Model{ID: id, CreatedAt: t, UpdatedAt: t}
}

type mockServerRepo struct {
	listFn               func(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
	countFn              func(ctx context.Context, createdByID uint) (int64, error)
	createFn             func(ctx context.Context, s *domain.Server) error
	getByIDFn            func(ctx context.Context, id uint) (*domain.Server, error)
	updateFn             func(ctx context.Context, s *domain.Server) error
	deleteFn             func(ctx context.Context, id uint) error
	batchGetOntimeFn     func(ctx context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error)
	batchCreateServersFn func(ctx context.Context, servers []domain.Server) error
}

func (m *mockServerRepo) List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error) {
	return m.listFn(ctx, createdByID, limit, offset)
}

func (m *mockServerRepo) Count(ctx context.Context, createdByID uint) (int64, error) {
	return m.countFn(ctx, createdByID)
}

func (m *mockServerRepo) Create(ctx context.Context, s *domain.Server) error {
	return m.createFn(ctx, s)
}

func (m *mockServerRepo) GetByID(ctx context.Context, id uint) (*domain.Server, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockServerRepo) Update(ctx context.Context, s *domain.Server) error {
	return m.updateFn(ctx, s)
}

func (m *mockServerRepo) Delete(ctx context.Context, id uint) error {
	return m.deleteFn(ctx, id)
}

func (m *mockServerRepo) BatchGetOntime(ctx context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error) {
	return m.batchGetOntimeFn(ctx, req)
}

func (m *mockServerRepo) BatchCreateServers(ctx context.Context, servers []domain.Server) error {
	return m.batchCreateServersFn(ctx, servers)
}

var _ featservice.ServerRepository = (*mockServerRepo)(nil)

type mockOntimeCacheRepo struct {
	mGetFn func(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error)
	mSetFn func(ctx context.Context, items map[dto.BatchGetOntimeItem]float64) error
}

func (m *mockOntimeCacheRepo) MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
	if m.mGetFn != nil {
		return m.mGetFn(ctx, keys)
	}
	return make(map[dto.BatchGetOntimeItem]float64), nil
}

func (m *mockOntimeCacheRepo) MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]float64) error {
	if m.mSetFn == nil {
		return nil
	}
	return m.mSetFn(ctx, items)
}

var _ service.OntimeCacheRepository = (*mockOntimeCacheRepo)(nil)
