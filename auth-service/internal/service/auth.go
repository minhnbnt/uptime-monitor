package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/argon2"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
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

type AuthService struct {
	userRepository         UserRepository
	passwordEncoder        PasswordEncoder
	tokenGenerator         token.Generator
	tokenValidator         *token.Validator
	revokedTokenRepository token.RevokedTokenRepository
	logger                 *slog.Logger
}

func RegisterAuthService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthService, error) {
		return &AuthService{
			userRepository:         do.MustInvoke[*repository.UserRepository](i),
			passwordEncoder:        do.MustInvoke[*argon2.PasswordEncoder](i),
			tokenGenerator:         do.MustInvoke[token.Generator](i),
			tokenValidator:         do.MustInvoke[*token.Validator](i),
			revokedTokenRepository: do.MustInvoke[*repository.RedisRevokedTokenRepository](i),
			logger:                 do.MustInvoke[*slog.Logger](i),
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

	hash, err := s.passwordEncoder.Encode(req.Password)
	if err != nil {
		s.logger.Error("failed to hash password", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	user := domain.User{
		Email:    req.Email,
		Username: req.Username,
		Password: hash,
		Name:     req.Name,
	}

	err = s.userRepository.Create(ctx, &user)
	if errors.Is(err, apperrors.ErrEmailOrUsernameTaken) {
		return nil, err
	}

	if err != nil {
		s.logger.Error("failed to create user", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(&user)
	if err != nil {
		s.logger.Error("failed to generate access token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(&user)
	if err != nil {
		s.logger.Error("failed to generate refresh token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
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
		return apperrors.ErrInvalidRefreshToken
	}

	if err := s.revokedTokenRepository.Revoke(ctx, token); err != nil {
		s.logger.Error("failed to revoke token", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	return nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {

	user, err := s.userRepository.FindByEmailOrUsername(ctx, req.Login)
	if err != nil {
		s.logger.Error("failed to find user", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}
	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	ok, err := s.passwordEncoder.Verify(req.Password, user.Password)
	if err != nil {
		s.logger.Error("failed to verify password", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	if !ok {
		return nil, apperrors.ErrInvalidCredentials
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user)
	if err != nil {
		s.logger.Error("failed to generate access token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user)
	if err != nil {
		s.logger.Error("failed to generate refresh token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(*user),
	}, nil
}

func (s *AuthService) GetUser(ctx context.Context, id uint) (*dto.UserProfile, error) {

	user, err := s.userRepository.FindByID(ctx, id)
	if err != nil {
		s.logger.Error("failed to find user", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}
	if user == nil {
		return nil, apperrors.ErrNotFound
	}

	profile := toUserProfile(*user)
	return &profile, nil
}

func (s *AuthService) Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error) {

	userID, _, err := s.tokenValidator.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to find user", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}
	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user)
	if err != nil {
		s.logger.Error("failed to generate access token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user)
	if err != nil {
		s.logger.Error("failed to generate refresh token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(*user),
	}, nil
}
