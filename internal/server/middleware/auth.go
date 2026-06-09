package middleware

import (
	"context"
	"net/http"

	"github.com/rs/cors"
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

type AuthMiddleware struct {
	tokenValidator *authservice.TokenValidator
}

func RegisterAuthMiddleware(i do.Injector) *AuthMiddleware {
	return &AuthMiddleware{
		tokenValidator: do.MustInvoke[*authservice.TokenValidator](i),
	}
}

func (m *AuthMiddleware) HandleBearerAuth(ctx context.Context, _ api.OperationName, t api.BearerAuth) (context.Context, error) {

	userID, err := m.tokenValidator.ValidateAccessToken(t.Token)
	if err != nil {
		return ctx, err
	}

	return context.WithValue(ctx, userIDKey{}, userID), nil
}

func CORSMiddleware() func(http.Handler) http.Handler {

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization"},
	})

	return c.Handler
}
