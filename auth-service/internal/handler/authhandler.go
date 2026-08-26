package handler

import (
	"context"
	"errors"
	"slices"

	"github.com/google/uuid"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/service"
)

type AccessTokenValidator interface { // ponytail: used by forwardauth.go
	ValidateAccessToken(tokenStr string) (*token.AccessTokenInfo, error)
}

type AuthHandler struct {
	authService    AuthService
	sessionService SessionService
	tokenValidator AccessTokenValidator
}

func RegisterAuthHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthHandler, error) {
		return &AuthHandler{
			authService:    do.MustInvoke[*service.AuthService](i),
			sessionService: do.MustInvoke[*service.SessionService](i),
			tokenValidator: do.MustInvoke[*token.Validator](i),
		}, nil
	})
}

var _ AuthService = (*service.AuthService)(nil)
var _ SessionService = (*service.SessionService)(nil)

type AuthService interface {
	Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error)
	Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error)
	Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error)
	GetUser(ctx context.Context, id uint) (*dto.UserProfile, error)
	CreatePingSession(ctx context.Context, userID uint) (*dto.AuthResponse, error)
}

type SessionService interface {
	Logout(ctx context.Context, refreshToken string) error
	ListSessions(ctx context.Context, userID uint, currentSessionID string, page, perPage int) ([]dto.SessionInfo, int, error)
	RevokeSession(ctx context.Context, userID uint, sessionID string) error
}

func appScopedInfo(ctx context.Context) (*token.AccessTokenInfo, error) {

	info, ok := tokenInfoFromContext(ctx)
	if !ok {
		return nil, apperrors.ErrInvalidAccessToken
	}

	if !slices.Contains(info.Scopes, string(domain.ScopeApp)) {
		return nil, apperrors.ErrForbidden
	}

	return info, nil
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

	err := h.sessionService.Logout(ctx, req.RefreshToken)
	if err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}

func (h *AuthHandler) CreatePingSession(ctx context.Context) (*api.AuthResponse, error) {

	info, err := appScopedInfo(ctx)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	result, err := h.authService.CreatePingSession(ctx, info.UserID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return toAPIAuthResponse(result), nil
}

func (h *AuthHandler) ListSessions(ctx context.Context, params api.ListSessionsParams) (*api.SessionListResponse, error) {

	info, err := appScopedInfo(ctx)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	page := 1
	if params.Page.IsSet() {
		page = params.Page.Value
	}

	perPage := 20
	if params.PerPage.IsSet() {
		perPage = params.PerPage.Value
	}

	items, total, err := h.sessionService.ListSessions(ctx, info.UserID, info.SID, page, perPage)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	data := make([]api.SessionInfo, 0, len(items))
	for _, item := range items {

		id, parseErr := uuid.Parse(item.ID)
		if parseErr != nil {
			return nil, apperrors.ToAPIError(apperrors.ErrInternal)
		}

		data = append(data, api.SessionInfo{
			ID:        id,
			Scopes:    item.Scopes,
			Current:   item.Current,
			CreatedAt: item.CreatedAt,
			ExpiresAt: item.ExpiresAt,
		})
	}

	return &api.SessionListResponse{
		Data: data,
		Meta: api.PaginationMeta{
			Page:    api.NewOptInt(page),
			PerPage: api.NewOptInt(perPage),
			Total:   api.NewOptInt(total),
		},
	}, nil
}

func (h *AuthHandler) RevokeSession(ctx context.Context, params api.RevokeSessionParams) (api.RevokeSessionRes, error) {

	info, err := appScopedInfo(ctx)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	err = h.sessionService.RevokeSession(ctx, info.UserID, params.SessionId.String())
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.RevokeSessionNoContent{}, nil
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

	info, ok := tokenInfoFromContext(ctx)
	if !ok {
		return nil, apperrors.ToAPIError(apperrors.ErrInvalidAccessToken)
	}

	return &api.ValidateTokenOK{UserID: int(info.UserID)}, nil
}

func (h *AuthHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {

	if errors.Is(err, ogenerrors.ErrSecurityRequirementIsNotSatisfied) {
		err = apperrors.ErrInvalidAccessToken
	}

	return apperrors.ToAPIError(err)
}
