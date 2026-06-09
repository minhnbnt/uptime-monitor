package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/repository/ontime"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
)

func gormModel(id uint, t time.Time) gorm.Model {
	return gorm.Model{ID: id, CreatedAt: t, UpdatedAt: t}
}

type mockServerRepo struct {
	listFn           func(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
	countFn          func(ctx context.Context, createdByID uint) (int64, error)
	createFn         func(ctx context.Context, s *domain.Server) error
	getByIDFn        func(ctx context.Context, id uint) (*domain.Server, error)
	updateFn         func(ctx context.Context, s *domain.Server) error
	deleteFn         func(ctx context.Context, id uint) error
	batchGetOntimeFn func(ctx context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error)
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

type mockEndpointRepo struct {
	upsertEndpointFn func(ctx context.Context, endpoint domain.Endpoint) error
}

func (m *mockEndpointRepo) UpsertEndpoint(ctx context.Context, endpoint domain.Endpoint) error {
	return m.upsertEndpointFn(ctx, endpoint)
}

type mockOntimeCacheRepo struct {
	mGetFn func(ctx context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error)
	mSetFn func(ctx context.Context, items map[ontimerepo.OntimeCacheKey]float64) error
}

func (m *mockOntimeCacheRepo) MGet(ctx context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
	if m.mGetFn == nil {
		return nil, nil
	}
	return m.mGetFn(ctx, keys)
}

func (m *mockOntimeCacheRepo) MSet(ctx context.Context, items map[ontimerepo.OntimeCacheKey]float64) error {
	if m.mSetFn == nil {
		return nil
	}
	return m.mSetFn(ctx, items)
}

type mockLogger struct {
	infoCalled bool
	warnCalled bool
	lastMsg    string
}

func (m *mockLogger) Info(msg string, fields ...logger.Field) {
	m.infoCalled = true
	m.lastMsg = msg
}
func (m *mockLogger) Warn(msg string, fields ...logger.Field) {
	m.warnCalled = true
	m.lastMsg = msg
}
func (m *mockLogger) Error(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Debug(msg string, fields ...logger.Field)  {}
func (m *mockLogger) Fatal(msg string, fields ...logger.Field)  {}
func (m *mockLogger) With(fields ...logger.Field) logger.Logger { return m }
