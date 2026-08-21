package authclient

import (
	"context"
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
}

func NewAuthMiddleware(ctx context.Context, issuer string) (*AuthMiddleware, error) {

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, err
	}

	verifier := provider.Verifier(&oidc.Config{
		SkipClientIDCheck: true,
	})

	return &AuthMiddleware{verifier: verifier}, nil
}

func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token, err := am.verifier.Verify(r.Context(), raw)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		claims := map[string]any{}
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

		ctx := context.WithValue(r.Context(), userIDKey{}, uid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
