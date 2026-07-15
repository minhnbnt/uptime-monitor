package testcontainers

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
)

func StartPostgres(ctx context.Context) (tc.Container, string) {
	req := tc.ContainerRequest{
		Image:        "postgres:18-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "testdb",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(120 * time.Second),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start postgres container: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())
	return container, dsn
}

func CreateTestDB(tb testing.TB, dsn string) *gorm.DB {
	tb.Helper()

	u, err := url.Parse(dsn)
	if err != nil {
		tb.Fatalf("parse dsn: %v", err)
	}

	name := sanitizeDBName(tb.Name())
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	dbName := fmt.Sprintf("t_%s_%s", name, suffix)
	if len(dbName) > 63 {
		dbName = dbName[:63]
	}

	u.Path = "/testdb"
	pgDB, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{})
	if err != nil {
		tb.Fatalf("connect to postgres: %v", err)
	}

	if err := pgDB.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)).Error; err != nil {
		_ = closeDB(pgDB)
		tb.Fatalf("create database %s: %v", dbName, err)
	}
	_ = closeDB(pgDB)

	u.Path = "/" + dbName
	testDB, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{TranslateError: true})
	if err != nil {
		tb.Fatalf("connect to test db %s: %v", dbName, err)
	}

	if err := testDB.AutoMigrate(&domain.ServerEvent{}); err != nil {
		tb.Fatalf("auto migrate: %v", err)
	}

	tb.Cleanup(func() {
		if sqlDB, err := testDB.DB(); err == nil {
			sqlDB.Close()
		}

		u.Path = "/testdb"
		cleanupDB, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{})
		if err != nil {
			tb.Logf("cleanup: connect to postgres: %v", err)
			return
		}
		defer func() {
			if sqlDB, err := cleanupDB.DB(); err == nil {
				sqlDB.Close()
			}
		}()

		if err := cleanupDB.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, dbName)).Error; err != nil {
			tb.Logf("drop database %s: %v", dbName, err)
		}
	})

	return testDB
}

func sanitizeDBName(name string) string {
	name = strings.ToLower(name)
	name = strings.NewReplacer("/", "_", "#", "_", " ", "_", ".", "_", "-", "_").Replace(name)
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}
	return strings.Trim(name, "_")
}

func closeDB(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
