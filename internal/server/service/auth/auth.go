package auth

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/repository/auth"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
)

var (
	ErrEmailOrUsernameTaken = errors.New("email or username already exists")
	ErrInvalidCredentials   = errors.New("invalid email/username or password")
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmailOrUsername(ctx context.Context, login string) (*domain.User, error)
	FindByID(ctx context.Context, id uint) (*domain.User, error)
}

type PasswordEncoder interface {
	Encode(password string) (string, error)
	Verify(password, encodedHash string) (bool, error)
}

type TokenGenerator interface {
	GenerateAccessToken(user *domain.User) (string, error)
	GenerateRefreshToken(user *domain.User) (string, error)
}

type RevokedTokenRepository interface {
	Revoke(ctx context.Context, token *jwtutil.Token) error
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

type AuthService struct {
	userRepository         UserRepository
	passwordEncoder        PasswordEncoder
	tokenGenerator         TokenGenerator
	tokenValidator         *TokenValidator
	revokedTokenRepository RevokedTokenRepository
}

func RegisterAuthService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthService, error) {
		return &AuthService{
			userRepository:         do.MustInvoke[*authrepo.UserRepository](i),
			passwordEncoder:        do.MustInvoke[*serverinfra.Argon2PasswordEncoder](i),
			tokenGenerator:         do.MustInvoke[TokenGenerator](i),
			tokenValidator:         do.MustInvoke[*TokenValidator](i),
			revokedTokenRepository: do.MustInvoke[RevokedTokenRepository](i),
		}, nil
	})
}

func toUserProfile(u domain.User) dto.UserProfile {
	return dto.UserProfile{
		ID:       u.ID,
		Email:    u.Email,
		Username: u.Username,
		Name:     u.Name,
	}
}

func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*dto.AuthResponse, error) {

	existing, err := s.userRepository.FindByEmailOrUsername(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	if existing != nil {
		return nil, ErrEmailOrUsernameTaken
	}

	hash, err := s.passwordEncoder.Encode(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := domain.User{
		Email:    req.Email,
		Username: req.Username,
		Password: hash,
		Name:     req.Name,
	}

	if err := s.userRepository.Create(ctx, &user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(user),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {

	token, err := s.tokenValidator.ParseRefreshToken(refreshToken)
	if err != nil {
		return ErrInvalidCredentials
	}

	return s.revokedTokenRepository.Revoke(ctx, token)
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {

	user, err := s.userRepository.FindByEmailOrUsername(ctx, req.Login)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	ok, err := s.passwordEncoder.Verify(req.Password, user.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to verify password: %w", err)
	}

	if !ok {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(*user),
	}, nil
}

func (s *AuthService) Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error) {

	userID, _, err := s.tokenValidator.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(*user),
	}, nil
}
