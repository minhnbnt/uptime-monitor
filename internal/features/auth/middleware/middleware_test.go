package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
)

func TestGetUserID(t *testing.T) {
	t.Run("returns 0 when no userID in context", func(t *testing.T) {
		if got := GetUserID(context.Background()); got != 0 {
			t.Errorf("got %d, want 0", got)
		}
	})

	t.Run("returns userID when present", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), userIDKey{}, uint(42))
		if got := GetUserID(ctx); got != 42 {
			t.Errorf("got %d, want 42", got)
		}
	})
}

type mockTokenValidator struct {
	validateAccessTokenFn func(tokenStr string) (uint, error)
}

func (m *mockTokenValidator) ValidateAccessToken(tokenStr string) (uint, error) {
	return m.validateAccessTokenFn(tokenStr)
}

func TestAuthMiddleware_HandleBearerAuth(t *testing.T) {
	t.Run("valid token returns context with userID", func(t *testing.T) {
		m := &AuthMiddleware{
			tokenValidator: &mockTokenValidator{
				validateAccessTokenFn: func(_ string) (uint, error) {
					return 42, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		ctx, err := m.HandleBearerAuth(context.Background(), "", api.BearerAuth{Token: "valid-token"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := GetUserID(ctx); got != 42 {
			t.Errorf("userID = %d, want 42", got)
		}
	})

	t.Run("invalid token returns ErrInvalidAccessToken", func(t *testing.T) {
		m := &AuthMiddleware{
			tokenValidator: &mockTokenValidator{
				validateAccessTokenFn: func(_ string) (uint, error) {
					return 0, apperrors.ErrInvalidAccessToken
				},
			},
			logger: logger.NewMockLogger(),
		}

		_, err := m.HandleBearerAuth(context.Background(), "", api.BearerAuth{Token: "bad-token"})
		if !errors.Is(err, apperrors.ErrInvalidAccessToken) {
			t.Errorf("got %v, want %v", err, apperrors.ErrInvalidAccessToken)
		}
	})
}
