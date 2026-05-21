package infrashtructure

import (
	"context"
	"fmt"
	"net/http"

	"github.com/samber/do/v2"
)

type PingWorker struct {
	httpClient *http.Client
}

func RegisterPingWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingWorker, error) {
		return &PingWorker{httpClient: http.DefaultClient}, nil
	})
}

func (p *PingWorker) Ping(ctx context.Context, method, url string) (statusCode int, err error) {

	request, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("failed to do request: %w", err)
	}

	return response.StatusCode, response.Body.Close()
}
