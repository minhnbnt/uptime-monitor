package workflow

import (
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func SendReportWorkflow(ctx workflow.Context, userID uuid.UUID, fromDate time.Time) error {

	if fromDate.IsZero() {
		fromDate = workflow.Now(ctx).Add(-30 * 24 * time.Hour)
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	})

	return workflow.ExecuteActivity(ctx, "SendUserDigest", userID, fromDate).Get(ctx, nil)
}
