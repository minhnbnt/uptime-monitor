package service

import (
	"context"
	"errors"
	"testing"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
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
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
				createFn: func(_ context.Context, u *domain.User) error {
					u.ID = 10
					return nil
				},
			},
			encoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "hashed-pass", nil
				},
			},
			jwtParser: &mockTokenParser{
				newTokenFn: func(_ string, claims map[string]any) (string, error) {
					if claims["sub"] != uint(10) {
						t.Error("wrong sub claim")
					}
					return "jwt-token", nil
				},
			},
		}

		result, err := svc.Register(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Token != "jwt-token" {
			t.Errorf("Token = %q, want jwt-token", result.Token)
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
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					u := domainUser(1, "a@b.com", "other")
					return &u, nil
				},
			},
		}

		_, err := svc.Register(context.Background(), req)
		if !errors.Is(err, ErrEmailOrUsernameTaken) {
			t.Errorf("got %v, want ErrEmailOrUsernameTaken", err)
		}
	})

	t.Run("find error", func(t *testing.T) {
		svc := &AuthService{
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, errors.New("db error")
				},
			},
		}

		_, err := svc.Register(context.Background(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("encode error", func(t *testing.T) {
		svc := &AuthService{
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
				createFn: func(_ context.Context, _ *domain.User) error {
					return nil
				},
			},
			encoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "", errors.New("hash failed")
				},
			},
		}

		_, err := svc.Register(context.Background(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("create error", func(t *testing.T) {
		svc := &AuthService{
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
				createFn: func(_ context.Context, _ *domain.User) error {
					return errors.New("create failed")
				},
			},
			encoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "hash", nil
				},
			},
		}

		_, err := svc.Register(context.Background(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("token error", func(t *testing.T) {
		svc := &AuthService{
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, nil
				},
				createFn: func(_ context.Context, u *domain.User) error {
					u.ID = 1
					return nil
				},
			},
			encoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "hash", nil
				},
			},
			jwtParser: &mockTokenParser{
				newTokenFn: func(_ string, _ map[string]any) (string, error) {
					return "", errors.New("jwt failed")
				},
			},
		}

		_, err := svc.Register(context.Background(), req)
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
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &matchUser, nil
				},
			},
			encoder: &mockPasswordEncoder{
				verifyFn: func(_, _ string) (bool, error) {
					return true, nil
				},
			},
			jwtParser: &mockTokenParser{
				newTokenFn: func(_ string, _ map[string]any) (string, error) {
					return "jwt-token", nil
				},
			},
		}

		result, err := svc.Login(context.Background(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Token != "jwt-token" {
			t.Errorf("Token = %q", result.Token)
		}
		if result.User.ID != 5 {
			t.Errorf("User.ID = %d, want 5", result.User.ID)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		svc := &AuthService{
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return nil, errors.New("not found")
				},
			},
		}

		_, err := svc.Login(context.Background(), req)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("got %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		svc := &AuthService{
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &matchUser, nil
				},
			},
			encoder: &mockPasswordEncoder{
				verifyFn: func(_, _ string) (bool, error) {
					return false, nil
				},
			},
		}

		_, err := svc.Login(context.Background(), req)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Errorf("got %v, want ErrInvalidCredentials", err)
		}
	})

	t.Run("verify error", func(t *testing.T) {
		svc := &AuthService{
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &matchUser, nil
				},
			},
			encoder: &mockPasswordEncoder{
				verifyFn: func(_, _ string) (bool, error) {
					return false, errors.New("verify error")
				},
			},
		}

		_, err := svc.Login(context.Background(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("token error", func(t *testing.T) {
		svc := &AuthService{
			userRepo: &mockUserRepo{
				findByEmailOrUsernameFn: func(_ context.Context, _ string) (*domain.User, error) {
					return &matchUser, nil
				},
			},
			encoder: &mockPasswordEncoder{
				verifyFn: func(_, _ string) (bool, error) {
					return true, nil
				},
			},
			jwtParser: &mockTokenParser{
				newTokenFn: func(_ string, _ map[string]any) (string, error) {
					return "", errors.New("jwt failed")
				},
			},
		}

		_, err := svc.Login(context.Background(), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
