package authclient

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type userIDKey struct{}
type scopesKey struct{}

func GetUserID(ctx context.Context) uint {
	v := ctx.Value(userIDKey{})
	if v == nil {
		return 0
	}
	return v.(uint)
}

func GetScopes(ctx context.Context) []string {
	v := ctx.Value(scopesKey{})
	if v == nil {
		return nil
	}
	return v.([]string)
}

func HasScope(ctx context.Context, scope string) bool {
	scopes := GetScopes(ctx)
	return slices.Contains(scopes, scope)
}

type AuthMiddleware struct {
	log *slog.Logger
}

func NewAuthMiddleware(log *slog.Logger) *AuthMiddleware {
	return &AuthMiddleware{log: log}
}

func (am *AuthMiddleware) XUserIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		uid := strings.TrimSpace(r.Header.Get("X-User-ID"))

		if uid == "" {
			next.ServeHTTP(w, r)
		}

		id, err := strconv.ParseUint(uid, 10, 64)
		if err != nil {
			am.log.Warn("invalid X-User-ID", slog.String("value", uid))
		} else {

			ctx := context.WithValue(r.Context(), userIDKey{}, uint(id))
			scopes := strings.Fields(r.Header.Get("X-Scopes"))
			if len(scopes) > 0 {
				ctx = context.WithValue(ctx, scopesKey{}, scopes)
			}

			r = r.WithContext(ctx)
		}

		next.ServeHTTP(w, r)
	})
}

func (am *AuthMiddleware) RequireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !HasScope(r.Context(), scope) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
