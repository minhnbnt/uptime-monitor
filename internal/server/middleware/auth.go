package middleware

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
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

type AuthMiddleware struct {
	tokenValidator *authservice.TokenValidator
	logger         logger.Logger
}

func RegisterAuthMiddleware(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthMiddleware, error) {
		return &AuthMiddleware{
			tokenValidator: do.MustInvoke[*authservice.TokenValidator](i),
			logger:         do.MustInvoke[logger.Logger](i),
		}, nil
	})
}

func (m *AuthMiddleware) HandleBearerAuth(ctx context.Context, _ api.OperationName, t api.BearerAuth) (context.Context, error) {

	userID, err := m.tokenValidator.ValidateAccessToken(t.Token)
	if err != nil {
		m.logger.Debug("bearer auth failed", logger.Error(err))
		return ctx, apperrors.ErrInvalidCredentials
	}

	return context.WithValue(ctx, userIDKey{}, userID), nil
}
