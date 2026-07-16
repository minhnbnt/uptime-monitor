package repository

import (
	"context"
	"flag"
	"os"
	"testing"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var testDB *gorm.DB
var testRedisAddr string
var testDSN string

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()

		redisContainer, addr := testcontainers.StartRedisAddr(ctx)
		defer func() { _ = redisContainer.Terminate(ctx) }()
		testRedisAddr = addr
		pgContainer, dsn := testcontainers.StartPostgres(ctx, testcontainers.ParadedbConfig())
		defer func() { _ = pgContainer.Terminate(ctx) }()
		testDSN = dsn
	}

	os.Exit(m.Run())
}

func initTestDB(tb testing.TB) *gorm.DB {
	tb.Helper()
	return testcontainers.CreateTestDB(tb, testDSN, func(db *gorm.DB) {
		if err := config.EnablePGSearch(db); err != nil {
			tb.Fatalf("enable pg_search: %v", err)
		}
		if err := db.Create(&domain.User{
			Model:    gorm.Model{ID: 1},
			Email:    "test@test.com",
			Username: "test",
			Password: "x",
			Name:     "Test",
		}).Error; err != nil {
			tb.Fatalf("seed test user: %v", err)
		}
	})
}
