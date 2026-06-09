package service

import (
	"strconv"

	"github.com/samber/do/v2"

	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

type TokenValidator struct {
	provider *jwtutil.Provider
}

func RegisterTokenValidator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TokenValidator, error) {
		return &TokenValidator{
			provider: do.MustInvoke[*jwtutil.Provider](i),
		}, nil
	})
}

func (tv *TokenValidator) ValidateUserToken(tokenStr string) (uint, error) {

	token, err := tv.provider.Parse(tokenStr)
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
