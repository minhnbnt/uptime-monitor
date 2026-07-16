package authclient

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/samber/do/v2"
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
	log *slog.Logger
}

func RegisterAuthMiddleware(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthMiddleware, error) {
		return &AuthMiddleware{
			log: do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (am *AuthMiddleware) XUserIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		uid := r.Header.Get("X-User-ID")
		uid = strings.TrimSpace(uid)

		if uid == "" {
			next.ServeHTTP(w, r)
			return
		}

		id, err := strconv.ParseUint(uid, 10, 64)
		if err != nil {
			am.log.Warn("invalid X-User-ID", slog.String("value", uid))
		} else {
			r = r.WithContext(context.WithValue(r.Context(), userIDKey{}, uint(id)))
		}

		next.ServeHTTP(w, r)
	})
}
