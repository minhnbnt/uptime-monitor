package testcontainers

import (
	"context"
	"fmt"
	"os"
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
