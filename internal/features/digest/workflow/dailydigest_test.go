package workflow

import (
	"context"
	"errors"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

func TestSendReportWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	var capturedUserID uint
	env.RegisterActivityWithOptions(
		func(ctx context.Context, userID uint) error {
			capturedUserID = userID
			return nil
		},
		activity.RegisterOptions{Name: "SendUserDigest"},
	)

	env.ExecuteWorkflow(SendReportWorkflow, uint(1))

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedUserID != 1 {
		t.Errorf("userID = %d, want 1", capturedUserID)
	}
}

func TestSendReportWorkflow_ActivityError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.RegisterActivityWithOptions(
		func(ctx context.Context, userID uint) error {
			return errors.New("digest failed")
		},
		activity.RegisterOptions{Name: "SendUserDigest"},
	)

	env.ExecuteWorkflow(SendReportWorkflow, uint(1))

	if !env.IsWorkflowCompleted() {
		t.Fatal("workflow did not complete")
	}
	if err := env.GetWorkflowError(); err == nil {
		t.Fatal("expected error from SendUserDigest activity")
	}
}
