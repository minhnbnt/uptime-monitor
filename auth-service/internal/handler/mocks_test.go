package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
)

type mockAuthService struct {
	registerFn          func(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	loginFn             func(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	refreshFn           func(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error)
	getUserFn           func(ctx context.Context, id uint) (*dto.UserProfile, error)
	createPingSessionFn func(ctx context.Context, userID uint) (*dto.AuthResponse, error)
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

func (m *mockAuthService) GetUser(ctx context.Context, id uint) (*dto.UserProfile, error) {
	if m.getUserFn == nil {
		return nil, nil
	}
	return m.getUserFn(ctx, id)
}

func (m *mockAuthService) CreatePingSession(ctx context.Context, userID uint) (*dto.AuthResponse, error) {
	if m.createPingSessionFn == nil {
		return nil, nil
	}
	return m.createPingSessionFn(ctx, userID)
}

type mockSessionService struct {
	logoutFn        func(ctx context.Context, refreshToken string) error
	listSessionsFn  func(ctx context.Context, userID uint, currentSessionID string, page, perPage int) ([]dto.SessionInfo, int, error)
	revokeSessionFn func(ctx context.Context, userID uint, sessionID string) error
}

func (m *mockSessionService) Logout(ctx context.Context, refreshToken string) error {
	if m.logoutFn == nil {
		return nil
	}
	return m.logoutFn(ctx, refreshToken)
}

func (m *mockSessionService) ListSessions(ctx context.Context, userID uint, currentSessionID string, page, perPage int) ([]dto.SessionInfo, int, error) {
	if m.listSessionsFn == nil {
		return nil, 0, nil
	}
	return m.listSessionsFn(ctx, userID, currentSessionID, page, perPage)
}

func (m *mockSessionService) RevokeSession(ctx context.Context, userID uint, sessionID string) error {
	if m.revokeSessionFn == nil {
		return nil
	}
	return m.revokeSessionFn(ctx, userID, sessionID)
}
