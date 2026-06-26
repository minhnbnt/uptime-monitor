package workflow

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.temporal.io/sdk/activity"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

var temporalClient temporalclient.Client

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()
		c := startTemporal(ctx)
		defer func() { _ = c.Terminate(ctx) }()

		host, err := c.Host(ctx)
		if err != nil {
			log.Fatalf("host: %v", err)
		}
		port, err := c.MappedPort(ctx, "7233")
		if err != nil {
			log.Fatalf("port: %v", err)
		}
		addr := fmt.Sprintf("%s:%s", host, port.Port())

		client, err := temporalclient.Dial(temporalclient.Options{HostPort: addr})
		if err != nil {
			log.Fatalf("dial: %v", err)
		}
		temporalClient = client
	}
	os.Exit(m.Run())
}

func startTemporal(ctx context.Context) testcontainers.Container {
	req := testcontainers.ContainerRequest{
		Image:        "temporalio/temporal:1.7.2",
		ExposedPorts: []string{"7233/tcp"},
		Cmd:          []string{"server", "start-dev", "--ip", "0.0.0.0"},
		WaitingFor:   wait.ForListeningPort("7233/tcp").WithStartupTimeout(90 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Fatalf("start temporal: %v", err)
	}
	return c
}

func skipIfShort(tb testing.TB) {
	tb.Helper()
	if testing.Short() {
		tb.Skip("skipping integration test")
	}
}

func TestPingWorkflow_Success(t *testing.T) {
	skipIfShort(t)

	w := worker.New(temporalClient, "ping-test", worker.Options{})
	w.RegisterWorkflow(PingWorkflow)

	var capturedMethod string
	w.RegisterActivityWithOptions(
		func(ctx context.Context, method, url string) (int, error) {
			capturedMethod = method
			return 200, nil
		},
		activity.RegisterOptions{Name: "Ping"},
	)

	var recordedEvent *domain.ServerEvent
	w.RegisterActivityWithOptions(
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
		activity.RegisterOptions{Name: "Record"},
	)

	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	run, err := temporalClient.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "ping-test",
		ID:                 fmt.Sprintf("ping-test-%d", time.Now().UnixNano()),
		WorkflowRunTimeout: 30 * time.Second,
	}, PingWorkflow, uint(1), "GET", "https://example.com", 200)
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if err := run.Get(t.Context(), nil); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	if recordedEvent == nil {
		t.Fatal("expected Record activity to be called")
	}
	if recordedEvent.Status != domain.StatusOn {
		t.Errorf("Status = %q, want %q", recordedEvent.Status, domain.StatusOn)
	}
	if capturedMethod != "GET" {
		t.Errorf("method = %q", capturedMethod)
	}
}

func TestPingWorkflow_StatusOffOnPingError(t *testing.T) {
	skipIfShort(t)

	w := worker.New(temporalClient, "ping-test", worker.Options{})
	w.RegisterWorkflow(PingWorkflow)

	w.RegisterActivityWithOptions(
		func(ctx context.Context, method, url string) (int, error) {
			return 0, errors.New("connection refused")
		},
		activity.RegisterOptions{Name: "Ping"},
	)

	var recordedEvent *domain.ServerEvent
	w.RegisterActivityWithOptions(
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
		activity.RegisterOptions{Name: "Record"},
	)

	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	run, err := temporalClient.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "ping-test",
		ID:                 fmt.Sprintf("ping-test-%d", time.Now().UnixNano()),
		WorkflowRunTimeout: 30 * time.Second,
	}, PingWorkflow, uint(1), "GET", "https://example.com", 200)
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if err := run.Get(t.Context(), nil); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	if recordedEvent == nil {
		t.Fatal("expected Record activity to be called")
	}
	if recordedEvent.Status != domain.StatusOff {
		t.Errorf("Status = %q, want %q", recordedEvent.Status, domain.StatusOff)
	}
}

func TestPingWorkflow_StatusOffOnCodeMismatch(t *testing.T) {
	skipIfShort(t)

	w := worker.New(temporalClient, "ping-test", worker.Options{})
	w.RegisterWorkflow(PingWorkflow)

	w.RegisterActivityWithOptions(
		func(ctx context.Context, method, url string) (int, error) {
			return 500, nil
		},
		activity.RegisterOptions{Name: "Ping"},
	)

	var recordedEvent *domain.ServerEvent
	w.RegisterActivityWithOptions(
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
		activity.RegisterOptions{Name: "Record"},
	)

	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	run, err := temporalClient.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "ping-test",
		ID:                 fmt.Sprintf("ping-test-%d", time.Now().UnixNano()),
		WorkflowRunTimeout: 30 * time.Second,
	}, PingWorkflow, uint(1), "GET", "https://example.com", 200)
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if err := run.Get(t.Context(), nil); err != nil {
		t.Fatalf("workflow error: %v", err)
	}

	if recordedEvent == nil {
		t.Fatal("expected Record activity")
	}
	if recordedEvent.Status != domain.StatusOff {
		t.Errorf("Status = %q, want %q", recordedEvent.Status, domain.StatusOff)
	}
}

func TestPingWorkflow_RecordActivityError(t *testing.T) {
	skipIfShort(t)

	w := worker.New(temporalClient, "ping-test", worker.Options{})
	w.RegisterWorkflow(PingWorkflow)

	w.RegisterActivityWithOptions(
		func(ctx context.Context, method, url string) (int, error) {
			return 200, nil
		},
		activity.RegisterOptions{Name: "Ping"},
	)
	w.RegisterActivityWithOptions(
		func(ctx context.Context, event *domain.ServerEvent) error {
			return errors.New("db error")
		},
		activity.RegisterOptions{Name: "Record"},
	)

	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	run, err := temporalClient.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "ping-test",
		ID:                 fmt.Sprintf("ping-test-%d", time.Now().UnixNano()),
		WorkflowRunTimeout: 30 * time.Second,
	}, PingWorkflow, uint(1), "GET", "https://example.com", 200)
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if err := run.Get(t.Context(), nil); err == nil {
		t.Fatal("expected error from record activity")
	}
}
