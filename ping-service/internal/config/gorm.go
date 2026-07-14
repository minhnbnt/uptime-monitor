package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	sloggorm "github.com/orandin/slog-gorm"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type GORMWrapper struct {
	db *gorm.DB
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
	log := do.MustInvoke[*slog.Logger](i)

	gormLogger := sloggorm.New(
		sloggorm.WithHandler(log.Handler()),
		sloggorm.WithTraceAll(),
	)

	var db *gorm.DB
	var err error

	for attempt := range 30 {
		db, err = gorm.Open(dialector, &gorm.Config{
			Logger:         gormLogger,
			TranslateError: true,
		})
		if err == nil {
			break
		}
		log.Warn(
			"gorm open failed, retrying",
			slog.Int("attempt", attempt+1),
			slog.Any("error", err),
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

	if err := db.AutoMigrate(&domain.ServerEvent{}); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &GORMWrapper{db: db}, nil
}

func RegisterGORMDB(i do.Injector) {
	do.Provide(i, func(i do.Injector) (gorm.Dialector, error) {
		cfg := do.MustInvoke[*Config](i)
		return newPostgresDriver(cfg)
	})

	do.Provide(i, newGORMDatabase)
}
