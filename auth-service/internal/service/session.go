package service

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/infrastructure/token"
)

type SessionRepository interface {
	Create(ctx context.Context, session *domain.Session) error
	GetByJTI(ctx context.Context, jti string) (*domain.Session, error)
	DeleteByJTI(ctx context.Context, jti string) error
	DeleteByJTIAndUser(ctx context.Context, userID uint, jti string) (bool, error)
	FindByUser(ctx context.Context, userID uint) ([]domain.Session, error)
	IncrementCounter(ctx context.Context, session *domain.Session) (bool, error)
}

// SessionService owns the session lifecycle: listing, revocation and logout.
// Token issuance lives in AuthService.
type SessionService struct {
	sessionRepository SessionRepository
	tokenValidator    *token.Validator
	tokenService      *TokenService
	logger            *slog.Logger
	tokenConfig       *config.TokenConfig
}

func RegisterSessionService(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*SessionService, error) {
		return &SessionService{
			sessionRepository: do.MustInvoke[*repository.SessionRepository](i),
			tokenValidator:    do.MustInvoke[*token.Validator](i),
			tokenService:      do.MustInvoke[*TokenService](i),
			logger:            do.MustInvoke[*slog.Logger](i),
			tokenConfig:       do.MustInvoke[*config.TokenConfig](i),
		}, nil
	})
}

func (s *SessionService) Logout(ctx context.Context, refreshToken string) error {

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

func (s *SessionService) ListSessions(ctx context.Context, userID uint, currentSessionID string, page, perPage int) ([]dto.SessionInfo, int, error) {

	sessions, err := s.sessionRepository.FindByUser(ctx, userID)
	if err != nil {
		s.logger.Error("failed to list sessions", slog.Any("error", err))
		return nil, 0, apperrors.ErrInternal
	}

	now := time.Now()
	active := lo.Filter(sessions, func(s domain.Session, _ int) bool {
		return s.ExpiresAt.After(now)
	})

	total := len(active)

	start := min((page-1)*perPage, total)
	end := min(start+perPage, total)

	items := lo.Map(active[start:end], func(session domain.Session, _ int) dto.SessionInfo {

		id := session.JTI.String()

		return dto.SessionInfo{
			ID:        id,
			Scopes:    session.ScopeList(),
			Current:   id == currentSessionID,
			CreatedAt: session.CreatedAt,
			ExpiresAt: session.ExpiresAt,
		}
	})

	return items, total, nil
}

func (s *SessionService) RevokeSession(ctx context.Context, userID uint, sessionID string) error {

	found, err := s.sessionRepository.DeleteByJTIAndUser(ctx, userID, sessionID)
	if err != nil {
		s.logger.Error("failed to revoke session", slog.Any("error", err))
		return apperrors.ErrInternal
	}

	if !found {
		return apperrors.ErrNotFound
	}

	return nil
}

func (s *SessionService) RotateSession(ctx context.Context, user *domain.User, session *domain.Session) (*dto.TokenPair, error) {

	// Atomic CAS: increment counter only if it matches
	updated, err := s.sessionRepository.IncrementCounter(ctx, session)
	if err != nil {
		s.logger.Error("failed to increment session counter", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}
	if !updated {
		// Counter mismatch — replay hoặc concurrent write
		return nil, apperrors.ErrInvalidRefreshToken
	}

	request := dto.TokenGenerateRequest{
		JTI:     session.JTI.String(),
		Counter: session.Counter + 1,
		Scopes:  session.ScopeList(),
	}

	tokenPair, err := s.tokenService.GenerateTokenPairWithCounter(user, request)
	if err != nil {
		return nil, err
	}

	return tokenPair, nil
}

func (s *SessionService) GetSessionByJTI(ctx context.Context, jti string) (*domain.Session, error) {

	session, err := s.sessionRepository.GetByJTI(ctx, jti)
	if err != nil {
		s.logger.Error("failed to get session", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}
	if session == nil {
		return nil, nil
	}
	// Check if session is expired
	if session.ExpiresAt.Before(time.Now()) {
		return nil, nil
	}

	return session, nil
}

func (s *SessionService) CreateSession(ctx context.Context, user *domain.User, scope []string) (*dto.TokenPair, error) {

	tokenPair, err := s.tokenService.GenerateTokenPair(user, scope)
	if err != nil {
		return nil, err
	}

	session := domain.Session{
		UserID:    user.ID,
		JTI:       tokenPair.JTI,
		Scopes:    strings.Join(scope, " "),
		ExpiresAt: time.Now().Add(s.tokenConfig.GetRefreshTokenTTL()),
	}

	err = s.sessionRepository.Create(ctx, &session)
	if err != nil {
		s.logger.Error("failed to store session", slog.Any("error", err))
		return nil, apperrors.ErrInternal
	}

	return tokenPair, nil
}
