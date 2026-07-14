package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/jwt"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/logger"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/token"
)

func testConfig() *config.Config {
	return &config.Config{
		JWT: config.JWTConfig{Key: "test-signing-key"},
		Token: config.TokenCfg{
			AccessTTL:     "15m",
			RefreshTTL:    "168h",
			AccessIssuer:  "uptime-monitor",
			RefreshIssuer: "uptime-monitor-refresh",
		},
	}
}

func setupProviderWithConfig(t *testing.T) (*jwt.Provider, *config.TokenConfig) {
	t.Helper()
	i := do.New()
	config.RegisterConfig(testConfig())(i)
	config.RegisterJwtConfig(i)
	config.RegisterTokenConfig(i)
	jwt.RegisterProvider(i)
	return do.MustInvoke[*jwt.Provider](i), do.MustInvoke[*config.TokenConfig](i)
}

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

	t.Run("email or username taken", func(t *testing.T) {
		svc := &AuthService{
			logger: logger.NewMockLogger(),
			userRepository: &mockUserRepo{
				createFn: func(_ context.Context, _ *domain.User) error {
					return apperrors.ErrEmailOrUsernameTaken
				},
			},
			passwordEncoder: &mockPasswordEncoder{
				encodeFn: func(_ string) (string, error) {
					return "hashed-pass", nil
				},
			},
		}

		_, err := svc.Register(t.Context(), req)
		if !errors.Is(err, apperrors.ErrEmailOrUsernameTaken) {
			t.Errorf("got %v, want apperrors.ErrEmailOrUsernameTaken", err)
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

func TestAuthService_Logout(t *testing.T) {

	t.Run("success", func(t *testing.T) {
		p, tc := setupProviderWithConfig(t)
		tokenStr := generateRefreshToken(t, p, tc, "5")

		svc := &AuthService{
			logger:         logger.NewMockLogger(),
			tokenValidator: token.NewTokenValidator(p, tc, nil, nil),
			revokedTokenRepository: &mockRevokedTokenRepo{
				revokeFn: func(_ context.Context, _ *jwt.Token) error {
					return nil
				},
			},
		}

		err := svc.Logout(t.Context(), tokenStr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		p, tc := setupProviderWithConfig(t)

		svc := &AuthService{
			logger:         logger.NewMockLogger(),
			tokenValidator: token.NewTokenValidator(p, tc, nil, nil),
		}

		err := svc.Logout(t.Context(), "invalid-token")
		if !errors.Is(err, apperrors.ErrInvalidRefreshToken) {
			t.Errorf("got %v, want apperrors.ErrInvalidRefreshToken", err)
		}
	})

	t.Run("revoke error", func(t *testing.T) {
		p, tc := setupProviderWithConfig(t)
		tokenStr := generateRefreshToken(t, p, tc, "5")

		svc := &AuthService{
			logger:         logger.NewMockLogger(),
			tokenValidator: token.NewTokenValidator(p, tc, nil, nil),
			revokedTokenRepository: &mockRevokedTokenRepo{
				revokeFn: func(_ context.Context, _ *jwt.Token) error {
					return errors.New("redis down")
				},
			},
		}

		err := svc.Logout(t.Context(), tokenStr)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestAuthService_Refresh(t *testing.T) {
	validUser := domainUser(5, "a@b.com", "user1")

	t.Run("success", func(t *testing.T) {
		p, tc := setupProviderWithConfig(t)
		tokenStr := generateRefreshToken(t, p, tc, "5")

		svc := &AuthService{
			logger:         logger.NewMockLogger(),
			tokenValidator: token.NewTokenValidator(p, tc, &mockRevokedTokenRepo{}, logger.NewMockLogger()),
			userRepository: &mockUserRepo{
				findByIDFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &validUser, nil
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

		result, err := svc.Refresh(t.Context(), dto.RefreshRequest{RefreshToken: tokenStr})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.AccessToken != "access-token" {
			t.Errorf("AccessToken = %q, want access-token", result.AccessToken)
		}
		if result.RefreshToken != "refresh-token" {
			t.Errorf("RefreshToken = %q, want refresh-token", result.RefreshToken)
		}
		if result.User.ID != 5 {
			t.Errorf("User.ID = %d, want 5", result.User.ID)
		}
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		p, tc := setupProviderWithConfig(t)

		svc := &AuthService{
			logger:         logger.NewMockLogger(),
			tokenValidator: token.NewTokenValidator(p, tc, &mockRevokedTokenRepo{}, logger.NewMockLogger()),
		}

		_, err := svc.Refresh(t.Context(), dto.RefreshRequest{RefreshToken: "invalid"})
		if !errors.Is(err, apperrors.ErrInvalidRefreshToken) {
			t.Errorf("got %v, want apperrors.ErrInvalidRefreshToken", err)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		p, tc := setupProviderWithConfig(t)
		tokenStr := generateRefreshToken(t, p, tc, "99")

		svc := &AuthService{
			logger:         logger.NewMockLogger(),
			tokenValidator: token.NewTokenValidator(p, tc, &mockRevokedTokenRepo{}, logger.NewMockLogger()),
			userRepository: &mockUserRepo{
				findByIDFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return nil, nil
				},
			},
		}

		_, err := svc.Refresh(t.Context(), dto.RefreshRequest{RefreshToken: tokenStr})
		if !errors.Is(err, apperrors.ErrInvalidCredentials) {
			t.Errorf("got %v, want apperrors.ErrInvalidCredentials", err)
		}
	})

	t.Run("find by id error", func(t *testing.T) {
		p, tc := setupProviderWithConfig(t)
		tokenStr := generateRefreshToken(t, p, tc, "5")

		svc := &AuthService{
			logger:         logger.NewMockLogger(),
			tokenValidator: token.NewTokenValidator(p, tc, &mockRevokedTokenRepo{}, logger.NewMockLogger()),
			userRepository: &mockUserRepo{
				findByIDFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return nil, errors.New("db error")
				},
			},
		}

		_, err := svc.Refresh(t.Context(), dto.RefreshRequest{RefreshToken: tokenStr})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("access token generation error", func(t *testing.T) {
		p, tc := setupProviderWithConfig(t)
		tokenStr := generateRefreshToken(t, p, tc, "5")

		svc := &AuthService{
			logger:         logger.NewMockLogger(),
			tokenValidator: token.NewTokenValidator(p, tc, &mockRevokedTokenRepo{}, logger.NewMockLogger()),
			userRepository: &mockUserRepo{
				findByIDFn: func(_ context.Context, _ uint) (*domain.User, error) {
					return &validUser, nil
				},
			},
			tokenGenerator: &mockTokenGenerator{
				generateAccessTokenFn: func(_ *domain.User) (string, error) {
					return "", errors.New("jwt failed")
				},
			},
		}

		_, err := svc.Refresh(t.Context(), dto.RefreshRequest{RefreshToken: tokenStr})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func generateRefreshToken(t *testing.T, p *jwt.Provider, tc *config.TokenConfig, sub string) string {
	t.Helper()
	tokenStr, err := p.NewToken(tc.GetRefreshTokenIssuer(), map[string]any{
		"sub": sub,
		"jti": "0195f0b0-0000-7000-8000-000000000000",
		"exp": time.Now().Add(7 * 24 * time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("NewToken error: %v", err)
	}
	return tokenStr
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
