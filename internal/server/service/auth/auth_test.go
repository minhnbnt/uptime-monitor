package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

func TestAuthService_Register(t *testing.T) {
	req := dto.RegisterRequest{
		Email:    "a@b.com",
		Username: "user1",
		Password: "password123",
		Name:     "Test",
	}

	t.Run("success", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
				createFn: func(_ context.Context, u *domain.User) error {
					u.ID = 10
					return nil
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "hashed-pass", nil
				},
			},
			tokenGenerator: &mockTokenGenerator{
				generateAccessTokenFn: func(user *domain.User) (string, error) {
					if user.ID != 10 {
						t.Error("wrong user id")
					}
					return "access-token", nil
				},
				generateRefreshTokenFn: func(user *domain.User) (string, error) {
					return "refresh-token", nil
				},
			},
		}

		result, err := svc.Register(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AccessToken != "access-token" {
			t.Errorf("Token = %q, want access-token", result.AccessToken)
		}
		if result.RefreshToken != "refresh-token" {
			t.Errorf("RefreshToken = %q, want refresh-token", result.RefreshToken)
		}
		if result.User.ID != 10 {
			t.Errorf("User.ID = %d, want 10", result.User.ID)
		}
		if result.User.Email != "a@b.com" {
			t.Errorf("User.Email = %q", result.User.Email)
		}
	})

	t.Run("email taken", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					u := domainUser(1, "a@b.com", "other")
					return &u, nil
				},
			},
		}

		_, err := svc.Register(t.Context(), req)
		if !errors.Is(err, apperrors.ErrEmailOrUsernameTaken) {
			t.Errorf("got %v, want apperrors.ErrEmailOrUsernameTaken", err)
		}
	})

	t.Run("find error", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, errors.New("db error")
				},
			},
		}

		_, err := svc.Register(t.Context(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("encode error", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
				createFn: func(_ context.Context, _ *domain.User) error {
					return nil
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "", errors.New("hash failed")
				},
			},
		}

		_, err := svc.Register(t.Context(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("create error", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
				createFn: func(_ context.Context, _ *domain.User) error {
					return errors.New("create failed")
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "hash", nil
				},
			},
		}

		_, err := svc.Register(t.Context(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("token error", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
				createFn: func(_ context.Context, u *domain.User) error {
					u.ID = 1
					return nil
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "hash", nil
				},
			},
			tokenGenerator: &mockTokenGenerator{
				generateAccessTokenFn: func(_ *domain.User) (string, error) {
					return "", errors.New("jwt failed")
				},
			},
		}

		_, err := svc.Register(t.Context(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAuthService_Login(t *testing.T) {
	req := dto.LoginRequest{
		Login:    "a@b.com",
		Password: "password123",
	}

	matchUser := domainUser(5, "a@b.com", "user1")

	t.Run("success", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &matchUser, nil
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				verifyFn: func(_, _ string) (bool, error) {
					return true, nil
				},
			},
			tokenGenerator: &mockTokenGenerator{
				generateAccessTokenFn: func(_ *domain.User) (string, error) {
					return "access-token", nil
				},
				generateRefreshTokenFn: func(_ *domain.User) (string, error) {
					return "refresh-token", nil
				},
			},
		}

		result, err := svc.Login(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AccessToken != "access-token" {
			t.Errorf("Token = %q, want access-token", result.AccessToken)
		}
		if result.RefreshToken != "refresh-token" {
			t.Errorf("RefreshToken = %q, want refresh-token", result.RefreshToken)
		}
		if result.User.ID != 5 {
			t.Errorf("User.ID = %d, want 5", result.User.ID)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
			},
		}

		_, err := svc.Login(t.Context(), req)
		if !errors.Is(err, apperrors.ErrInvalidCredentials) {
			t.Errorf("got %v, want apperrors.ErrInvalidCredentials", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &matchUser, nil
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				verifyFn: func(_, _ string) (bool, error) {
					return false, nil
				},
			},
		}

		_, err := svc.Login(t.Context(), req)
		if !errors.Is(err, apperrors.ErrInvalidCredentials) {
			t.Errorf("got %v, want apperrors.ErrInvalidCredentials", err)
		}
	})

	t.Run("verify error", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &matchUser, nil
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				verifyFn: func(_, _ string) (bool, error) {
					return false, errors.New("verify error")
				},
			},
		}

		_, err := svc.Login(t.Context(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("token error", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &matchUser, nil
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				verifyFn: func(_, _ string) (bool, error) {
					return true, nil
				},
			},
			tokenGenerator: &mockTokenGenerator{
				generateAccessTokenFn: func(_ *domain.User) (string, error) {
					return "", errors.New("jwt failed")
				},
			},
		}

		_, err := svc.Login(t.Context(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
