package temporal

import (
	"github.com/samber/do/v2"
	temporalclient "go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

type ClientWrapper struct {
	client temporalclient.Client
}

func newClientOption(i do.Injector) (*temporalclient.Options, error) {

	cfg := do.MustInvoke[*Config](i)

	logger := do.MustInvoke[*zap.Logger](i)
	logger = logger.WithOptions(zap.AddCallerSkip(1))

	return &temporalclient.Options{
		HostPort: cfg.Host,
		Logger:   &TemporalLogger{logger: logger.Sugar()},
	}, nil
}

func newClient(i do.Injector) (*ClientWrapper, error) {

	option := do.MustInvoke[*temporalclient.Options](i)

	client, err := temporalclient.Dial(*option)
	if err != nil {
		return nil, err
	}

	return &ClientWrapper{client: client}, nil
}

func RegisterClient(i do.Injector) {
	do.Provide(i, newClientOption)
	do.Provide(i, newClient)
}

func (cw *ClientWrapper) Shutdown() {
	cw.client.Close()
}

func (cw *ClientWrapper) GetClient() temporalclient.Client {
	return cw.client
}
