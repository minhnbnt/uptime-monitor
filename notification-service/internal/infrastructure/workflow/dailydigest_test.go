package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

var errTestFailed = errors.New("activity failed")

var testUserID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

func TestSendReportWorkflow_PassesFromDateToActivity(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	fromDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	var actualUserID uuid.UUID
	var actualFromDate time.Time

	env.RegisterActivityWithOptions(
		func(_ context.Context, userID uuid.UUID, from time.Time) error {
			actualUserID = userID
			actualFromDate = from
			return nil
		},
		activity.RegisterOptions{Name: "SendUserDigest"},
	)

	env.ExecuteWorkflow(SendReportWorkflow, testUserID, fromDate)

	require.True(t, env.IsWorkflowCompleted())
	require.NoError(t, env.GetWorkflowError())
	require.Equal(t, testUserID, actualUserID)
	require.Equal(t, fromDate, actualFromDate)
}

func TestSendReportWorkflow_FailsOnActivityError(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	fromDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	env.RegisterActivityWithOptions(
		func(_ context.Context, _ uuid.UUID, _ time.Time) error {
			return errTestFailed
		},
		activity.RegisterOptions{Name: "SendUserDigest"},
	)

	env.ExecuteWorkflow(SendReportWorkflow, testUserID, fromDate)

	require.True(t, env.IsWorkflowCompleted())
	require.Error(t, env.GetWorkflowError())
}
