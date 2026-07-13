package authclient

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
)

type userIDKey struct{}

func GetUserID(ctx context.Context) uint {
	v := ctx.Value(userIDKey{})
	if v == nil {
		return 0
	}
	return v.(uint)
}

type AuthMiddleware struct{}

func RegisterAuthMiddleware(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthMiddleware, error) {
		return &AuthMiddleware{}, nil
	})
}

func (m *AuthMiddleware) HandleBearerAuth(ctx context.Context, _ api.OperationName, t api.BearerAuth) (context.Context, error) {
	return ctx, nil
}

func XUserIDMiddleware(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid := r.Header.Get("X-User-ID")
			if uid != "" {
				id, err := strconv.ParseUint(strings.TrimSpace(uid), 10, 64)
				if err != nil {
					log.Warn("invalid X-User-ID", slog.String("value", uid))
				} else {
					r = r.WithContext(context.WithValue(r.Context(), userIDKey{}, uint(id)))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
