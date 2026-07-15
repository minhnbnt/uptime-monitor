package config

import (
	"fmt"
	"sync"

	"github.com/samber/do/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type GORMWrapper struct {
	db *gorm.DB
	mu sync.Mutex
}

func (w *GORMWrapper) GetDB() *gorm.DB {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.db
}

func RegisterGORMDB(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*GORMWrapper, error) {
		cfg := do.MustInvoke[*Config](i)
		return newGORMDatabase(cfg)
	})
}

func newGORMDatabase(cfg *Config) (*GORMWrapper, error) {

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.DB.Host, cfg.DB.Port, cfg.DB.User, cfg.DB.Password, cfg.DB.DBName,
	)

	dialector := postgres.New(postgres.Config{DSN: dsn})

	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(5)

	if err := db.AutoMigrate(&domain.NotificationConfig{}); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}

	return &GORMWrapper{db: db}, nil
}
