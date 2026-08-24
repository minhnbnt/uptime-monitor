package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
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
	userRepository    UserRepository
	passwordEncoder   PasswordEncoder
	tokenGenerator    token.Generator
	tokenValidator    *token.Validator
	sessionRepository SessionRepository
	tokenConfig       *config.TokenConfig
	logger            *slog.Logger
}

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	DeleteByJTI(ctx context.Context, jti string) error
	FindByUser(ctx context.Context, userID uint) ([]domain.Session, error)
}

func RegisterAuthService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthService, error) {
		return &AuthService{
			userRepository:    do.MustInvoke[*repository.UserRepository](i),
			passwordEncoder:   do.MustInvoke[*argon2.PasswordEncoder](i),
			tokenGenerator:    do.MustInvoke[token.Generator](i),
			tokenValidator:    do.MustInvoke[*token.Validator](i),
			sessionRepository: do.MustInvoke[*repository.SessionRepository](i),
			tokenConfig:       do.MustInvoke[*config.TokenConfig](i),
			logger:            do.MustInvoke[*slog.Logger](i),
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

	return s.issueTokens(ctx, &user, domain.DefaultScopes())
}

func (s *AuthService) issueTokens(ctx context.Context, user *domain.User, scopes []string) (*dto.AuthResponse, error) {

	if len(scopes) == 0 {
		scopes = domain.DefaultScopes()
	}

	refreshToken, jti, err := s.tokenGenerator.GenerateRefreshToken(user)
	if err != nil {
		s.logger.Error("failed to generate refresh token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	sessionID, err := uuid.Parse(jti)
	if err != nil {
		s.logger.Error("failed to parse session id", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	session := &domain.Session{
		UserID:    user.ID,
		JTI:       sessionID,
		Scopes:    strings.Join(scopes, " "),
		ExpiresAt: time.Now().Add(s.tokenConfig.GetRefreshTokenTTL()),
	}

	if err := s.sessionRepository.Create(ctx, session); err != nil {
		s.logger.Error("failed to create session", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user, scopes, jti)
	if err != nil {
		s.logger.Error("failed to generate access token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return &dto.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         toUserProfile(*user),
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {

	token, err := s.tokenValidator.ParseRefreshToken(refreshToken)
	if err != nil {
		return apperrors.ErrInvalidRefreshToken
	}

	jti, err := token.JTI()
	if err != nil {
		return apperrors.ErrInvalidRefreshToken
	}

	if err := s.sessionRepository.DeleteByJTI(ctx, jti); err != nil {
		s.logger.Error("failed to delete session", slog.Any("error", err))
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

	return s.issueTokens(ctx, user, domain.DefaultScopes())
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

	info, err := s.tokenValidator.ValidateRefreshToken(ctx, req.RefreshToken)
	if err != nil {
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

	return s.issueTokens(ctx, user, info.Scopes)
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
	return s.issueTokens(ctx, user, scopes)
}
