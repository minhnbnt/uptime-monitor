package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
)

type tokenInfoKey struct{}

func tokenInfoFromContext(ctx context.Context) (*token.AccessTokenInfo, bool) {
	info, ok := ctx.Value(tokenInfoKey{}).(*token.AccessTokenInfo)
	return info, ok
}

type ForwardAuthHandler struct {
	validator AccessTokenValidator
}

func RegisterForwardAuthHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ForwardAuthHandler, error) {
		validator := do.MustInvoke[*token.Validator](i)
		return &ForwardAuthHandler{validator: validator}, nil
	})
}

func (h *ForwardAuthHandler) HandleBearerAuth(ctx context.Context, _ api.OperationName, t api.BearerAuth) (context.Context, error) {

	info, err := h.validator.ValidateAccessToken(t.Token)
	if err != nil {
		return ctx, apperrors.ErrInvalidAccessToken
	}

	return context.WithValue(ctx, tokenInfoKey{}, info), nil
}

func (h *ForwardAuthHandler) GetTokenInfo(ctx context.Context) (*token.AccessTokenInfo, error) {

	info, ok := tokenInfoFromContext(ctx)
	if !ok {
		return nil, apperrors.ErrForbidden
	}

	return info, nil
}

func getTokenFromHeader(auth string) (string, error) {

	if !strings.HasPrefix(auth, "Bearer ") {
		return "", errors.New("invalid authorization header")
	}

	return strings.TrimPrefix(auth, "Bearer "), nil
}

func (h *ForwardAuthHandler) ForwardAuth(w http.ResponseWriter, r *http.Request) {

	tokenStr, err := getTokenFromHeader(r.Header.Get("Authorization"))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	info, err := h.validator.ValidateAccessToken(tokenStr)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	w.Header().Set("X-User-ID", fmt.Sprint(info.UserID))
	w.Header().Set("X-Scopes", strings.Join(info.Scopes, " "))
	w.Header().Set("X-Session-ID", info.SID)

	w.WriteHeader(http.StatusOK)
}
