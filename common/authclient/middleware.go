package authclient

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/google/uuid"
)

type userIDKey struct{}

func GetUserID(ctx context.Context) uuid.UUID {
	v := ctx.Value(userIDKey{})
	if v == nil {
		return uuid.Nil
	}
	return v.(uuid.UUID)
}

type AuthMiddleware struct {
	verifier *oidc.IDTokenVerifier
	log      *slog.Logger
}

func NewAuthMiddleware(ctx context.Context, issuer string, log *slog.Logger) (*AuthMiddleware, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}
	verifier := provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})
	return &AuthMiddleware{verifier: verifier, log: log}, nil
}

func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := extractBearerToken(r.Header.Get("Authorization"))
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token, err := am.verifier.Verify(r.Context(), raw)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var claims map[string]any
		if err := token.Claims(&claims); err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		sub, _ := claims["sub"].(string)
		uid, err := uuid.Parse(sub)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), userIDKey{}, uid))
		next.ServeHTTP(w, r)
	})
}

func extractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("invalid authorization header")
	}
	return parts[1], nil
}
