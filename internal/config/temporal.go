package config

import (
	"log/slog"

	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"
)

type TemporalClientWrapper struct {
	client temporalclient.Client
}

func newClientOption(i do.Injector) (*temporalclient.Options, error) {

	cfg := do.MustInvoke[*Config](i)
	log := do.MustInvoke[*slog.Logger](i)

	return &temporalclient.Options{
		HostPort: cfg.Temporal.Host,
		Logger:   log,
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
	do.Provide(i, newClientOption)
	do.Provide(i, newTemporalClient)
}

func (cw *TemporalClientWrapper) Shutdown() {
	cw.client.Close()
}

func (cw *TemporalClientWrapper) GetClient() temporalclient.Client {
	return cw.client
}
