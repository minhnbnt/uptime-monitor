package app

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"runtime/debug"
	"sync"
	"testing"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/handler"
	digestHandler "github.com/minhnbnt/uptime-monitor/internal/features/digest/handler"
	importerHandler "github.com/minhnbnt/uptime-monitor/internal/features/importer/handler"
	notificationHandler "github.com/minhnbnt/uptime-monitor/internal/features/notification/handler"
	ontimeHandler "github.com/minhnbnt/uptime-monitor/internal/features/ontime/handler"
	pingHandler "github.com/minhnbnt/uptime-monitor/internal/features/ping/handler"
	serverHandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
	"github.com/minhnbnt/uptime-monitor/internal/server"
	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var (
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	temporalCont   testcontainers.Container

	testCfg *config.Config
)

func TestMain(m *testing.M) {

	flag.Parse()

	if !testing.Short() {

		ctx := context.Background()

		pgContainer, _ = testcontainers.StartPostgres(ctx, testcontainers.PostgresConfig{
			Image:    testcontainers.DefaultParadedbImage,
			User:     "postgres",
			Password: "postgres",
			DBName:   "uptime_monitor",
		})
		redisContainer, _ = testcontainers.StartRedis(ctx)
		temporalCont, _ = testcontainers.StartTemporal(ctx)

		pgHost, pgPort := testcontainers.ContainerHostPort(ctx, pgContainer, "5432")
		redisHost, redisPort := testcontainers.ContainerHostPort(ctx, redisContainer, "6379")
		temporalHost, temporalPort := testcontainers.ContainerHostPort(ctx, temporalCont, "7233")

		testCfg = &config.Config{
			DB: config.DBConfig{
				Host:     pgHost,
				Port:     pgPort,
				User:     "postgres",
				Password: "postgres",
				DBName:   "uptime_monitor",
			},
			Redis: config.RedisConfig{
				Addr: redisHost + ":" + redisPort,
			},
			JWT: config.JWTConfig{
				Key: "test-key-for-wiring",
			},
			Temporal: config.TemporalConfig{
				Host: temporalHost + ":" + temporalPort,
			},
			Scheduler: config.SchedulerCfg{Backend: "redis"},
			Token: config.TokenCfg{
				AccessTTL:     "15m",
				RefreshTTL:    "168h",
				AccessIssuer:  "uptime-monitor",
				RefreshIssuer: "uptime-monitor-refresh",
			},
			Argon2: config.Argon2Cfg{
				Memory:      16384,
				Iterations:  2,
				Parallelism: 1,
				SaltLength:  16,
				KeyLength:   32,
			},
			Log: config.LogConfig{Level: "info"},
			Mail: config.MailConfig{
				SMTPHost: "localhost",
				SMTPPort: 587,
			},
		}
	}

	code := m.Run()

	if !testing.Short() {

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

func newTestConfig() *config.Config {
	return &config.Config{
		DB: config.DBConfig{
			Host:     testCfg.DB.Host,
			Port:     testCfg.DB.Port,
			User:     testCfg.DB.User,
			Password: testCfg.DB.Password,
			DBName:   testCfg.DB.DBName,
		},
		Redis:     testCfg.Redis,
		JWT:       testCfg.JWT,
		Temporal:  testCfg.Temporal,
		Scheduler: testCfg.Scheduler,
		Token:     testCfg.Token,
		Argon2:    testCfg.Argon2,
		Log:       testCfg.Log,
		Mail:      testCfg.Mail,
	}
}

func TestApp_Wiring_CompositeHandler(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	injector := do.New()
	RegisterPackagesFromConfig(injector, newTestConfig(), true)

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
	RegisterPackagesFromConfig(injector, newTestConfig(), true)

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
		l, err := listenConfig.Listen(t.Context(), "tcp", ":8080")
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
