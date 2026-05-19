package config

import (
	"fmt"
	"os"

	"github.com/samber/do/v2"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

func newPostgresDriver(i do.Injector) (gorm.Dialector, error) {

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"),
	)

	return postgres.Open(dsn), nil
}

func newGORMDatabase(i do.Injector) (*gorm.DB, error) {

	dialector := do.MustInvoke[gorm.Dialector](i)
	logger := do.MustInvoke[*zap.Logger](i)

	gormLogger := zapgorm2.New(logger)

	return gorm.Open(dialector, &gorm.Config{Logger: gormLogger})
}

func RegisterGORMDB(i do.Injector) {
	do.Provide(i, newPostgresDriver)
	do.Provide(i, newGORMDatabase)
}
