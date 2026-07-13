package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/token"
)

func NewForwardAuthHandler(v *token.TokenValidator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		userID, err := v.ValidateAccessToken(strings.TrimPrefix(auth, "Bearer "))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))
		w.WriteHeader(http.StatusOK)
	})
}
