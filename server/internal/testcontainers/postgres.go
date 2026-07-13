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

	"github.com/minhnbnt/uptime-monitor/internal/config"
)

func StartPostgres(ctx context.Context, cfg ...PostgresConfig) (Container, string) {
	c := PostgresConfig{}
	if len(cfg) > 0 {
		c = cfg[0]
	}

	user := apply(c.User, defaultPostgresUser)
	dbName := apply(c.DBName, defaultPostgresDB)
	image := apply(c.Image, defaultPostgresImage)
	password := apply(c.Password, defaultPostgresPass)

	req := tc.ContainerRequest{
		Image:        image,
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     user,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       dbName,
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

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port.Port(), dbName)

	return container, dsn
}

func OpenGORM(dsn string, opts ...gorm.Option) *gorm.DB {
	db, err := gorm.Open(postgres.Open(dsn), opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "gorm open: %v\n", err)
		os.Exit(1)
	}
	return db
}

func RunMigrations(db *gorm.DB) {
	if err := config.RunMigration(db); err != nil {
		fmt.Fprintf(os.Stderr, "run migration: %v\n", err)
		os.Exit(1)
	}
}

func CreateTestDB(tb testing.TB, dsn string, init ...func(*gorm.DB)) *gorm.DB {
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

	u.Path = "/postgres"
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

	RunMigrations(testDB)

	for _, fn := range init {
		fn(testDB)
	}

	tb.Cleanup(func() {
		if sqlDB, err := testDB.DB(); err == nil {
			sqlDB.Close()
		}

		u.Path = "/postgres"
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
