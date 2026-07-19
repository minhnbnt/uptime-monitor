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

const maxBodyBytes = 1 << 20 // 1MB

type Response struct {
	StatusCode int
	Body       string
}

type PingWorker struct {
	httpClient *http.Client
}

func RegisterPingWorker(i do.Injector) {
	do.Provide(i, func(_ do.Injector) (*PingWorker, error) {
		return &PingWorker{httpClient: &http.Client{Timeout: 30 * time.Second}}, nil
	})
}

func (p *PingWorker) Ping(ctx context.Context, ep *domain.Endpoint) (*Response, error) {

	timeout := ep.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, ep.Method, ep.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to read body: %w", err)
	}

	return &Response{StatusCode: response.StatusCode, Body: string(bodyBytes)}, nil
}
