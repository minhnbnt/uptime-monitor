package middleware

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	authservice "github.com/minhnbnt/uptime-monitor/internal/server/service/auth"
)

type userIDKey struct{}

func GetUserID(ctx context.Context) uint {

	v := ctx.Value(userIDKey{})
	if v == nil {
		return 0
	}

	return v.(uint)
}

type AccessTokenValidator interface {
	ValidateAccessToken(tokenStr string) (uint, error)
}

type AuthMiddleware struct {
	tokenValidator AccessTokenValidator
}

func RegisterAuthMiddleware(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthMiddleware, error) {
		return &AuthMiddleware{
			tokenValidator: do.MustInvoke[*authservice.TokenValidator](i),
		}, nil
	})
}

func (m *AuthMiddleware) HandleBearerAuth(ctx context.Context, _ api.OperationName, t api.BearerAuth) (context.Context, error) {

	userID, err := m.tokenValidator.ValidateAccessToken(t.Token)
	if err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, userIDKey{}, userID), nil
}
