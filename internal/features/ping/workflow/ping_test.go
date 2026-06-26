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

var temporalAddr string

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
		temporalAddr = fmt.Sprintf("%s:%s", host, port.Port())
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

func runPingWorker(tb testing.TB, pingFn, recordFn any) temporalclient.Client {
	tb.Helper()

	c, err := temporalclient.Dial(temporalclient.Options{HostPort: temporalAddr})
	if err != nil {
		tb.Fatalf("dial: %v", err)
	}

	w := worker.New(c, "test-ping-queue", worker.Options{})
	w.RegisterActivityWithOptions(pingFn, activity.RegisterOptions{Name: "Ping"})
	w.RegisterActivityWithOptions(recordFn, activity.RegisterOptions{Name: "Record"})
	w.RegisterWorkflow(PingWorkflow)

	if err := w.Start(); err != nil {
		tb.Fatalf("start worker: %v", err)
	}

	tb.Cleanup(func() {
		w.Stop()
		c.Close()
	})

	return c
}

func TestPingWorkflow_Success(t *testing.T) {
	skipIfShort(t)

	var capturedMethod string
	var recordedEvent *domain.ServerEvent

	c := runPingWorker(t,
		func(ctx context.Context, method, url string) (int, error) {
			capturedMethod = method
			return 200, nil
		},
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
	)

	run, err := c.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "test-ping-queue",
		ID:                 fmt.Sprintf("test-ping-success-%d", time.Now().UnixNano()),
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

	var recordedEvent *domain.ServerEvent

	c := runPingWorker(t,
		func(ctx context.Context, method, url string) (int, error) {
			return 0, errors.New("connection refused")
		},
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
	)

	run, err := c.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "test-ping-queue",
		ID:                 fmt.Sprintf("test-ping-off-pingerr-%d", time.Now().UnixNano()),
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

	var recordedEvent *domain.ServerEvent

	c := runPingWorker(t,
		func(ctx context.Context, method, url string) (int, error) {
			return 500, nil
		},
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
	)

	run, err := c.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "test-ping-queue",
		ID:                 fmt.Sprintf("test-ping-off-code-%d", time.Now().UnixNano()),
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

	c := runPingWorker(t,
		func(ctx context.Context, method, url string) (int, error) {
			return 200, nil
		},
		func(ctx context.Context, event *domain.ServerEvent) error {
			return errors.New("db error")
		},
	)

	run, err := c.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "test-ping-queue",
		ID:                 fmt.Sprintf("test-ping-recorderr-%d", time.Now().UnixNano()),
		WorkflowRunTimeout: 30 * time.Second,
	}, PingWorkflow, uint(1), "GET", "https://example.com", 200)
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}

	if err := run.Get(t.Context(), nil); err == nil {
		t.Fatal("expected error from record activity")
	}
}
