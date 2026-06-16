package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

func newPostgresDriver(cfg *Config) (gorm.Dialector, error) {

	config := map[string]string{
		"host":     cfg.DB.Host,
		"port":     cfg.DB.Port,
		"user":     cfg.DB.User,
		"password": cfg.DB.Password,
		"dbname":   cfg.DB.DBName,
		"sslmode":  "disable",
	}

	props := lo.MapToSlice(config, func(k string, v string) string {
		return fmt.Sprintf("%s=%s", k, v)
	})

	dsn := strings.Join(props, " ")

	return postgres.Open(dsn), nil
}

func newGORMDatabase(i do.Injector) (*GORMWrapper, error) {

	dialector := do.MustInvoke[gorm.Dialector](i)
	logger := do.MustInvoke[*zap.Logger](i)

	gormLogger := zapgorm2.New(logger)

	var db *gorm.DB
	var err error

	for attempt := range 30 {

		db, err = gorm.Open(dialector, &gorm.Config{Logger: gormLogger})
		if err == nil {
			break
		}

		logger.Warn(
			"gorm open failed, retrying",
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)

		time.Sleep(time.Second)
	}

	if err != nil {
		return nil, fmt.Errorf("gorm open after 30 retries: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	schemas := []any{
		&domain.User{},
		&domain.Server{},
		&domain.Endpoint{},
		&domain.ServerEvent{},
	}

	if err := db.AutoMigrate(schemas...); err != nil {
		logger.Warn("failed to run migrations", zap.Error(err))
	}

	searchEnabled := false
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS pg_search").Error; err != nil {
		logger.Warn("pg_search extension not available, ParadeDB search disabled", zap.Error(err))
	} else if err := db.Exec(`CREATE INDEX IF NOT EXISTS servers_search_idx ON servers USING bm25 (id, name) WITH (key_field='id')`).Error; err != nil {
		logger.Warn("failed to create BM25 index, ParadeDB search disabled", zap.Error(err))
	} else {
		searchEnabled = true
	}

	return &GORMWrapper{db: db, SearchEnabled: searchEnabled}, nil
}

func RegisterGORMDB(i do.Injector) {

	do.Provide(i, func(i do.Injector) (gorm.Dialector, error) {
		cfg := do.MustInvoke[*Config](i)
		return newPostgresDriver(cfg)
	})

	do.Provide(i, newGORMDatabase)
}

type GORMWrapper struct {
	db            *gorm.DB
	SearchEnabled bool
}

func (gw *GORMWrapper) GetDB() *gorm.DB {
	return gw.db
}

func (gw *GORMWrapper) Shutdown() error {
	sqlDB, err := gw.db.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

type DBConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
}
