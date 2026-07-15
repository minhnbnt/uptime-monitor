package serverclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type Client struct {
	baseURL string
	client  *http.Client
}

func RegisterClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Client, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return &Client{
			baseURL: cfg.ServerService.Addr,
			client:  &http.Client{Timeout: 30 * time.Second},
		}, nil
	})
}

func (a *Client) List(ctx context.Context, createdByID uint, limit, offset int) ([]domain.Server, error) {

	url := fmt.Sprintf("%s/api/v1/servers?limit=%d&offset=%d", a.baseURL, limit, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", createdByID))

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	type serverResp struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}

	var servers []serverResp
	if err := json.NewDecoder(resp.Body).Decode(&servers); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := make([]domain.Server, len(servers))
	for i, s := range servers {
		result[i] = domain.Server{ID: s.ID, Name: s.Name}
	}
	return result, nil
}

func (a *Client) CountByStatus(ctx context.Context, createdByID uint) (total, online, offline int64, err error) {

	url := fmt.Sprintf("%s/api/v1/servers/count", a.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("X-User-ID", fmt.Sprintf("%d", createdByID))

	resp, err := a.client.Do(req)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, 0, 0, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	type countResp struct {
		Total   int64 `json:"total"`
		Online  int64 `json:"online"`
		Offline int64 `json:"offline"`
	}

	var c countResp
	if err := json.NewDecoder(resp.Body).Decode(&c); err != nil {
		return 0, 0, 0, fmt.Errorf("decode response: %w", err)
	}

	return c.Total, c.Online, c.Offline, nil
}
