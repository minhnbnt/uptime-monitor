package authclient

import (
	"context"
	"net/http"
	"strconv"
	"strings"
)

type userIDKey struct{}

func GetUserID(ctx context.Context) uint {
	v := ctx.Value(userIDKey{})
	if v == nil {
		return 0
	}
	return v.(uint)
}

func XUserIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid := r.Header.Get("X-User-ID")
		uid = strings.TrimSpace(uid)

		if uid == "" {
			next.ServeHTTP(w, r)
			return
		}

		id, err := strconv.ParseUint(uid, 10, 64)
		if err == nil {
			r = r.WithContext(context.WithValue(r.Context(), userIDKey{}, uint(id)))
		}

		next.ServeHTTP(w, r)
	})
}
