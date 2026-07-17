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

func SkipIfShort(tb testing.TB) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
}

func StartRedisAddr(ctx context.Context) (tc.Container, string) {
	req := tc.ContainerRequest{
		Image:        "valkey/valkey:9-alpine",
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
