package token

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/jwt"
)

type tokenGenerator struct {
	provider    *jwt.Provider
	tokenConfig *config.TokenConfig
}

func RegisterGenerator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (Generator, error) {
		return &tokenGenerator{
			provider:    do.MustInvoke[*jwt.Provider](i),
			tokenConfig: do.MustInvoke[*config.TokenConfig](i),
		}, nil
	})
}

func (tg *tokenGenerator) GenerateAccessToken(user *domain.User, scopes []string, sessionID string) (string, error) {

	sub := fmt.Sprint(user.ID)
	issuer := tg.tokenConfig.GetAccessTokenIssuer()
	claims := map[string]any{
		"sub":      sub,
		"email":    user.Email,
		"username": user.Username,
		"scope":    strings.Join(scopes, " "),
		"sid":      sessionID,
		"exp": time.Now().
			Add(tg.tokenConfig.GetAccessTokenTTL()).
			Unix(),
	}

	token, err := tg.provider.NewToken(issuer, claims)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (tg *tokenGenerator) GenerateRefreshToken(user *domain.User) (string, string, error) {

	jti, err := uuid.NewRandom()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate jti: %v", err)
	}

	sub := fmt.Sprint(user.ID)
	issuer := tg.tokenConfig.GetRefreshTokenIssuer()
	claims := map[string]any{
		"sub": sub,
		"jti": jti.String(),
		"exp": time.Now().Add(tg.tokenConfig.GetRefreshTokenTTL()).Unix(),
	}

	token, err := tg.provider.NewToken(issuer, claims)
	if err != nil {
		return "", "", err
	}

	return token, jti.String(), nil
}
