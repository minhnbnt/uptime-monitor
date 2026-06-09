package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/repository/auth"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
)

var (
	ErrEmailOrUsernameTaken = errors.New("email or username already exists")
	ErrInvalidCredentials   = errors.New("invalid email/username or password")
)

type AuthService struct {
	userRepository  UserRepository
	passwordEncoder PasswordEncoder
	tokenGenerator  TokenGenerator
}

func RegisterAuthService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthService, error) {
		return &AuthService{
			userRepository:  do.MustInvoke[*authrepo.UserRepository](i),
			passwordEncoder: do.MustInvoke[*serverinfra.Argon2PasswordEncoder](i),
			tokenGenerator:  do.MustInvoke[TokenGenerator](i),
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
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(user),
	}, nil
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
		Token:        accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(*user),
	}, nil
}
