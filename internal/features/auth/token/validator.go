package token

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/jwt"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/repository"
)

type TokenValidator struct {
	provider         *jwt.Provider
	tokenConfig      *config.TokenConfig
	revokedTokenRepo RevokedTokenRepository
	logger           *slog.Logger
}

func RegisterTokenValidator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TokenValidator, error) {
		return NewTokenValidator(
			do.MustInvoke[*jwt.Provider](i),
			do.MustInvoke[*config.TokenConfig](i),
			do.MustInvoke[*repository.RedisRevokedTokenRepository](i),
			do.MustInvoke[*slog.Logger](i),
		), nil
	})
}

func NewTokenValidator(
	provider *jwt.Provider,
	tokenConfig *config.TokenConfig,
	revokedTokenRepo RevokedTokenRepository,
	logger *slog.Logger,
) *TokenValidator {
	return &TokenValidator{
		logger:           logger,
		provider:         provider,
		tokenConfig:      tokenConfig,
		revokedTokenRepo: revokedTokenRepo,
	}
}

func (tv *TokenValidator) ValidateAccessToken(tokenStr string) (uint, error) {

	expectedIssuer := tv.tokenConfig.GetAccessTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		tv.logger.Debug("invalid access token", slog.Any("error", err))
		return 0, apperrors.ErrInvalidAccessToken
	}

	sub, err := token.Subject()
	if err != nil {
		tv.logger.Debug("invalid access token subject", slog.Any("error", err))
		return 0, apperrors.ErrInvalidAccessToken
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		tv.logger.Debug("invalid access token subject format", slog.Any("error", err))
		return 0, apperrors.ErrInvalidAccessToken
	}

	return uint(userID), nil
}

func (tv *TokenValidator) ValidateRefreshToken(ctx context.Context, tokenStr string) (uint, string, error) {

	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		tv.logger.Debug("invalid refresh token", slog.Any("error", err))
		return 0, "", apperrors.ErrInvalidRefreshToken
	}

	jti, err := token.JTI()
	if err != nil {
		tv.logger.Debug("invalid refresh token jti", slog.Any("error", err))
		return 0, "", apperrors.ErrInvalidRefreshToken
	}

	revoked, err := tv.revokedTokenRepo.IsRevoked(ctx, jti)
	if err != nil {
		tv.logger.Debug("failed to check revoked token", slog.Any("error", err))
		return 0, "", apperrors.ErrInvalidRefreshToken
	}
	if revoked {
		return 0, "", apperrors.ErrInvalidRefreshToken
	}

	sub, err := token.Subject()
	if err != nil {
		tv.logger.Debug("invalid refresh token subject", slog.Any("error", err))
		return 0, "", apperrors.ErrInvalidRefreshToken
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		tv.logger.Debug("invalid refresh token subject format", slog.Any("error", err))
		return 0, "", apperrors.ErrInvalidRefreshToken
	}

	return uint(userID), jti, nil
}

func (tv *TokenValidator) ParseRefreshToken(tokenStr string) (*jwt.Token, error) {
	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	return tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
}

var _ RevokedTokenRepository = (*repository.RedisRevokedTokenRepository)(nil)
