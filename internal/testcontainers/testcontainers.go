package testcontainers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type Container = tc.Container

type PostgresConfig struct {
	Image    string
	User     string
	Password string
	DBName   string
}

func apply[T comparable](v T, def T) T {

	var zero T
	if v != zero {
		return v
	}

	return def
}

const (
	defaultPostgresImage = "postgres:17-alpine"
	defaultPostgresUser  = "test"
	defaultPostgresPass  = "test"
	defaultPostgresDB    = "uptime_test"

	defaultRedisImage = "redis:8-alpine"

	defaultTemporalImage = "temporalio/temporal:1.7.2"

	DefaultParadedbImage = "paradedb/paradedb:pg18"
)

func ParadedbConfig() PostgresConfig {
	return PostgresConfig{Image: DefaultParadedbImage}
}

func StartPostgres(ctx context.Context, cfg ...PostgresConfig) (Container, string) {
	c := PostgresConfig{}
	if len(cfg) > 0 {
		c = cfg[0]
	}

	user := apply(c.User, defaultPostgresUser)
	password := apply(c.Password, defaultPostgresPass)
	dbName := apply(c.DBName, defaultPostgresDB)
	image := apply(c.Image, defaultPostgresImage)

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

func StartRedis(ctx context.Context) (Container, *redis.Client) {
	req := tc.ContainerRequest{
		Image:        defaultRedisImage,
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForLog("Ready to accept connections tcp").
			WithStartupTimeout(60 * time.Second),
	}
	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start redis container: %v\n", err)
		os.Exit(1)
	}

	host, err := container.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	port, err := container.MappedPort(ctx, "6379")
	if err != nil {
		fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}

	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", host, port.Port()),
	})

	return container, client
}

func StartTemporal(ctx context.Context) (Container, string) {
	req := tc.ContainerRequest{
		Image:        defaultTemporalImage,
		ExposedPorts: []string{"7233/tcp"},
		Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0"},
		WaitingFor:   wait.ForListeningPort("7233/tcp").WithStartupTimeout(120 * time.Second),
	}
	c, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "start temporal container: %v\n", err)
		os.Exit(1)
	}

	host, err := c.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	port, err := c.MappedPort(ctx, "7233")
	if err != nil {
		fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	return c, addr
}

func ContainerHostPort(ctx context.Context, c Container, port string) (string, string) {
	host, err := c.Host(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container host: %v\n", err)
		os.Exit(1)
	}
	mapped, err := c.MappedPort(ctx, port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "container port: %v\n", err)
		os.Exit(1)
	}
	return host, mapped.Port()
}

func CleanRedis(tb testing.TB, client *redis.Client) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
	if err := client.FlushDB(context.Background()).Err(); err != nil {
		tb.Fatalf("flush redis: %v", err)
	}
}
