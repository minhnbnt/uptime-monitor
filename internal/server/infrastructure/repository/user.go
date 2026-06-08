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

func RegisterUserRepository(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*UserRepository, error) {
		dbWrapper := do.MustInvoke[*config.GORMWrapper](i)
		return &UserRepository{db: dbWrapper.GetDB()}, nil
	})
}

func (r *UserRepository) Create(ctx context.Context, user *domain.User) error {
	return gorm.G[domain.User](r.db).Create(ctx, user)
}

func (r *UserRepository) FindByEmailOrUsername(ctx context.Context, login string) (*domain.User, error) {

	user, err := gorm.G[domain.User](r.db).
		Where("email = ? OR username = ?", login, login).
		First(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	return &user, nil
}
