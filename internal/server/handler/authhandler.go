package handler

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	authservice "github.com/minhnbnt/uptime-monitor/internal/server/service/auth"
)

type AuthHandler struct {
	authService AuthService
}

func RegisterAuthHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthHandler, error) {
		return &AuthHandler{
			authService: do.MustInvoke[*authservice.AuthService](i),
		}, nil
	})
}

func (h *AuthHandler) Register(ctx context.Context, req *api.RegisterRequest) (*api.AuthResponse, error) {

	dtoReq := dto.RegisterRequest{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
		Name:     req.Name,
	}

	result, err := h.authService.Register(ctx, dtoReq)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return toAPIAuthResponse(result), nil
}

func (h *AuthHandler) Login(ctx context.Context, req *api.LoginRequest) (*api.AuthResponse, error) {

	dtoReq := dto.LoginRequest{
		Login:    req.Login,
		Password: req.Password,
	}

	result, err := h.authService.Login(ctx, dtoReq)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return toAPIAuthResponse(result), nil
}

func (h *AuthHandler) LoginRefresh(ctx context.Context, req *api.RefreshTokenRequest) (*api.AuthResponse, error) {

	dtoReq := dto.RefreshRequest{RefreshToken: req.RefreshToken}

	result, err := h.authService.Refresh(ctx, dtoReq)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return toAPIAuthResponse(result), nil
}

func toAPIAuthResponse(result *dto.AuthResponse) *api.AuthResponse {
	return &api.AuthResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User: api.UserProfile{
			ID:       int(result.User.ID),
			Email:    result.User.Email,
			Username: result.User.Username,
			Name:     result.User.Name,
		},
	}
}

func (h *AuthHandler) Logout(ctx context.Context, req *api.RefreshTokenRequest) error {

	err := h.authService.Logout(ctx, req.RefreshToken)
	if err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}
