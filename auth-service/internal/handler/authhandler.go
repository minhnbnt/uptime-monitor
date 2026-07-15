package handler

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/service"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
)

type userIDKey struct{}

type AccessTokenValidator interface { // ponytail: used by forwardauth.go
	ValidateAccessToken(tokenStr string) (uint, error)
}

type AuthHandler struct {
	authService    AuthService
	tokenValidator AccessTokenValidator
}

func RegisterAuthHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthHandler, error) {
		return &AuthHandler{
			authService:    do.MustInvoke[*service.AuthService](i),
			tokenValidator: do.MustInvoke[*token.TokenValidator](i),
		}, nil
	})
}

var _ AuthService = (*service.AuthService)(nil)

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

func (h *AuthHandler) HandleBearerAuth(ctx context.Context, _ api.OperationName, t api.BearerAuth) (context.Context, error) {

	userID, err := h.tokenValidator.ValidateAccessToken(t.Token)
	if err != nil {
		return ctx, apperrors.ErrInvalidAccessToken
	}

	return context.WithValue(ctx, userIDKey{}, userID), nil
}

func (h *AuthHandler) GetUser(ctx context.Context, params api.GetUserParams) (*api.UserProfile, error) {

	user, err := h.authService.GetUser(ctx, uint(params.ID))
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.UserProfile{
		ID:       int(user.ID),
		Email:    user.Email,
		Username: user.Username,
		Name:     user.Name,
	}, nil
}

func (h *AuthHandler) ValidateToken(ctx context.Context) (*api.ValidateTokenOK, error) {

	userID, ok := ctx.Value(userIDKey{}).(uint)
	if !ok || userID == 0 {
		return nil, apperrors.ToAPIError(apperrors.ErrInvalidAccessToken)
	}

	return &api.ValidateTokenOK{
		UserID: int(userID),
	}, nil
}

func (h *AuthHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	return apperrors.ToAPIError(err)
}
