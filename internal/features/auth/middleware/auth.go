package middleware

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/token"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
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
	logger         logger.Logger
}

func RegisterAuthMiddleware(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthMiddleware, error) {
		return &AuthMiddleware{
			tokenValidator: do.MustInvoke[*token.TokenValidator](i),
			logger:         do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (m *AuthMiddleware) HandleBearerAuth(ctx context.Context, _ api.OperationName, t api.BearerAuth) (context.Context, error) {

	userID, err := m.tokenValidator.ValidateAccessToken(t.Token)
	if err != nil {
		m.logger.Debug("bearer auth failed", logger.Error(err))
		return ctx, apperrors.ErrInvalidAccessToken
	}

	return context.WithValue(ctx, userIDKey{}, userID), nil
}
