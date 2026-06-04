package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

type mockAuthService struct {
	registerFn func(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	loginFn    func(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
}

func (m *mockAuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	return m.registerFn(ctx, req)
}
func (m *mockAuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	return m.loginFn(ctx, req)
}

type mockServerService struct {
	listServersFn   func(ctx context.Context, page, perPage int) ([]dto.Server, error)
	createServerFn  func(ctx context.Context, req dto.CreateServerRequest) (*dto.Server, error)
	getServerFn     func(ctx context.Context, id uint) (*dto.Server, error)
	updateServerFn  func(ctx context.Context, id uint, req dto.UpdateServerRequest) (*dto.Server, error)
	deleteServerFn  func(ctx context.Context, id uint) error
}

func (m *mockServerService) ListServers(ctx context.Context, page, perPage int) ([]dto.Server, error) {
	return m.listServersFn(ctx, page, perPage)
}
func (m *mockServerService) CreateServer(ctx context.Context, req dto.CreateServerRequest) (*dto.Server, error) {
	return m.createServerFn(ctx, req)
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

type mockOntimeService struct {
	listServersWithOntimeFn func(ctx context.Context, page, perPage int) ([]dto.ServerWithOntime, int64, error)
}

func (m *mockOntimeService) ListServersWithOntime(ctx context.Context, page, perPage int) ([]dto.ServerWithOntime, int64, error) {
	return m.listServersWithOntimeFn(ctx, page, perPage)
}

type mockEndpointService struct {
	setCheckMethodFn func(ctx context.Context, serverID uint, req dto.SetCheckMethodRequest) error
}

func (m *mockEndpointService) SetCheckMethod(ctx context.Context, serverID uint, req dto.SetCheckMethodRequest) error {
	return m.setCheckMethodFn(ctx, serverID, req)
}
