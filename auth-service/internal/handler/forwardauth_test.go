package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
)

type fakeTokenValidator struct {
	byToken map[string]*token.AccessTokenInfo
}

func (f *fakeTokenValidator) ValidateAccessToken(tokenStr string) (*token.AccessTokenInfo, error) {
	info, ok := f.byToken[tokenStr]
	if !ok {
		return nil, apperrors.ErrInvalidAccessToken
	}
	return info, nil
}

func newFakeTokenValidator() *fakeTokenValidator {
	return &fakeTokenValidator{byToken: map[string]*token.AccessTokenInfo{
		"app-token":  {UserID: 42, Scopes: []string{"app"}},
		"ping-token": {UserID: 42, Scopes: []string{"ping"}},
	}}
}

func TestCreatePingSession_FullServer(t *testing.T) {

	newServer := func(t *testing.T, fn func(ctx context.Context, userID uint) (*dto.AuthResponse, error)) *api.Server {
		t.Helper()

		h := &AuthHandler{
			authService: &mockAuthService{createPingSessionFn: fn},
		}
		fw := &ForwardAuthHandler{validator: newFakeTokenValidator()}

		srv, err := api.NewServer(h, fw)
		if err != nil {
			t.Fatalf("NewServer: %v", err)
		}
		return srv
	}

	call := func(t *testing.T, srv *api.Server, bearer string) int {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/api/v1/auth/sessions/ping", nil)
		if bearer != "" {
			req.Header.Set("Authorization", "Bearer "+bearer)
		}
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		return rec.Code
	}

	t.Run("with app scope", func(t *testing.T) {
		var gotUserID uint
		srv := newServer(t, func(_ context.Context, userID uint) (*dto.AuthResponse, error) {
			gotUserID = userID
			return &dto.AuthResponse{
				AccessToken:  "access",
				RefreshToken: "refresh",
				User:         dto.UserProfile{ID: 42},
			}, nil
		})

		if code := call(t, srv, "app-token"); code != http.StatusOK {
			t.Fatalf("status = %d, want 200", code)
		}
		if gotUserID != 42 {
			t.Errorf("userID = %d, want 42", gotUserID)
		}
	})

	t.Run("without app scope", func(t *testing.T) {
		called := false
		srv := newServer(t, func(_ context.Context, _ uint) (*dto.AuthResponse, error) {
			called = true
			return nil, nil
		})

		if code := call(t, srv, "ping-token"); code != http.StatusForbidden {
			t.Fatalf("status = %d, want 403", code)
		}
		if called {
			t.Error("service must not be called")
		}
	})

	t.Run("without token", func(t *testing.T) {
		srv := newServer(t, func(_ context.Context, _ uint) (*dto.AuthResponse, error) {
			return nil, nil
		})

		if code := call(t, srv, ""); code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", code)
		}
	})

	t.Run("with invalid token", func(t *testing.T) {
		srv := newServer(t, func(_ context.Context, _ uint) (*dto.AuthResponse, error) {
			return nil, nil
		})

		if code := call(t, srv, "bogus"); code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", code)
		}
	})
}

func TestForwardAuth(t *testing.T) {

	t.Run("ok", func(t *testing.T) {
		fw := &ForwardAuthHandler{validator: newFakeTokenValidator()}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer app-token")
		rec := httptest.NewRecorder()
		fw.ForwardAuth(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if got := rec.Header().Get("X-User-ID"); got != "42" {
			t.Errorf("X-User-ID = %q, want 42", got)
		}
		if got := rec.Header().Get("X-Scopes"); got != "app" {
			t.Errorf("X-Scopes = %q, want app", got)
		}
	})

	t.Run("missing authorization header", func(t *testing.T) {
		fw := &ForwardAuthHandler{validator: newFakeTokenValidator()}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/verify", nil)
		rec := httptest.NewRecorder()
		fw.ForwardAuth(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		fw := &ForwardAuthHandler{validator: newFakeTokenValidator()}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Basic abc")
		rec := httptest.NewRecorder()
		fw.ForwardAuth(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		fw := &ForwardAuthHandler{validator: newFakeTokenValidator()}

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/auth/verify", nil)
		req.Header.Set("Authorization", "Bearer nope")
		rec := httptest.NewRecorder()
		fw.ForwardAuth(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})
}
