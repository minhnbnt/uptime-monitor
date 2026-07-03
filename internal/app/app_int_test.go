package app

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/samber/do/v2"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/minhnbnt/uptime-monitor/internal/features/auth/handler"
	digestHandler "github.com/minhnbnt/uptime-monitor/internal/features/digest/handler"
	importerHandler "github.com/minhnbnt/uptime-monitor/internal/features/importer/handler"
	notificationHandler "github.com/minhnbnt/uptime-monitor/internal/features/notification/handler"
	ontimeHandler "github.com/minhnbnt/uptime-monitor/internal/features/ontime/handler"
	pingHandler "github.com/minhnbnt/uptime-monitor/internal/features/ping/handler"
	serverHandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
	"github.com/minhnbnt/uptime-monitor/internal/server"
)

var (
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	temporalCont   testcontainers.Container
)

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()

		pgContainer = startPostgres(ctx)
		redisContainer, _ = startRedis(ctx)
		temporalCont = startTemporal(ctx)

		pgHost, pgPort := getHostPort(ctx, pgContainer, "5432")
		redisHost, redisPort := getHostPort(ctx, redisContainer, "6379")
		temporalHost, temporalPort := getHostPort(ctx, temporalCont, "7233")

		os.Setenv("DB_HOST", pgHost)
		os.Setenv("DB_PORT", pgPort)
		os.Setenv("DB_USER", "postgres")
		os.Setenv("DB_PASSWORD", "postgres")
		os.Setenv("DB_NAME", "uptime_monitor")

		os.Setenv("REDIS_ADDR", redisHost+":"+redisPort)

		os.Setenv("TEMPORAL_HOST", temporalHost+":"+temporalPort)

		os.Setenv("JWT_KEY", "test-key-for-wiring")
	}

	code := m.Run()

	if !testing.Short() {
		os.Unsetenv("DB_HOST")
		os.Unsetenv("DB_PORT")
		os.Unsetenv("DB_USER")
		os.Unsetenv("DB_PASSWORD")
		os.Unsetenv("DB_NAME")
		os.Unsetenv("REDIS_ADDR")
		os.Unsetenv("TEMPORAL_HOST")
		os.Unsetenv("JWT_KEY")

		ctx := context.Background()
		if pgContainer != nil {
			_ = pgContainer.Terminate(ctx)
		}
		if redisContainer != nil {
			_ = redisContainer.Terminate(ctx)
		}
		if temporalCont != nil {
			_ = temporalCont.Terminate(ctx)
		}
	}

	os.Exit(code)
}

func startPostgres(ctx context.Context) testcontainers.Container {
	req := testcontainers.ContainerRequest{
		Image:        "paradedb/paradedb:pg17",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "postgres",
			"POSTGRES_PASSWORD": "postgres",
			"POSTGRES_DB":       "uptime_monitor",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(120 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req, Started: true,
	})
	if err != nil {
		log.Fatalf("start postgres: %v", err)
	}
	return c
}

func startRedis(_ context.Context) (testcontainers.Container, any) {
	req := testcontainers.ContainerRequest{
		Image:        "redis:8-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForLog("Ready to accept connections tcp").WithStartupTimeout(60 * time.Second),
	}
	c, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
		ContainerRequest: req, Started: true,
	})
	if err != nil {
		log.Fatalf("start redis: %v", err)
	}
	return c, nil
}

func startTemporal(ctx context.Context) testcontainers.Container {
	req := testcontainers.ContainerRequest{
		Image:        "temporalio/temporal:1.7.2",
		ExposedPorts: []string{"7233/tcp"},
		Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0"},
		WaitingFor:   wait.ForListeningPort("7233/tcp").WithStartupTimeout(120 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req, Started: true,
	})
	if err != nil {
		log.Fatalf("start temporal: %v", err)
	}
	return c
}

func getHostPort(ctx context.Context, c testcontainers.Container, port string) (string, string) {
	host, err := c.Host(ctx)
	if err != nil {
		log.Fatalf("container host: %v", err)
	}
	mapped, err := c.MappedPort(ctx, port)
	if err != nil {
		log.Fatalf("container port: %v", err)
	}
	return host, mapped.Port()
}

func TestApp_Wiring_CompositeHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	injector := do.New()
	RegisterPackages(injector, "", true)

	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()

	ch := do.MustInvoke[*server.CompositeHandler](injector)
	if ch == nil {
		t.Fatal("CompositeHandler is nil")
	}

	_ = do.MustInvoke[*serverHandler.ServerHandler](injector)
	_ = do.MustInvoke[*serverHandler.EndpointHandler](injector)
	_ = do.MustInvoke[*handler.AuthHandler](injector)
	_ = do.MustInvoke[*importerHandler.ImportHandler](injector)
	_ = do.MustInvoke[*ontimeHandler.OntimeHandler](injector)
	_ = do.MustInvoke[*notificationHandler.NotificationHandler](injector)

	_ = do.MustInvoke[*pingHandler.ZSetWorkerRunner](injector)
	_ = do.MustInvoke[*digestHandler.DigestWorkerRunner](injector)

	_ = ctx
}

func TestApp_RunAllGoroutines_ContextCancel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	injector := do.New()
	RegisterPackages(injector, "", true)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	wg := sync.WaitGroup{}
	errs := make(chan error, 4)

	run := func(name string, fn func()) {
		wg.Go(func() {

			defer func() {
				if r := recover(); r != nil {
					errs <- fmt.Errorf("%s panicked: %v\n%s", name, r, debug.Stack())
				}
			}()

			fn()
		})
	}

	run("RunPingWorker", func() { RunPingWorker(ctx, injector) })
	run("RunDigestWorker", func() { RunDigestWorker(ctx, injector) })

	run("RunWebServer", func() {

		listenConfig := net.ListenConfig{}
		l, err := listenConfig.Listen(context.Background(), "tcp", ":8080")
		if err != nil {
			return
		}

		l.Close()

		RunWebServer(ctx, injector, true)
	})

	time.Sleep(2 * time.Second)

	cancel()

	doneCtx, cancel := context.WithCancel(t.Context())
	go func() {
		wg.Wait()
		cancel()
		_, _ = injector.ShutdownOnSignalsWithContext(ctx)
	}()

	select {
	case <-doneCtx.Done():
	case <-time.After(15 * time.Second):
		t.Fatal("goroutines did not exit within 15s after cancel")
	}

	close(errs)
	for err := range errs {
		t.Error(err)
	}
}
