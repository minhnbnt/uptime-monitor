package userclient

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
)

type Client struct {
	baseURL      string
	serviceToken string
	client       *http.Client
	logger       *slog.Logger
}

func RegisterClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*Client, error) {
		cfg := do.MustInvoke[*config.Config](i)
		return &Client{
			baseURL:      cfg.Auth.Issuer,
			serviceToken: cfg.Auth.ServiceToken,
			client:       &http.Client{Timeout: 10 * time.Second},
			logger:       do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (a *Client) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {

	url := fmt.Sprintf("%s/admin/users/%s", a.baseURL, id.String())
	a.logger.Debug(
		"userclient.FindByID: sending request",
		slog.String("url", url),
		slog.String("user_id", id.String()),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+a.serviceToken)

	resp, err := a.client.Do(req)
	if err != nil {
		a.logger.Error(
			"userclient.FindByID: request failed",
			slog.String("url", url),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		a.logger.Debug(
			"userclient.FindByID: user not found",
			slog.String("user_id", id.String()),
		)
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}

		a.logger.Error(
			"userclient.FindByID: unexpected status",
			slog.String("url", url),
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)

		return nil, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	type userResponse struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	}

	u := userResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	uid, err := uuid.Parse(u.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id in response: %w", err)
	}

	return &domain.User{
		ID:    uid,
		Email: u.Email,
	}, nil
}
