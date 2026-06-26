package workflow

import (
	"context"
	"errors"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

func TestPingWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	var capturedMethod string
	env.RegisterActivityWithOptions(
		func(ctx context.Context, method, url string) (int, error) {
			capturedMethod = method
			_ = url
			return 200, nil
		},
		activity.RegisterOptions{Name: "Ping"},
	)

	var recordedEvent *domain.ServerEvent
	env.RegisterActivityWithOptions(
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
		activity.RegisterOptions{Name: "Record"},
	)

	env.ExecuteWorkflow(PingWorkflow, uint(1), "GET", "https://example.com", 200)

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
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
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func(ctx context.Context, method, url string) (int, error) {
			return 0, errors.New("connection refused")
		},
		activity.RegisterOptions{Name: "Ping"},
	)

	var recordedEvent *domain.ServerEvent
	env.RegisterActivityWithOptions(
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
		activity.RegisterOptions{Name: "Record"},
	)

	env.ExecuteWorkflow(PingWorkflow, uint(1), "GET", "https://example.com", 200)

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recordedEvent == nil {
		t.Fatal("expected Record activity to be called")
	}
	if recordedEvent.Status != domain.StatusOff {
		t.Errorf("Status = %q, want %q", recordedEvent.Status, domain.StatusOff)
	}
}

func TestPingWorkflow_StatusOffOnCodeMismatch(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func(ctx context.Context, method, url string) (int, error) {
			return 500, nil
		},
		activity.RegisterOptions{Name: "Ping"},
	)

	var recordedEvent *domain.ServerEvent
	env.RegisterActivityWithOptions(
		func(ctx context.Context, event *domain.ServerEvent) error {
			recordedEvent = event
			return nil
		},
		activity.RegisterOptions{Name: "Record"},
	)

	env.ExecuteWorkflow(PingWorkflow, uint(1), "GET", "https://example.com", 200)

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if recordedEvent == nil {
		t.Fatal("expected Record activity")
	}
	if recordedEvent.Status != domain.StatusOff {
		t.Errorf("Status = %q, want %q", recordedEvent.Status, domain.StatusOff)
	}
}

func TestPingWorkflow_RecordActivityError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func(ctx context.Context, method, url string) (int, error) {
			return 200, nil
		},
		activity.RegisterOptions{Name: "Ping"},
	)
	env.RegisterActivityWithOptions(
		func(ctx context.Context, event *domain.ServerEvent) error {
			return errors.New("db error")
		},
		activity.RegisterOptions{Name: "Record"},
	)

	env.ExecuteWorkflow(PingWorkflow, uint(1), "GET", "https://example.com", 200)

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("expected error from record activity")
	}
}
