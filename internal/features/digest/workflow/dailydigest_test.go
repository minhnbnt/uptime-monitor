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

func TestSendReportWorkflow_Success(t *testing.T) {
	skipIfShort(t)

	w := worker.New(temporalClient, "digest-test", worker.Options{})
	w.RegisterWorkflow(SendReportWorkflow)

	var capturedUserID uint
	w.RegisterActivityWithOptions(
		func(ctx context.Context, userID uint) error {
			capturedUserID = userID
			return nil
		},
		activity.RegisterOptions{Name: "SendUserDigest"},
	)

	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	run, err := temporalClient.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "digest-test",
		ID:                 fmt.Sprintf("digest-test-%d", time.Now().UnixNano()),
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

	w := worker.New(temporalClient, "digest-test", worker.Options{})
	w.RegisterWorkflow(SendReportWorkflow)

	w.RegisterActivityWithOptions(
		func(ctx context.Context, userID uint) error {
			return errors.New("digest failed")
		},
		activity.RegisterOptions{Name: "SendUserDigest"},
	)

	if err := w.Start(); err != nil {
		t.Fatalf("start worker: %v", err)
	}
	defer w.Stop()

	run, err := temporalClient.ExecuteWorkflow(t.Context(), temporalclient.StartWorkflowOptions{
		TaskQueue:          "digest-test",
		ID:                 fmt.Sprintf("digest-test-%d", time.Now().UnixNano()),
		WorkflowRunTimeout: 30 * time.Second,
	}, SendReportWorkflow, uint(1))
	if err != nil {
		t.Fatalf("execute workflow: %v", err)
	}
	if err := run.Get(t.Context(), nil); err == nil {
		t.Fatal("expected error from SendUserDigest activity")
	}
}
