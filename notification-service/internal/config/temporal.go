package config

import (
	"fmt"
	"sync"

	"github.com/samber/do/v2"
	"go.temporal.io/sdk/client"
)

type TemporalClientWrapper struct {
	client client.Client
	mu     sync.Mutex
}

func (w *TemporalClientWrapper) GetClient() client.Client {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.client
}

func RegisterTemporalClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*TemporalClientWrapper, error) {
		cfg := do.MustInvoke[*Config](i)

		c, err := client.Dial(client.Options{
			HostPort: cfg.Temporal.HostPort,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to Temporal: %w", err)
		}

		return &TemporalClientWrapper{client: c}, nil
	})
}
