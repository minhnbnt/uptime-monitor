package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/samber/do/v2"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"

	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
)

func newPostgresDriver(i do.Injector) (gorm.Dialector, error) {

	config := map[string]string{
		"host":     os.Getenv("DB_HOST"),
		"port":     os.Getenv("DB_PORT"),
		"user":     os.Getenv("DB_USER"),
		"password": os.Getenv("DB_PASSWORD"),
		"dbname":   os.Getenv("DB_NAME"),
		"sslmode":  "disable",
	}

	tokens := make([]string, 0, len(config))
	for k, v := range config {
		if len(v) > 0 {
			token := fmt.Sprintf("%s=%s", k, v)
			tokens = append(tokens, token)
		}
	}

	dsn := strings.Join(tokens, " ")
	return postgres.Open(dsn), nil
}

func newGORMDatabase(i do.Injector) (*GORMWrapper, error) {

	dialector := do.MustInvoke[gorm.Dialector](i)
	logger := do.MustInvoke[*zap.Logger](i)

	gormLogger := zapgorm2.New(logger)

	db, err := gorm.Open(dialector, &gorm.Config{Logger: gormLogger})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&domain.Server{}, &domain.Endpoint{}); err != nil {
		return nil, err
	}

	return &GORMWrapper{db: db}, nil
}

func RegisterGORMDB(i do.Injector) {
	do.Provide(i, newPostgresDriver)
	do.Provide(i, newGORMDatabase)
}

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
