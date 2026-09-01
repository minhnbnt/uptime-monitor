package token

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/jwt"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/repository"
)

type Validator struct {
	provider    *jwt.Provider
	tokenConfig *config.TokenConfig
	sessionRepo SessionRepository
	logger      *slog.Logger
}

func RegisterValidator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Validator, error) {
		return NewValidator(
			do.MustInvoke[*jwt.Provider](i),
			do.MustInvoke[*config.TokenConfig](i),
			do.MustInvoke[*repository.SessionRepository](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func NewValidator(
	provider *jwt.Provider,
	tokenConfig *config.TokenConfig,
	sessionRepo SessionRepository,
	logger *slog.Logger,
) *Validator {
	return &Validator{
		logger:      logger,
		provider:    provider,
		tokenConfig: tokenConfig,
		sessionRepo: sessionRepo,
	}
}

type AccessTokenInfo struct {
	UserID uint
	Scopes []string
	SID    string
}

type RefreshTokenInfo struct {
	UserID  uint
	JTI     string
	Scopes  []string
	Counter int64
}

func (tv *Validator) ValidateAccessToken(tokenStr string) (*AccessTokenInfo, error) {

	expectedIssuer := tv.tokenConfig.GetAccessTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		tv.logger.Debug("invalid access token", slog.Any("error", err))
		return nil, apperrors.ErrInvalidAccessToken
	}

	sub, err := token.Subject()
	if err != nil {
		tv.logger.Debug("invalid access token subject", slog.Any("error", err))
		return nil, apperrors.ErrInvalidAccessToken
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		tv.logger.Debug("invalid access token subject format", slog.Any("error", err))
		return nil, apperrors.ErrInvalidAccessToken
	}

	var sid string
	var scopes []string
	if claims, err := token.Claims(); err == nil {

		if scopeStr, ok := claims["scope"].(string); ok {
			scopes = strings.Fields(scopeStr)
		}

		if sidClaim, ok := claims["sid"].(string); ok {
			sid = sidClaim
		}
	}

	return &AccessTokenInfo{UserID: uint(userID), Scopes: scopes, SID: sid}, nil
}

func (tv *Validator) ValidateRefreshToken(ctx context.Context, tokenStr string) (*RefreshTokenInfo, error) {

	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		tv.logger.Debug("invalid refresh token", slog.Any("error", err))
		return nil, apperrors.ErrInvalidRefreshToken
	}

	jti, err := token.JTI()
	if err != nil {
		tv.logger.Debug("invalid refresh token jti", slog.Any("error", err))
		return nil, apperrors.ErrInvalidRefreshToken
	}

	session, err := tv.sessionRepo.GetByJTI(ctx, jti)
	if err != nil {
		tv.logger.Debug("failed to get session", slog.Any("error", err))
		return nil, apperrors.ErrInvalidRefreshToken
	}

	if session == nil || session.ExpiresAt.Before(time.Now()) {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	sub, err := token.Subject()
	if err != nil {
		tv.logger.Debug("invalid refresh token subject", slog.Any("error", err))
		return nil, apperrors.ErrInvalidRefreshToken
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		tv.logger.Debug("invalid refresh token subject format", slog.Any("error", err))
		return nil, apperrors.ErrInvalidRefreshToken
	}

	var counter int64
	if claims, err := token.Claims(); err == nil {
		if c, ok := claims["counter"].(float64); ok {
			counter = int64(c)
		}
	}

	return &RefreshTokenInfo{UserID: uint(userID), JTI: jti, Scopes: strings.Fields(session.Scopes), Counter: counter}, nil
}

func (tv *Validator) ParseRefreshToken(tokenStr string) (*jwt.Token, error) {
	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	return tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
}
