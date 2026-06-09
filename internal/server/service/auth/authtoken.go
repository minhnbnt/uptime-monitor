package auth

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

type tokenGenerator struct {
	provider    *jwtutil.Provider
	tokenConfig *config.TokenConfig
}

func RegisterTokenGenerator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (TokenGenerator, error) {
		return &tokenGenerator{
			provider:    do.MustInvoke[*jwtutil.Provider](i),
			tokenConfig: do.MustInvoke[*config.TokenConfig](i),
		}, nil
	})
}

func (tg *tokenGenerator) GenerateAccessToken(user *domain.User) (string, error) {

	sub := fmt.Sprint(user.ID)
	return tg.provider.NewToken(tg.tokenConfig.GetAccessTokenIssuer(), map[string]any{
		"sub":      sub,
		"email":    user.Email,
		"username": user.Username,
		"exp":      time.Now().Add(tg.tokenConfig.GetAccessTokenTTL()).Unix(),
	})
}

func (tg *tokenGenerator) GenerateRefreshToken(user *domain.User) (string, error) {

	jti, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("failed to generate jti: %v", err)
	}

	sub := fmt.Sprint(user.ID)
	return tg.provider.NewToken(tg.tokenConfig.GetRefreshTokenIssuer(), map[string]any{
		"sub": sub,
		"jti": jti.String(),
		"exp": time.Now().Add(tg.tokenConfig.GetRefreshTokenTTL()).Unix(),
	})
}
