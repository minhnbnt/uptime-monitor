package temporal

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/config"
)

type Config struct {
	Host            string
	TaskQueue       string
	WorkflowName    string
	DigestTaskQueue string
}

func newConfig(cfg *config.Config) (*Config, error) {

	temporalCfg := cfg.Temporal

	return &Config{
		Host:            temporalCfg.Host,
		TaskQueue:       temporalCfg.TaskQueue,
		WorkflowName:    temporalCfg.WorkflowName,
		DigestTaskQueue: temporalCfg.DigestTaskQueue,
	}, nil
}

func RegisterConfig(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Config, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return newConfig(cfg)
	})
}
