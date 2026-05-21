package config

import (
	"os"

	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"
)

func newTemporalOption(i do.Injector) (*temporalclient.Options, error) {
	host := os.Getenv("TEMPORAL_HOST")
	if host == "" {
		host = "localhost:7233"
	}
	return &temporalclient.Options{
		HostPort: host,
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
