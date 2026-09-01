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
	userRepository  UserRepository
	passwordEncoder PasswordEncoder
	tokenValidator  *token.Validator
	sessionService  *SessionService
	logger          *slog.Logger
}

func RegisterAuthService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthService, error) {
		return &AuthService{
			userRepository:  do.MustInvoke[*repository.UserRepository](i),
			passwordEncoder: do.MustInvoke[*argon2.PasswordEncoder](i),
			tokenValidator:  do.MustInvoke[*token.Validator](i),
			sessionService:  do.MustInvoke[*SessionService](i),
			logger:          do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func toUserProfile(u *domain.User) dto.UserProfile {
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

	tokenPair, err := s.sessionService.CreateSession(ctx, &user, domain.DefaultScopes())
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         toUserProfile(&user),
	}, nil
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

	tokenPair, err := s.sessionService.CreateSession(ctx, user, domain.DefaultScopes())
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         toUserProfile(user),
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

	profile := toUserProfile(user)
	return &profile, nil
}

func (s *AuthService) Refresh(ctx context.Context, req dto.RefreshRequest) (*dto.AuthResponse, error) {

	info, err := s.tokenValidator.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	session, err := s.sessionService.GetSessionByJTI(ctx, info.JTI)
	if err != nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}
	if session == nil {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	if info.Counter != session.Counter {
		return nil, apperrors.ErrInvalidRefreshToken
	}

	user, err := s.userRepository.FindByID(ctx, info.UserID)
	if err != nil {
		s.logger.Error("failed to find user", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	if user == nil {
		return nil, apperrors.ErrInvalidCredentials
	}

	tokenPair, err := s.sessionService.RotateSession(ctx, user, session)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         toUserProfile(user),
	}, nil
}

func (s *AuthService) CreatePingSession(ctx context.Context, userID uint) (*dto.AuthResponse, error) {

	user, err := s.userRepository.FindByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to find user", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	if user == nil {
		return nil, apperrors.ErrNotFound
	}

	scopes := []string{string(domain.ScopePing)}
	tokenPair, err := s.sessionService.CreateSession(ctx, user, scopes)
	if err != nil {
		return nil, err
	}

	return &dto.AuthResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		User:         toUserProfile(user),
	}, nil
}
