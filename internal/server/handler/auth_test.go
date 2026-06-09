package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	authservice "github.com/minhnbnt/uptime-monitor/internal/server/service/auth"
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
					return nil, authservice.ErrEmailOrUsernameTaken
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
					return nil, authservice.ErrInvalidCredentials
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
