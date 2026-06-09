package auth

import (
	"context"
	"strconv"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

type TokenValidator struct {
	provider         *jwtutil.Provider
	tokenConfig      *config.TokenConfig
	revokedTokenRepo RevokedTokenRepository
}

func RegisterTokenValidator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TokenValidator, error) {
		return &TokenValidator{
			provider:         do.MustInvoke[*jwtutil.Provider](i),
			tokenConfig:      do.MustInvoke[*config.TokenConfig](i),
			revokedTokenRepo: do.MustInvoke[RevokedTokenRepository](i),
		}, nil
	})
}

func (tv *TokenValidator) ValidateAccessToken(tokenStr string) (uint, error) {

	expectedIssuer := tv.tokenConfig.GetAccessTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		return 0, err
	}

	sub, err := token.Subject()
	if err != nil {
		return 0, err
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		return 0, err
	}

	return uint(userID), nil
}

func (tv *TokenValidator) ValidateRefreshToken(ctx context.Context, tokenStr string) (uint, string, error) {

	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		return 0, "", err
	}

	jti, err := token.JTI()
	if err != nil {
		return 0, "", err
	}

	revoked, err := tv.revokedTokenRepo.IsRevoked(ctx, jti)
	if err != nil {
		return 0, "", err
	}
	if revoked {
		return 0, "", ErrInvalidCredentials
	}

	sub, err := token.Subject()
	if err != nil {
		return 0, "", err
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		return 0, "", err
	}

	return uint(userID), jti, nil
}

func (tv *TokenValidator) ParseRefreshToken(tokenStr string) (*jwtutil.Token, error) {
	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	return tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
}
