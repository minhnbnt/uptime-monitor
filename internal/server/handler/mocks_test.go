package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

type mockAuthService struct {
	registerFn func(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	loginFn    func(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	refreshFn  func(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error)
}

func (m *mockAuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	return m.registerFn(ctx, req)
}
func (m *mockAuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	return m.loginFn(ctx, req)
}
func (m *mockAuthService) Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error) {
	return m.refreshFn(ctx, req)
}
func (m *mockAuthService) Logout(ctx context.Context, refreshToken string) error {
	return nil
}

type mockServerService struct {
	listServersFn   func(ctx context.Context, createdByID uint, page, perPage int) ([]dto.Server, error)
	createServerFn  func(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error)
	getServerFn     func(ctx context.Context, id uint) (*dto.Server, error)
	updateServerFn  func(ctx context.Context, id uint, req dto.UpdateServerRequest) (*dto.Server, error)
	deleteServerFn  func(ctx context.Context, id uint) error
	searchServersFn func(ctx context.Context, q string, createdByID uint, page, perPage int) ([]dto.Server, int64, error)
}

func (m *mockServerService) ListServers(ctx context.Context, createdByID uint, page, perPage int) ([]dto.Server, error) {
	return m.listServersFn(ctx, createdByID, page, perPage)
}
func (m *mockServerService) CreateServer(ctx context.Context, req dto.CreateServerRequest, createdByID uint) (*dto.Server, error) {
	return m.createServerFn(ctx, req, createdByID)
}
func (m *mockServerService) GetServer(ctx context.Context, id uint) (*dto.Server, error) {
	return m.getServerFn(ctx, id)
}
func (m *mockServerService) UpdateServer(ctx context.Context, id uint, req dto.UpdateServerRequest) (*dto.Server, error) {
	return m.updateServerFn(ctx, id, req)
}
func (m *mockServerService) DeleteServer(ctx context.Context, id uint) error {
	return m.deleteServerFn(ctx, id)
}
func (m *mockServerService) SearchServers(ctx context.Context, q string, createdByID uint, page, perPage int) ([]dto.Server, int64, error) {
	if m.searchServersFn == nil {
		return nil, 0, nil
	}
	return m.searchServersFn(ctx, q, createdByID, page, perPage)
}

type mockOntimeService struct {
	listServersWithOntimeFn func(ctx context.Context, createdByID uint, page, perPage int) ([]dto.ServerWithOntime, int64, error)
	getServerWithOntimeFn   func(ctx context.Context, serverID uint) (*dto.ServerWithOntime, error)
}

func (m *mockOntimeService) ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]dto.ServerWithOntime, int64, error) {
	return m.listServersWithOntimeFn(ctx, createdByID, page, perPage)
}

func (m *mockOntimeService) GetServerWithOntime(ctx context.Context, serverID uint) (*dto.ServerWithOntime, error) {
	if m.getServerWithOntimeFn == nil {
		return nil, nil
	}
	return m.getServerWithOntimeFn(ctx, serverID)
}

type mockEndpointService struct {
	setCheckMethodFn func(ctx context.Context, serverID uint, req dto.SetCheckMethodRequest) error
	testEndpointFn   func(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error)
}

func (m *mockEndpointService) SetCheckMethod(ctx context.Context, serverID uint, req dto.SetCheckMethodRequest) error {
	return m.setCheckMethodFn(ctx, serverID, req)
}

func (m *mockEndpointService) TestEndpoint(ctx context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {
	return m.testEndpointFn(ctx, req)
}
