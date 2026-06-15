package auth

import (
	"context"
	"errors"
	"strconv"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

type TokenValidator struct {
	provider         *jwtutil.Provider
	tokenConfig      *config.TokenConfig
	revokedTokenRepo RevokedTokenRepository
	logger           logger.Logger
}

func RegisterTokenValidator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TokenValidator, error) {
		return &TokenValidator{
			provider:         do.MustInvoke[*jwtutil.Provider](i),
			tokenConfig:      do.MustInvoke[*config.TokenConfig](i),
			revokedTokenRepo: do.MustInvoke[RevokedTokenRepository](i),
			logger:           do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (tv *TokenValidator) ValidateAccessToken(tokenStr string) (uint, error) {

	expectedIssuer := tv.tokenConfig.GetAccessTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		tv.logger.Debug("invalid access token", logger.Error(err))
		return 0, errors.New("invalid access token")
	}

	sub, err := token.Subject()
	if err != nil {
		tv.logger.Debug("invalid access token subject", logger.Error(err))
		return 0, errors.New("invalid access token")
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		tv.logger.Debug("invalid access token subject format", logger.Error(err))
		return 0, errors.New("invalid access token")
	}

	return uint(userID), nil
}

func (tv *TokenValidator) ValidateRefreshToken(ctx context.Context, tokenStr string) (uint, string, error) {

	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		tv.logger.Debug("invalid refresh token", logger.Error(err))
		return 0, "", errors.New("invalid refresh token")
	}

	jti, err := token.JTI()
	if err != nil {
		tv.logger.Debug("invalid refresh token jti", logger.Error(err))
		return 0, "", errors.New("invalid refresh token")
	}

	revoked, err := tv.revokedTokenRepo.IsRevoked(ctx, jti)
	if err != nil {
		tv.logger.Debug("failed to check revoked token", logger.Error(err))
		return 0, "", errors.New("invalid refresh token")
	}
	if revoked {
		return 0, "", apperrors.ErrInvalidCredentials
	}

	sub, err := token.Subject()
	if err != nil {
		tv.logger.Debug("invalid refresh token subject", logger.Error(err))
		return 0, "", errors.New("invalid refresh token")
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		tv.logger.Debug("invalid refresh token subject format", logger.Error(err))
		return 0, "", errors.New("invalid refresh token")
	}

	return uint(userID), jti, nil
}

func (tv *TokenValidator) ParseRefreshToken(tokenStr string) (*jwtutil.Token, error) {
	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	return tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
}
