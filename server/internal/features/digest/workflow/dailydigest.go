package workflow

import (
	"time"

	"go.temporal.io/sdk/workflow"
)

func SendReportWorkflow(ctx workflow.Context, userID uint) error {

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	})

	return workflow.ExecuteActivity(ctx, "SendUserDigest", userID).Get(ctx, nil)
}
