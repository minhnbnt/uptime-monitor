package service

import (
	"log/slog"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
)

type TokenService struct {
	tokenGenerator token.Generator
	logger         *slog.Logger
}

func RegisterTokenService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TokenService, error) {
		return &TokenService{
			tokenGenerator: do.MustInvoke[token.Generator](i),
			logger:         do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (s *TokenService) GenerateTokenPair(user *domain.User, scopes []string) (*dto.TokenPair, error) {

	jti, err := uuid.NewRandom()
	if err != nil {
		s.logger.Error("failed to generate uuid", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user, jti.String(), 0)
	if err != nil {
		s.logger.Error("failed to generate refresh token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user, scopes, jti.String())
	if err != nil {
		s.logger.Error("failed to generate access token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return &dto.TokenPair{
		JTI:          jti,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (s *TokenService) GenerateTokenPairWithCounter(user *domain.User, req dto.TokenGenerateRequest) (*dto.TokenPair, error) {

	refreshToken, err := s.tokenGenerator.GenerateRefreshToken(user, req.JTI, req.Counter)
	if err != nil {
		s.logger.Error("failed to generate refresh token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	accessToken, err := s.tokenGenerator.GenerateAccessToken(user, req.Scopes, req.JTI)
	if err != nil {
		s.logger.Error("failed to generate access token", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	sessionID, err := uuid.Parse(req.JTI)
	if err != nil {
		s.logger.Error("failed to parse jti", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return &dto.TokenPair{
		JTI:          sessionID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}
