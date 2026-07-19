package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
)

type ForwardAuthHandler struct {
	validator *token.Validator
}

func RegisterForwardAuthHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ForwardAuthHandler, error) {
		validator := do.MustInvoke[*token.Validator](i)
		return &ForwardAuthHandler{validator: validator}, nil
	})
}

func getTokenFromHeader(auth string) (string, error) {

	if !strings.HasPrefix(auth, "Bearer ") {
		return "", errors.New("invalid token")
	}

	return strings.TrimPrefix(auth, "Bearer "), nil
}

func (h *ForwardAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	auth := r.Header.Get("Authorization")
	token, err := getTokenFromHeader(auth)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	userID, err := h.validator.ValidateAccessToken(token)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("X-User-ID", strconv.FormatUint(uint64(userID), 10))
	w.WriteHeader(http.StatusOK)
}
