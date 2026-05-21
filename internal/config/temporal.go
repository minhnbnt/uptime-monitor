package config

import (
	"os"

	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"
)

type TemporalConfig struct {
	Host       string
	TaskQueue  string
	Workflow   string
}

func newTemporalConfig(i do.Injector) (*TemporalConfig, error) {
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
	return &TemporalConfig{
		Host:      host,
		TaskQueue: taskQueue,
		Workflow:  workflow,
	}, nil
}

func RegisterTemporalConfig(i do.Injector) {
	do.Provide(i, newTemporalConfig)
}

func newTemporalOption(i do.Injector) (*temporalclient.Options, error) {
	cfg := do.MustInvoke[*TemporalConfig](i)
	return &temporalclient.Options{
		HostPort: cfg.Host,
	}, nil
}

func newTemporalClient(i do.Injector) (*TemporalClientWrapper, error) {

	option := do.MustInvoke[*temporalclient.Options](i)

	client, err := temporalclient.Dial(*option)
	if err != nil {
		return nil, err
	}

	return &TemporalClientWrapper{client: client}, nil
}

func RegisterTemporalClient(i do.Injector) {
	do.Provide(i, newTemporalOption)
	do.Provide(i, newTemporalClient)
}

type TemporalClientWrapper struct {
	client temporalclient.Client
}

func (tcw *TemporalClientWrapper) Shutdown() {
	tcw.client.Close()
}

func (tcw *TemporalClientWrapper) GetClient() temporalclient.Client {
	return tcw.client
}
