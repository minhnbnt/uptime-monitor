package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
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
	serverID, userID uint,
	deletedAt *time.Time,
) error {

	owner := domain.ServerOwner{
		ServerID: serverID,
		UserID:   userID,
	}

	if deletedAt != nil {
		owner.DeletedAt = gorm.DeletedAt{Time: *deletedAt, Valid: true}
	}

	result := r.db.WithContext(ctx).
		Table("server_owners").
		Save(&owner)

	return result.Error
}

func (r *ServerOwnerRepository) Delete(ctx context.Context, serverID uint) error {

	rowAffected, err := gorm.G[domain.ServerOwner](r.db).
		Where("server_id = ?", serverID).
		Delete(ctx)

	if err != nil {
		return fmt.Errorf("delete server owner: %w", err)
	}

	if rowAffected == 0 {
		return fmt.Errorf("no server owner found with server_id = %d", serverID)
	}

	return nil
}
