package repository

import "github.com/samber/do/v2"

type SchedulerBackend string

const (
	SchedulerBackendTemporal SchedulerBackend = "temporal"
	SchedulerBackendRedis    SchedulerBackend = "redis"
)

func RegisterSchedulerBackend(i do.Injector, backend SchedulerBackend) {
	do.Provide(i, func(i do.Injector) (*SchedulerBackend, error) {
		return &backend, nil
	})
}
