package testcontainers

import tc "github.com/testcontainers/testcontainers-go"

type Container = tc.Container

type PostgresConfig struct {
	Image    string
	User     string
	Password string
	DBName   string
}

const (
	defaultPostgresImage = DefaultParadedbImage
	defaultPostgresUser  = "test"
	defaultPostgresPass  = "test"
	defaultPostgresDB    = "uptime_test"

	defaultRedisImage = "valkey/valkey:9-alpine"

	defaultTemporalImage = "temporalio/temporal:1.7.2"

	DefaultParadedbImage = "paradedb/paradedb:pg18"
)

func ParadedbConfig() PostgresConfig {
	return PostgresConfig{Image: DefaultParadedbImage}
}

func apply[T comparable](v T, def T) T {
	var zero T
	if v != zero {
		return v
	}
	return def
}
