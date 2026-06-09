package auth

import (
	"strconv"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

type TokenValidator struct {
	provider    *jwtutil.Provider
	tokenConfig *config.TokenConfig
}

func RegisterTokenValidator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TokenValidator, error) {
		return &TokenValidator{
			provider:    do.MustInvoke[*jwtutil.Provider](i),
			tokenConfig: do.MustInvoke[*config.TokenConfig](i),
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

func (tv *TokenValidator) ValidateRefreshToken(tokenStr string) (uint, string, error) {

	expectedIssuer := tv.tokenConfig.GetRefreshTokenIssuer()
	token, err := tv.provider.ParseWithIssuer(tokenStr, expectedIssuer)
	if err != nil {
		return 0, "", err
	}

	sub, err := token.Subject()
	if err != nil {
		return 0, "", err
	}

	userID, err := strconv.ParseUint(sub, 10, 64)
	if err != nil {
		return 0, "", err
	}

	jti, err := token.JTI()
	if err != nil {
		return 0, "", err
	}

	return uint(userID), jti, nil
}
