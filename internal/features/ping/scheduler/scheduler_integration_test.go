package scheduler

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var testRedis *redis.Client
var testDB *gorm.DB
var testDSN string

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()

		redisContainer, client := testcontainers.StartRedis(ctx)
		defer func() { _ = redisContainer.Terminate(ctx) }()
		testRedis = client

		pgContainer, dsn := testcontainers.StartPostgres(ctx, testcontainers.PostgresConfig{
			Image:    testcontainers.DefaultParadedbImage,
			User:     "postgres",
			Password: "postgres",
			DBName:   "uptime_monitor",
		})
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

func seedServer(tb testing.TB, id uint) {
	tb.Helper()
	testDB.Create(&domain.Server{
		Model:       gorm.Model{ID: id},
		Name:        fmt.Sprintf("server-%d", id),
		CreatedByID: 1,
	})
}

func seedEndpoint(tb testing.TB, id, serverID uint) {
	tb.Helper()
	testDB.Create(&domain.Endpoint{
		Model:    gorm.Model{ID: id},
		ServerID: serverID,
		URL:      fmt.Sprintf("https://example-%d.com", id),
		Method:   "GET",
	})
}
