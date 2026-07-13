package importer

import (
	"context"
	"io"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/features/importer/dto"
	featservice "github.com/minhnbnt/uptime-monitor/internal/features/server/service"
)

type mockServerRepo struct {
	listFn               func(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error)
	countFn              func(ctx context.Context, createdByID uint) (int64, error)
	createFn             func(ctx context.Context, s *domain.Server) error
	getByIDFn            func(ctx context.Context, id uint) (*domain.Server, error)
	updateFn             func(ctx context.Context, s *domain.Server) error
	deleteFn             func(ctx context.Context, id uint) error
	batchCreateServersFn func(ctx context.Context, servers []domain.Server) error
}

func (m *mockServerRepo) List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error) {
	return m.listFn(ctx, createdByID, limit, offset)
}
func (m *mockServerRepo) Count(ctx context.Context, createdByID uint) (int64, error) {
	return m.countFn(ctx, createdByID)
}
func (m *mockServerRepo) CountByStatus(ctx context.Context, createdByID uint) (total, online, offline int64, err error) {
	return 0, 0, 0, nil
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

func (m *mockServerRepo) BatchCreateServers(ctx context.Context, servers []domain.Server) error {
	if m.batchCreateServersFn == nil {
		return nil
	}
	return m.batchCreateServersFn(ctx, servers)
}

var _ featservice.ServerRepository = (*mockServerRepo)(nil)

type mockEndpointRepo struct {
	upsertEndpointFn       func(ctx context.Context, endpoint domain.Endpoint) error
	deleteByServerIDFn     func(ctx context.Context, serverID uint) error
	batchCreateEndpointsFn func(ctx context.Context, endpoints []domain.Endpoint) error
	updateMonitorStatusFn  func(ctx context.Context, endpointID uint, status domain.ServerStatus) error
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

func (m *mockEndpointRepo) UpdateMonitorStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error {
	if m.updateMonitorStatusFn == nil {
		return nil
	}
	return m.updateMonitorStatusFn(ctx, endpointID, status)
}

var _ featservice.EndpointRepository = (*mockEndpointRepo)(nil)

type mockExcelParser struct {
	parseImportFileFn func(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error)
}

func (m *mockExcelParser) ParseImportFile(file io.Reader) ([]dto.ImportRow, []dto.ImportRowError, error) {
	return m.parseImportFileFn(file)
}
