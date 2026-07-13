package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

type mockServerService struct {
	listServersFn   func(ctx context.Context, createdByID uint, page, perPage int) ([]dto.Server, int64, error)
	createServerFn  func(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error)
	getServerFn     func(ctx context.Context, id uint) (*dto.Server, error)
	updateServerFn  func(ctx context.Context, id uint, userID uint, req dto.UpdateServerRequest) (*dto.Server, error)
	deleteServerFn  func(ctx context.Context, id uint, userID uint) error
	searchServersFn func(ctx context.Context, params dto.SearchParams, createdByID uint) ([]dto.Server, int64, error)
}

func (m *mockServerService) ListServers(ctx context.Context, createdByID uint, page, perPage int) ([]dto.Server, int64, error) {
	return m.listServersFn(ctx, createdByID, page, perPage)
}
func (m *mockServerService) CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error) {
	return m.createServerFn(ctx, req, createdByID)
}
func (m *mockServerService) GetServer(ctx context.Context, id uint) (*dto.Server, error) {
	return m.getServerFn(ctx, id)
}
func (m *mockServerService) UpdateServer(ctx context.Context, id uint, userID uint, req dto.UpdateServerRequest) (*dto.Server, error) {
	return m.updateServerFn(ctx, id, userID, req)
}
func (m *mockServerService) DeleteServer(ctx context.Context, id uint, userID uint) error {
	return m.deleteServerFn(ctx, id, userID)
}
func (m *mockServerService) SearchServers(ctx context.Context, params dto.SearchParams, createdByID uint) ([]dto.Server, int64, error) {
	if m.searchServersFn == nil {
		return nil, 0, nil
	}
	return m.searchServersFn(ctx, params, createdByID)
}

type mockEndpointService struct {
	setCheckMethodFn func(ctx context.Context, serverID uint, userID uint, req dto.SetCheckMethodRequest) error
	testEndpointFn   func(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error)
}

func (m *mockEndpointService) SetCheckMethod(ctx context.Context, serverID uint, userID uint, req dto.SetCheckMethodRequest) error {
	return m.setCheckMethodFn(ctx, serverID, userID, req)
}

func (m *mockEndpointService) TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {
	return m.testEndpointFn(ctx, req)
}
