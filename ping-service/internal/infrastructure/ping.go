package infrastructure

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/samber/do/v2"
)

const maxBodyBytes = 1 << 20 // 1MB

type Response struct {
	StatusCode int
	Body       string
}

type PingClient struct {
	httpClient *http.Client
}

func NewPingClient(httpClient *http.Client) *PingClient {
	return &PingClient{httpClient: httpClient}
}

func RegisterPingWorker(i do.Injector) {
	do.Provide(i, func(_ do.Injector) (*PingClient, error) {
		return NewPingClient(&http.Client{Timeout: 30 * time.Second}), nil
	})
}

func (p *PingClient) Ping(ctx context.Context, timeout time.Duration, method, url string) (*Response, error) {

	if timeout <= 0 || timeout > 10*time.Second {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, method, url, nil)
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
