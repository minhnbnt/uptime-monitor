package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
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

func (r *ServerOwnerRepository) ListByUserID(
	ctx context.Context, userID uuid.UUID, page, perPage int,
) ([]domain.ServerOwner, error) {

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	owners, err := gorm.G[domain.ServerOwner](r.db).
		Where("user_id = ?", userID).
		Order("server_id ASC").
		Offset((page - 1) * perPage).
		Limit(perPage).
		Find(ctx)

	if err != nil {
		return nil, err
	}

	return owners, nil
}

// ListByUserAndServerIDs returns the ownership rows for the given server IDs
// that belong to userID. The returned set is the intersection of the requested
// IDs and what the user actually owns, so callers can compare lengths to detect
// unowned (or non-existent) servers.
func (r *ServerOwnerRepository) ListByUserAndServerIDs(
	ctx context.Context, userID uuid.UUID, serverIDs []uint,
) ([]domain.ServerOwner, error) {

	serverIDs = lo.Uniq(serverIDs)
	if len(serverIDs) == 0 {
		return nil, nil
	}

	owners, err := gorm.G[domain.ServerOwner](r.db).
		Where("user_id = ? AND server_id IN ?", userID, serverIDs).
		Find(ctx)

	if err != nil {
		return nil, err
	}

	return owners, nil
}

// GetByServerAndUser returns the ownership row for a single server owned by the
// user, or ErrNotFound when the user does not own it.
func (r *ServerOwnerRepository) GetByServerAndUser(
	ctx context.Context, serverID uint, userID uuid.UUID,
) (*domain.ServerOwner, error) {

	owners, err := r.ListByUserAndServerIDs(ctx, userID, []uint{serverID})
	if err != nil {
		return nil, err
	}

	if len(owners) == 0 {
		return nil, apperrors.ErrNotFound
	}

	return &owners[0], nil
}
