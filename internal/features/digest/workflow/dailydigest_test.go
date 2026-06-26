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

func runDigestWorker(tb testing.TB, digestFn any) temporalclient.Client {
	tb.Helper()

	c, err := temporalclient.Dial(temporalclient.Options{HostPort: temporalAddr})
	if err != nil {
		tb.Fatalf("dial: %v", err)
	}

	w := worker.New(c, "test-digest-queue", worker.Options{})
	w.RegisterActivityWithOptions(digestFn, activity.RegisterOptions{Name: "SendUserDigest"})
	w.RegisterWorkflow(SendReportWorkflow)

	if err := w.Start(); err != nil {
		tb.Fatalf("start worker: %v", err)
	}

	tb.Cleanup(func() {
		w.Stop()
		c.Close()
	})

	return c
}

func TestSendReportWorkflow_Success(t *testing.T) {
	skipIfShort(t)

	var capturedUserID uint
	c := runDigestWorker(t,
		func(ctx context.Context, userID uint) error {
			capturedUserID = userID
			return nil
		},
	)

	run, err := c.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "test-digest-queue",
		ID:                 fmt.Sprintf("test-digest-success-%d", time.Now().UnixNano()),
		WorkflowRunTimeout: 30 * time.Second,
	}, SendReportWorkflow, uint(1))
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}

	if err := run.Get(t.Context(), nil); err != nil {
		t.Fatalf("workflow error: %v", err)
	}
	if capturedUserID != 1 {
		t.Errorf("userID = %d, want 1", capturedUserID)
	}
}

func TestSendReportWorkflow_ActivityError(t *testing.T) {
	skipIfShort(t)

	c := runDigestWorker(t,
		func(ctx context.Context, userID uint) error {
			return errors.New("digest failed")
		},
	)

	run, err := c.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "test-digest-queue",
		ID:                 fmt.Sprintf("test-digest-err-%d", time.Now().UnixNano()),
		WorkflowRunTimeout: 30 * time.Second,
	}, SendReportWorkflow, uint(1))
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}

	if err := run.Get(t.Context(), nil); err == nil {
		t.Fatal("expected error from SendUserDigest activity")
	}
}
