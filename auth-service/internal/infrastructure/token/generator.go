package token

import (
	"fmt"
	"sort"
	"strings"
	"time"

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

	sort.Strings(scopes)

	sub := fmt.Sprint(user.ID)
	issuer := tg.tokenConfig.GetAccessTokenIssuer()
	accessTokenTTL := tg.tokenConfig.GetAccessTokenTTL()
	claims := map[string]any{
		"sub":      sub,
		"email":    user.Email,
		"username": user.Username,
		"scope":    strings.Join(scopes, " "),
		"sid":      sessionID,
		"exp":      time.Now().Add(accessTokenTTL).Unix(),
	}

	token, err := tg.provider.NewToken(issuer, claims)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (tg *tokenGenerator) GenerateRefreshToken(user *domain.User, jti string, counter int64) (string, error) {

	sub := fmt.Sprint(user.ID)
	issuer := tg.tokenConfig.GetRefreshTokenIssuer()
	refreshTokenTTL := tg.tokenConfig.GetRefreshTokenTTL()
	claims := map[string]any{
		"sub":     sub,
		"jti":     jti,
		"counter": counter,
		"exp":     time.Now().Add(refreshTokenTTL).Unix(),
	}

	token, err := tg.provider.NewToken(issuer, claims)
	if err != nil {
		return "", err
	}

	return token, nil
}
