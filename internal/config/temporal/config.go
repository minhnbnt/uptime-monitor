package temporal

import (
	"os"

	"github.com/samber/do/v2"
)

type Config struct {
	Host         string
	TaskQueue    string
	WorkflowName string
}

func newConfig(i do.Injector) (*Config, error) {

	host := os.Getenv("TEMPORAL_HOST")
	if host == "" {
		host = "localhost:7233"
	}

	taskQueue := os.Getenv("TEMPORAL_TASK_QUEUE")
	if taskQueue == "" {
		taskQueue = "ping-task-queue"
	}

	workflow := os.Getenv("TEMPORAL_WORKFLOW_NAME")
	if workflow == "" {
		workflow = "ping-workflow"
	}

	return &Config{
		Host:         host,
		TaskQueue:    taskQueue,
		WorkflowName: workflow,
	}, nil
}

func RegisterConfig(i do.Injector) {
	do.Provide(i, newConfig)
}
