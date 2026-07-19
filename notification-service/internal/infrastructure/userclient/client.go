package userclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type Client struct {
	baseURL string
	client  *http.Client
	logger  *slog.Logger
}

func RegisterClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Client, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return &Client{
			baseURL: cfg.AuthService.Addr,
			client:  &http.Client{Timeout: 10 * time.Second},
			logger:  do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (a *Client) FindByID(ctx context.Context, id uint) (*domain.User, error) {

	url := fmt.Sprintf("%s/api/v1/auth/private/users/%d", a.baseURL, id)
	a.logger.Debug("userclient.FindByID: sending request",
		slog.String("url", url), slog.Uint64("user_id", uint64(id)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		a.logger.Error("userclient.FindByID: request failed",
			slog.String("url", url), slog.Any("error", err))
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		a.logger.Debug("userclient.FindByID: user not found", slog.Uint64("user_id", uint64(id)))
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		a.logger.Error("userclient.FindByID: unexpected status",
			slog.String("url", url), slog.Int("status", resp.StatusCode), slog.String("body", string(body)))
		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	type userResponse struct {
		ID       int    `json:"id"`
		Email    string `json:"email"`
		Username string `json:"username"`
		Name     string `json:"name"`
	}

	var u userResponse
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &domain.User{
		ID:       uint(u.ID),
		Email:    u.Email,
		Username: u.Username,
		Name:     u.Name,
	}, nil
}
