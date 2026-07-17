package infrastructure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type PingWorker struct {
	httpClient *http.Client
}

func RegisterPingWorker(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*PingWorker, error) {
		return &PingWorker{httpClient: &http.Client{Timeout: 30 * time.Second}}, nil
	})
}

func (p *PingWorker) Ping(ctx context.Context, ep *domain.Endpoint) (statusCode int, err error) {

	timeout := ep.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, ep.Method, ep.URL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("failed to do request: %w", err)
	}

	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	return response.StatusCode, nil
}
