package testcontainers

import (
	"context"
	"fmt"
	"os"
	"testing"

	"gorm.io/gorm"
)

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

type tableNamer interface{ TableName() string }

func TruncateTables(tb testing.TB, db *gorm.DB, models ...tableNamer) {

	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}

	for _, m := range models {
		query := fmt.Sprintf("TRUNCATE TABLE %s CASCADE", m.TableName())
		db.WithContext(tb.Context()).Exec(query)
	}
}
