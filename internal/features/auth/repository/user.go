package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func RegisterUserRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*UserRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return NewUserRepository(dbWrapper.GetDB()), nil
	})
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*domain.User, error) {

	user, err := gorm.G[domain.User](r.db).Where("id = ?", id).First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	return gorm.G[domain.User](r.db).Create(ctx, user)
}

func (r *UserRepository) FindByEmailOrUsername(ctx context.Context, login string) (*domain.User, error) {

	user, err := gorm.G[domain.User](r.db).
		Where("email = ? OR username = ?", login, login).
		First(ctx)

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}

	return &user, nil
}
