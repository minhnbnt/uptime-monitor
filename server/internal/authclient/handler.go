package authclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
)

type AuthHandler struct {
	client *http.Client
	url    string
	logger *slog.Logger
}

func RegisterAuthHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*AuthHandler, error) {
		return &AuthHandler{
			client: &http.Client{},
			url:    "http://auth-service:8081",
			logger: do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func authServiceURL(base, path string) string {
	return fmt.Sprintf("%s/api/v1/auth/%s", base, path)
}

func postJSON[T any](h *AuthHandler, ctx context.Context, url string, body any) (*T, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(body); err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp api.ErrorResponseStatusCode
		if err := json.Unmarshal(respBody, &errResp.Response); err != nil {
			return nil, fmt.Errorf("auth-service error: status=%d", resp.StatusCode)
		}
		errResp.StatusCode = resp.StatusCode
		return nil, &errResp
	}

	var result T
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (h *AuthHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {
	return apperrors.ToAPIError(err)
}

func (h *AuthHandler) Register(ctx context.Context, req *api.RegisterRequest) (*api.AuthResponse, error) {
	return postJSON[api.AuthResponse](h, ctx, authServiceURL(h.url, "register"), req)
}

func (h *AuthHandler) Login(ctx context.Context, req *api.LoginRequest) (*api.AuthResponse, error) {
	return postJSON[api.AuthResponse](h, ctx, authServiceURL(h.url, "login"), req)
}

func (h *AuthHandler) LoginRefresh(ctx context.Context, req *api.RefreshTokenRequest) (*api.AuthResponse, error) {
	return postJSON[api.AuthResponse](h, ctx, authServiceURL(h.url, "refresh"), req)
}

func (h *AuthHandler) Logout(ctx context.Context, req *api.RefreshTokenRequest) error {
	url := authServiceURL(h.url, "logout")

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(req); err != nil {
		return apperrors.ToAPIError(fmt.Errorf("encode request: %w", err))
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return apperrors.ToAPIError(fmt.Errorf("create request: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(httpReq)
	if err != nil {
		return apperrors.ToAPIError(fmt.Errorf("do request: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp api.ErrorResponseStatusCode
		if err := json.NewDecoder(resp.Body).Decode(&errResp.Response); err != nil {
			return apperrors.ToAPIError(fmt.Errorf("auth-service error: status=%d", resp.StatusCode))
		}
		errResp.StatusCode = resp.StatusCode
		return &errResp
	}

	return nil
}
