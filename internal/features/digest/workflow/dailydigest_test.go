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

	"go.temporal.io/sdk/activity"
	temporalclient "go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/minhnbnt/uptime-monitor/internal/testcontainers"
)

var temporalClient temporalclient.Client

func TestMain(m *testing.M) {
	flag.Parse()
	if !testing.Short() {
		ctx := context.Background()
		c, addr := testcontainers.StartTemporal(ctx)
		defer func() { _ = c.Terminate(ctx) }()

		client, err := temporalclient.Dial(temporalclient.Options{HostPort: addr})
		if err != nil {
			log.Fatalf("dial: %v", err)
		}
		temporalClient = client
	}
	os.Exit(m.Run())
}

func TestSendReportWorkflow_Success(t *testing.T) {
	testcontainers.SkipIfShort(t)

	taskQueue := testcontainers.NewTestTemporalTaskQueue(t)
	w := worker.New(temporalClient, taskQueue, worker.Options{})
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
		TaskQueue:          taskQueue,
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
	testcontainers.SkipIfShort(t)

	taskQueue := testcontainers.NewTestTemporalTaskQueue(t)
	w := worker.New(temporalClient, taskQueue, worker.Options{})
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
		TaskQueue:          taskQueue,
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
