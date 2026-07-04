package testcontainers

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var redisDBCounter atomic.Int32

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

func StartRedisAddr(ctx context.Context) (Container, string) {
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

	addr := fmt.Sprintf("%s:%s", host, port.Port())
	return container, addr
}

func CleanRedis(tb testing.TB, client *redis.Client) {

	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}

	if err := client.FlushDB(tb.Context()).Err(); err != nil {
		tb.Fatalf("flush redis: %v", err)
	}
}

func NewTestRedis(tb testing.TB, addr string) *redis.Client {
	tb.Helper()
	idx := redisDBCounter.Add(1) - 1
	if idx > 15 {
		tb.Fatalf("exhausted Redis database indices: %d (max 15)", idx)
	}
	client := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   int(idx),
	})
	tb.Cleanup(func() { _ = client.Close() })
	return client
}
