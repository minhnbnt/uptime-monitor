package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/dto"
)

func TestAuthHandler_Register(t *testing.T) {
	validUser := dto.UserProfile{ID: 1, Email: "a@b.com", Username: "user1", Name: "Test"}

	t.Run("success", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				registerFn: func(_ context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
					return &dto.AuthResponse{AccessToken: "jwt", User: validUser}, nil
				},
			},
		}

		req := &api.RegisterRequest{Email: "a@b.com", Username: "user1", Password: "pass1234", Name: "Test"}
		resp, err := h.Register(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.AccessToken != "jwt" || resp.User.Email != "a@b.com" {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("email taken", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				registerFn: func(_ context.Context, _ dto.RegisterRequest) (*dto.AuthResponse, error) {
					return nil, apperrors.ErrEmailOrUsernameTaken
				},
			},
		}

		req := &api.RegisterRequest{Email: "a@b.com", Username: "user1", Password: "pass1234", Name: "Test"}
		_, err := h.Register(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusConflict {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusConflict)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				registerFn: func(_ context.Context, _ dto.RegisterRequest) (*dto.AuthResponse, error) {
					return nil, errors.New("db error")
				},
			},
		}

		req := &api.RegisterRequest{Email: "a@b.com", Username: "user1", Password: "pass1234", Name: "Test"}
		_, err := h.Register(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}

func TestAuthHandler_LoginRefresh(t *testing.T) {
	validUser := dto.UserProfile{ID: 1, Email: "a@b.com", Username: "user1", Name: "Test"}

	t.Run("success", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				refreshFn: func(_ context.Context, _ dto.RefreshRequest) (*dto.AuthResponse, error) {
					return &dto.AuthResponse{AccessToken: "new-jwt", RefreshToken: "new-refresh", User: validUser}, nil
				},
			},
		}

		req := &api.RefreshTokenRequest{RefreshToken: "valid-refresh-token"}
		resp, err := h.LoginRefresh(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.AccessToken != "new-jwt" || resp.RefreshToken != "new-refresh" {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				refreshFn: func(_ context.Context, _ dto.RefreshRequest) (*dto.AuthResponse, error) {
					return nil, apperrors.ErrInvalidRefreshToken
				},
			},
		}

		req := &api.RefreshTokenRequest{RefreshToken: "expired-token"}
		_, err := h.LoginRefresh(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusUnauthorized)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				refreshFn: func(_ context.Context, _ dto.RefreshRequest) (*dto.AuthResponse, error) {
					return nil, errors.New("unexpected")
				},
			},
		}

		req := &api.RefreshTokenRequest{RefreshToken: "some-token"}
		_, err := h.LoginRefresh(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}

func TestAuthHandler_Logout(t *testing.T) {

	t.Run("success", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				logoutFn: func(_ context.Context, _ string) error {
					return nil
				},
			},
		}

		req := &api.RefreshTokenRequest{RefreshToken: "some-token"}
		err := h.Logout(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				logoutFn: func(_ context.Context, _ string) error {
					return apperrors.ErrInvalidRefreshToken
				},
			},
		}

		req := &api.RefreshTokenRequest{RefreshToken: "invalid-token"}
		err := h.Logout(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusUnauthorized)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				logoutFn: func(_ context.Context, _ string) error {
					return errors.New("unexpected")
				},
			},
		}

		req := &api.RefreshTokenRequest{RefreshToken: "some-token"}
		err := h.Logout(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}

func TestAuthHandler_Login(t *testing.T) {
	validUser := dto.UserProfile{ID: 1, Email: "a@b.com", Username: "user1", Name: "Test"}

	t.Run("success", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				loginFn: func(_ context.Context, _ dto.LoginRequest) (*dto.AuthResponse, error) {
					return &dto.AuthResponse{AccessToken: "jwt", User: validUser}, nil
				},
			},
		}

		req := &api.LoginRequest{Login: "a@b.com", Password: "pass1234"}
		resp, err := h.Login(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.AccessToken != "jwt" {
			t.Errorf("access_token = %q", resp.AccessToken)
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				loginFn: func(_ context.Context, _ dto.LoginRequest) (*dto.AuthResponse, error) {
					return nil, apperrors.ErrInvalidCredentials
				},
			},
		}

		req := &api.LoginRequest{Login: "a@b.com", Password: "wrong"}
		_, err := h.Login(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusUnauthorized)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				loginFn: func(_ context.Context, _ dto.LoginRequest) (*dto.AuthResponse, error) {
					return nil, errors.New("db error")
				},
			},
		}

		req := &api.LoginRequest{Login: "a@b.com", Password: "pass1234"}
		_, err := h.Login(context.Background(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}
