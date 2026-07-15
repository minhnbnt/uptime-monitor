package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
)

type mockAuthService struct {
	registerFn func(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	loginFn    func(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	refreshFn  func(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error)
	logoutFn   func(ctx context.Context, refreshToken string) error
	getUserFn  func(ctx context.Context, id uint) (*dto.UserProfile, error)
}

func (m *mockAuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
	return m.registerFn(ctx, req)
}

func (m *mockAuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {
	return m.loginFn(ctx, req)
}

func (m *mockAuthService) Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error) {
	if m.refreshFn == nil {
		return nil, nil
	}
	return m.refreshFn(ctx, req)
}

func (m *mockAuthService) Logout(ctx context.Context, refreshToken string) error {
	if m.logoutFn == nil {
		return nil
	}
	return m.logoutFn(ctx, refreshToken)
}

func (m *mockAuthService) GetUser(ctx context.Context, id uint) (*dto.UserProfile, error) {
	if m.getUserFn == nil {
		return nil, nil
	}
	return m.getUserFn(ctx, id)
}
