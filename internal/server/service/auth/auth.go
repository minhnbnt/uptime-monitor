package auth

import (
	"context"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	authrepo "github.com/minhnbnt/uptime-monitor/internal/repository/auth"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	serverinfra "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure"
	jwtutil "github.com/minhnbnt/uptime-monitor/internal/server/infrastructure/jwt"
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
	logger                 logger.Logger
}

func RegisterAuthService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthService, error) {
		return &AuthService{
			userRepository:         do.MustInvoke[*authrepo.UserRepository](i),
			passwordEncoder:        do.MustInvoke[*serverinfra.Argon2PasswordEncoder](i),
			tokenGenerator:         do.MustInvoke[TokenGenerator](i),
			tokenValidator:         do.MustInvoke[*TokenValidator](i),
			revokedTokenRepository: do.MustInvoke[RevokedTokenRepository](i),
			logger:                 do.MustInvoke[logger.Logger](i),
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
		s.logger.Error("failed to find user", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	if existing != nil {
		return nil, apperrors.ErrEmailOrUsernameTaken
	}

	hash, err := s.passwordEncoder.Encode(req.Password)
	if err != nil {
		s.logger.Error("failed to hash password", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	user := domain.User{
		Email:    req.Email,
		Username: req.Username,
		Password: hash,
		Name:     req.Name,
	}

	if err := s.userRepository.Create(ctx, &user); err != nil {
		s.logger.Error("failed to create user", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(&user)
	if err != nil {
		s.logger.Error("failed to generate access token", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(&user)
	if err != nil {
		s.logger.Error("failed to generate refresh token", logger.Error(err))
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
		return apperrors.ErrInvalidCredentials
	}

	if err := s.revokedTokenRepository.Revoke(ctx, token); err != nil {
		s.logger.Error("failed to revoke token", logger.Error(err))
		return apperrors.ErrInternal
	}

	return nil
}

func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (*dto.AuthResponse, error) {

	user, err := s.userRepository.FindByEmailOrUsername(ctx, req.Login)
	if err != nil {
		s.logger.Error("failed to find user", logger.Error(err))
		return nil, apperrors.ErrInternal
	}
	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	ok, err := s.passwordEncoder.Verify(req.Password, user.Password)
	if err != nil {
		s.logger.Error("failed to verify password", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	if !ok {
		return nil, apperrors.ErrInvalidCredentials
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user)
	if err != nil {
		s.logger.Error("failed to generate access token", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user)
	if err != nil {
		s.logger.Error("failed to generate refresh token", logger.Error(err))
		return nil, apperrors.ErrInternal
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
		return nil, apperrors.ErrInvalidCredentials
	}

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to find user", logger.Error(err))
		return nil, apperrors.ErrInternal
	}
	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user)
	if err != nil {
		s.logger.Error("failed to generate access token", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user)
	if err != nil {
		s.logger.Error("failed to generate refresh token", logger.Error(err))
		return nil, apperrors.ErrInternal
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(*user),
	}, nil
}
