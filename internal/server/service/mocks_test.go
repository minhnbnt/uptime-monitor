package service

import (
	"context"
	"io"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
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

type mockEndpointRepo struct {
	upsertEndpointFn       func(ctx context.Context, endpoint domain.Endpoint) error
	deleteByServerIDFn     func(ctx context.Context, serverID uint) error
	batchCreateEndpointsFn func(ctx context.Context, endpoints []domain.Endpoint) error
}

func (m *mockEndpointRepo) UpsertEndpoint(ctx context.Context, endpoint domain.Endpoint) error {
	return m.upsertEndpointFn(ctx, endpoint)
}

func (m *mockEndpointRepo) DeleteByServerID(ctx context.Context, serverID uint) error {
	if m.deleteByServerIDFn == nil {
		return nil
	}
	return m.deleteByServerIDFn(ctx, serverID)
}

func (m *mockEndpointRepo) BatchCreateEndpoints(ctx context.Context, endpoints []domain.Endpoint) error {
	return m.batchCreateEndpointsFn(ctx, endpoints)
}

type mockExcelGenerator struct {
	generateTemplateFn func(w io.Writer) error
	parseImportFileFn  func(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}

func (m *mockExcelGenerator) GenerateTemplate(w io.Writer) error {
	return m.generateTemplateFn(w)
}

func (m *mockExcelGenerator) ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
	return m.parseImportFileFn(file)
}
