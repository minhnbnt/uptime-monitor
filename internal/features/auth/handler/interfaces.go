package handler

import (
	"context"

	"github.com/minhnbnt/uptime-monitor/internal/features/auth/dto"
)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error)
	Logout(ctx context.Context, refreshToken string) error
}
