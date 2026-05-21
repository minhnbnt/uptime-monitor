package infrashtructure

import (
	"fmt"
	"net/http"
)

type PingWorker struct {
	httpClient *http.Client
}

func (p *PingWorker) Ping(method, url string) (statusCode int, err error) {

	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("failed to do request: %w", err)
	}

	return response.StatusCode, response.Body.Close()
}
