package testcontainers

import (
	"context"
	"fmt"
	"os"
	"testing"
)

func SkipIfShort(tb testing.TB) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
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
