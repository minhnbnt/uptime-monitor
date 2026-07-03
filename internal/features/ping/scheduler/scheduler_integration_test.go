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

		testDB = testcontainers.OpenGORM(dsn)
		testcontainers.RunMigrations(testDB)

		if err := config.EnablePGSearch(testDB); err != nil {
			fmt.Fprintf(os.Stderr, "warning: pg_search not available: %v\n", err)
		}

		testDB.Create(&domain.User{
			Model:    gorm.Model{ID: 1},
			Email:    "test@test.com",
			Username: "test",
			Password: "x",
			Name:     "Test",
		})
	}
	os.Exit(m.Run())
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

func truncateTables(tb testing.TB) {

	tb.Helper()

	testcontainers.TruncateTables(
		tb, testDB,
		&domain.Server{},
		&domain.Endpoint{},
		&domain.ServerEvent{},
	)
}
