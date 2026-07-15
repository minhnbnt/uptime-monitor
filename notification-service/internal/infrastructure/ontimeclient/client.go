package ontimeclient

import (
	"bytes"
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
			baseURL: cfg.OntimeService.Addr,
			client:  &http.Client{Timeout: 30 * time.Second},
		}, nil
	})
}

func (a *Client) GetServersOntimeForDates(ctx context.Context, servers []domain.Server, dates []time.Time) (map[uint][]domain.OntimeStats, error) {

	type item struct {
		ServerID uint      `json:"server_id"`
		Date     string    `json:"date"`
	}

	type request struct {
		Items []item `json:"items"`
	}

	reqBody := request{}
	for _, sv := range servers {
		for _, d := range dates {
			reqBody.Items = append(reqBody.Items, item{
				ServerID: sv.ID,
				Date:     d.Format("2006-01-02"),
			})
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/servers/ontime/batch", a.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	type statsResponse struct {
		ServerID uint    `json:"server_id"`
		Date     string `json:"date"`
		Stats    float64 `json:"stats"`
	}

	type batchResponse struct {
		Results []statsResponse `json:"results"`
	}

	var batch batchResponse
	if err := json.NewDecoder(resp.Body).Decode(&batch); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	result := make(map[uint][]domain.OntimeStats)
	for _, s := range batch.Results {
		parsed, err := time.Parse("2006-01-02", s.Date)
		if err != nil {
			continue
		}
		result[s.ServerID] = append(result[s.ServerID], domain.OntimeStats{
			Date:  parsed,
			Stats: s.Stats,
		})
	}

	return result, nil
}
