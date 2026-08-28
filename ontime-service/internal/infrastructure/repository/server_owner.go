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

// OwnedServer is the minimal projection of server_owners needed to authorize
// and window ontime calculations without a round-trip to the server-service.
type OwnedServer struct {
	ServerID  uint
	CreatedAt time.Time
}

func (r *ServerOwnerRepository) GetOwnedServerIDs(
	ctx context.Context, userID uint, serverIDs []uint,
) ([]uint, error) {

	if len(serverIDs) == 0 {
		return nil, nil
	}

	var owned []uint
	err := r.db.WithContext(ctx).
		Model(&domain.ServerOwner{}).
		Where("server_id IN ?", serverIDs).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Pluck("server_id", &owned).
		Error
	if err != nil {
		return nil, err
	}

	return owned, nil
}

func (r *ServerOwnerRepository) GetOwnedServers(
	ctx context.Context, userID uint, serverIDs []uint,
) ([]OwnedServer, error) {

	if len(serverIDs) == 0 {
		return nil, nil
	}

	var rows []OwnedServer
	err := r.db.WithContext(ctx).
		Model(&domain.ServerOwner{}).
		Where("server_id IN ?", serverIDs).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Select("server_id, created_at").
		Scan(&rows).
		Error

	if err != nil {
		return nil, err
	}

	return rows, nil
}

// ListByUser returns every owned server (server_id + created_at) for a user,
// used to enumerate the server set without a round-trip to server-service.
func (r *ServerOwnerRepository) ListByUser(
	ctx context.Context, userID uint,
) ([]OwnedServer, error) {

	var rows []OwnedServer
	err := r.db.WithContext(ctx).
		Model(&domain.ServerOwner{}).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Select("server_id, created_at").
		Scan(&rows).
		Error

	if err != nil {
		return nil, err
	}

	return rows, nil
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

	result := r.db.WithContext(ctx).Save(&owner)
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
