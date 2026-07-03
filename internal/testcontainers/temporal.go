package testcontainers

import (
	"context"
	"fmt"
	"os"
	"time"

	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

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
