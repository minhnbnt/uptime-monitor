package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/auth-service/internal/domain"
)

type SessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) *SessionRepository {
	return &SessionRepository{db: db}
}

func RegisterSessionRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*SessionRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return NewSessionRepository(dbWrapper.GetDB()), nil
	})
}

func (r *SessionRepository) Create(ctx context.Context, session *domain.Session) error {

	err := gorm.G[domain.Session](r.db).Create(ctx, session)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (r *SessionRepository) GetByJTI(ctx context.Context, jti string) (*domain.Session, error) {

	id, err := uuid.Parse(jti)
	if err != nil {
		return nil, nil
	}

	session, err := gorm.G[domain.Session](r.db).
		Where("jti = ?", id).
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("get session by jti: %w", err)
	}

	return &session, nil
}

func (r *SessionRepository) DeleteByJTI(ctx context.Context, jti string) error {

	id, err := uuid.Parse(jti)
	if err != nil {
		return nil
	}

	rowAffected, err := gorm.G[domain.Session](r.db).
		Where("jti = ?", id).
		Delete(ctx)

	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	if rowAffected > 1 {
		return fmt.Errorf("unexpected rows affected: %d", rowAffected)
	}

	return nil
}

func (r *SessionRepository) DeleteByJTIAndUser(ctx context.Context, userID uint, jti string) (bool, error) {

	id, err := uuid.Parse(jti)
	if err != nil {
		return false, nil
	}

	rowAffected, err := gorm.G[domain.Session](r.db).
		Where("user_id = ? AND jti = ?", userID, id).
		Delete(ctx)

	if err != nil {
		return false, fmt.Errorf("delete session by user: %w", err)
	}

	return rowAffected > 0, nil
}

func (r *SessionRepository) FindByUser(ctx context.Context, userID uint) ([]domain.Session, error) {

	sessions, err := gorm.G[domain.Session](r.db).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(ctx)

	if err != nil {
		return nil, fmt.Errorf("find sessions by user: %w", err)
	}

	return sessions, nil
}
