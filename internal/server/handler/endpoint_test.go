package handler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

func TestEndpointHandler_SetCheckMethod(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				setCheckMethodFn: func(_ context.Context, id uint, req dto.SetCheckMethodRequest) error {
					return nil
				},
			},
			serverService: &mockServerService{
				getServerFn: func(_ context.Context, id uint) (*dto.Server, error) {
					return &dto.Server{ID: id, Name: "srv"}, nil
				},
			},
		}

		parsedURL, _ := url.Parse("https://example.com/h")
		req := &api.SetCheckMethodRequest{
			Method: api.CheckMethodTypePull,
			Endpoint: api.Endpoint{
				URL:          *parsedURL,
				Interval:     30,
				Timeout:      10,
				Method:       "GET",
				ExpectedCode: 200,
			},
		}
		resp, err := h.SetCheckMethod(context.Background(), req, api.SetCheckMethodParams{ID: 1})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Data.ID != 1 || resp.Data.Name != "srv" {
			t.Errorf("unexpected response: %+v", resp)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				setCheckMethodFn: func(_ context.Context, _ uint, _ dto.SetCheckMethodRequest) error {
					return errors.New("upsert failed")
				},
			},
		}

		parsedURL, _ := url.Parse("https://example.com/h")
		req := &api.SetCheckMethodRequest{
			Method: api.CheckMethodTypePull,
			Endpoint: api.Endpoint{
				URL:          *parsedURL,
				Interval:     30,
				Timeout:      10,
				Method:       "GET",
				ExpectedCode: 200,
			},
		}
		_, err := h.SetCheckMethod(context.Background(), req, api.SetCheckMethodParams{ID: 1})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusInternalServerError)
		}
	})
}
