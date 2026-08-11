package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
)

type ServerOwnerRepository struct {
	db *gorm.DB
}

func NewServerOwnerRepository(db *gorm.DB) *ServerOwnerRepository {
	return &ServerOwnerRepository{db: db}
}

func RegisterServerOwnerRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerOwnerRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &ServerOwnerRepository{db: dbWrapper.GetDB()}, nil
	})
}

func (r *ServerOwnerRepository) Upsert(
	ctx context.Context,
	serverID uint, userID uuid.UUID,
	deletedAt *time.Time,
) error {

	owner := domain.ServerOwner{
		ServerID: serverID,
		UserID:   userID,
	}

	if deletedAt != nil {
		owner.DeletedAt = gorm.DeletedAt{
			Time:  *deletedAt,
			Valid: true,
		}
	}

	result := r.db.WithContext(ctx).
		Table("server_owners").
		Save(&owner)

	return result.Error
}

func (r *ServerOwnerRepository) Delete(ctx context.Context, serverID uint) error {

	_, err := gorm.G[domain.ServerOwner](r.db).
		Where("server_id = ?", serverID).
		Delete(ctx)

	return err
}

func (r *ServerOwnerRepository) GetByServerID(ctx context.Context, serverID uint) (*domain.ServerOwner, error) {

	owner, err := gorm.G[domain.ServerOwner](r.db).
		Where("server_id = ?", serverID).
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperrors.ErrNotFound
	}

	if err != nil {
		return nil, err
	}

	return &owner, nil
}
