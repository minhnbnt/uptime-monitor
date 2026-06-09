package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	authservice "github.com/minhnbnt/uptime-monitor/internal/server/service/auth"
)

func TestAuthHandler_Register(t *testing.T) {
	validator := &RequestValidator{v: validator.New()}
	validBody := `{"email":"a@b.com","username":"user1","password":"pass1234","name":"Test"}`
	validUser := dto.UserProfile{ID: 1, Email: "a@b.com", Username: "user1", Name: "Test"}

	t.Run("success", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				registerFn: func(_ context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {
					return &dto.AuthResponse{AccessToken: "jwt", User: validUser}, nil
				},
			},
			validator: validator,
		}

		c, w := newGinContext("POST", "/api/v1/auth/register", validBody)
		h.Register(c)

		if w.Code != http.StatusCreated {
			t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
		}
		var resp api.AuthResponse
		parseJSON(w, &resp)
		if resp.AccessToken != "jwt" || resp.User.Email != "a@b.com" {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		h := &AuthHandler{validator: validator}
		c, w := newGinContext("POST", "/api/v1/auth/register", `{"bad`)
		h.Register(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("validation error", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{},
			validator:   validator,
		}
		c, w := newGinContext("POST", "/api/v1/auth/register",
			`{"email":"a@b.com","username":"u","password":"short","name":""}`)
		h.Register(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
		var errResp api.ErrorResponse
		parseJSON(w, &errResp)
		if errResp.Error.Code != "VALIDATION_ERROR" {
			t.Errorf("code = %s", errResp.Error.Code)
		}
	})

	t.Run("email taken", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				registerFn: func(_ context.Context, _ dto.RegisterRequest) (*dto.AuthResponse, error) {
					return nil, authservice.ErrEmailOrUsernameTaken
				},
			},
			validator: validator,
		}
		c, w := newGinContext("POST", "/api/v1/auth/register", validBody)
		h.Register(c)

		if w.Code != http.StatusConflict {
			t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				registerFn: func(_ context.Context, _ dto.RegisterRequest) (*dto.AuthResponse, error) {
					return nil, errors.New("db error")
				},
			},
			validator: validator,
		}
		c, w := newGinContext("POST", "/api/v1/auth/register", validBody)
		h.Register(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}

func TestAuthHandler_Login(t *testing.T) {
	validator := &RequestValidator{v: validator.New()}
	validBody := `{"login":"a@b.com","password":"pass1234"}`
	validUser := dto.UserProfile{ID: 1, Email: "a@b.com", Username: "user1", Name: "Test"}

	t.Run("success", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				loginFn: func(_ context.Context, _ dto.LoginRequest) (*dto.AuthResponse, error) {
					return &dto.AuthResponse{AccessToken: "jwt", User: validUser}, nil
				},
			},
			validator: validator,
		}
		c, w := newGinContext("POST", "/api/v1/auth/login", validBody)
		h.Login(c)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
		var resp api.AuthResponse
		parseJSON(w, &resp)
		if resp.AccessToken != "jwt" {
			t.Errorf("access_token = %q", resp.AccessToken)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		h := &AuthHandler{validator: validator}
		c, w := newGinContext("POST", "/api/v1/auth/login", "{bad}")
		h.Login(c)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("invalid credentials", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				loginFn: func(_ context.Context, _ dto.LoginRequest) (*dto.AuthResponse, error) {
					return nil, authservice.ErrInvalidCredentials
				},
			},
			validator: validator,
		}
		c, w := newGinContext("POST", "/api/v1/auth/login", validBody)
		h.Login(c)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &AuthHandler{
			authService: &mockAuthService{
				loginFn: func(_ context.Context, _ dto.LoginRequest) (*dto.AuthResponse, error) {
					return nil, errors.New("db error")
				},
			},
			validator: validator,
		}
		c, w := newGinContext("POST", "/api/v1/auth/login", validBody)
		h.Login(c)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}
