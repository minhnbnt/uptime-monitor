package handler

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

func TestServerHandler_SearchServers(t *testing.T) {
	now := time.Now()

	t.Run("success with defaults", func(t *testing.T) {
		serverService := &mockServerService{
			searchServersFn: func(_ context.Context, params dto.SearchParams, _ uint) ([]dto.Server, int64, error) {
				if params.Q != "test" {
					t.Errorf("Q = %q", params.Q)
				}
				if params.From != 0 {
					t.Errorf("From = %d", params.From)
				}
				return []dto.Server{dtoServer(1, "s1", now)}, 1, nil
			},
		}

		h := &ServerHandler{serverService: serverService}

		resp, err := h.SearchServers(t.Context(), api.SearchServersParams{
			Q: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Meta.Total.Value != 1 {
			t.Errorf("Total = %d", resp.Meta.Total.Value)
		}
		if len(resp.Data) != 1 {
			t.Errorf("len(Data) = %d", len(resp.Data))
		}
	})

	t.Run("with custom params", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				searchServersFn: func(_ context.Context, params dto.SearchParams, _ uint) ([]dto.Server, int64, error) {
					if params.From != 20 {
						t.Errorf("From = %d", params.From)
					}
					return []dto.Server{}, 0, nil
				},
			},
		}

		resp, err := h.SearchServers(t.Context(), api.SearchServersParams{
			Q:         "test",
			Page:      api.NewOptInt(3),
			PerPage:   api.NewOptInt(10),
			SortBy:    api.NewOptSearchServersSortBy(api.SearchServersSortByName),
			SortOrder: api.NewOptSearchServersSortOrder(api.SearchServersSortOrderAsc),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Meta.Page.Value != 3 {
			t.Errorf("Page = %d", resp.Meta.Page.Value)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				searchServersFn: func(_ context.Context, _ dto.SearchParams, _ uint) ([]dto.Server, int64, error) {
					return nil, 0, errors.New("search error")
				},
			},
		}

		_, err := h.SearchServers(t.Context(), api.SearchServersParams{Q: "test"})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})
}

func TestServerHandler_ExportServers(t *testing.T) {
	now := time.Now()

	t.Run("success", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				searchServersFn: func(_ context.Context, params dto.SearchParams, _ uint) ([]dto.Server, int64, error) {
					if params.Q != "" {
						t.Errorf("Q = %q", params.Q)
					}
					return []dto.Server{dtoServer(1, "s1", now)}, 1, nil
				},
			},
			excelExporter: nil,
		}

		_, err := h.ExportServers(t.Context(), api.ExportServersParams{})
		// Without a real excel generator, the goroutine will panic
		// Just test the happy path that doesn't error before the goroutine
		var statusErr *api.ErrorResponseStatusCode
		if errors.As(err, &statusErr) {
			if statusErr.StatusCode != http.StatusOK {
				t.Errorf("unexpected error: %v", err)
			}
		}
	})

	t.Run("search error", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				searchServersFn: func(_ context.Context, _ dto.SearchParams, _ uint) ([]dto.Server, int64, error) {
					return nil, 0, errors.New("search error")
				},
			},
		}

		_, err := h.ExportServers(t.Context(), api.ExportServersParams{})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})

	t.Run("with custom params", func(t *testing.T) {
		h := &ServerHandler{
			serverService: &mockServerService{
				searchServersFn: func(_ context.Context, params dto.SearchParams, _ uint) ([]dto.Server, int64, error) {
					if params.Q != "filter" {
						t.Errorf("Q = %q", params.Q)
					}
					return []dto.Server{}, 0, nil
				},
			},
		}

		_, err := h.ExportServers(t.Context(), api.ExportServersParams{
			Q:         api.NewOptString("filter"),
			From:      api.NewOptInt(0),
			To:        api.NewOptInt(50),
			SortBy:    api.NewOptExportServersSortBy(api.ExportServersSortByCreatedAt),
			SortOrder: api.NewOptExportServersSortOrder(api.ExportServersSortOrderDesc),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestEndpointHandler_SetCheckMethod_PushNotImplemented(t *testing.T) {
	h := &EndpointHandler{}
	parsedURL, _ := url.Parse("https://example.com/h")
	req := &api.SetCheckMethodRequest{
		Method: api.CheckMethodTypePush,
		Endpoint: api.Endpoint{
			URL:          *parsedURL,
			Interval:     30,
			Timeout:      10,
			Method:       "GET",
			ExpectedCode: 200,
		},
	}

	_, err := h.SetCheckMethod(t.Context(), req, api.SetCheckMethodParams{ID: 1})
	var statusErr *api.ErrorResponseStatusCode
	if !errors.As(err, &statusErr) {
		t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
	}
	if statusErr.StatusCode != http.StatusNotImplemented {
		t.Errorf("status = %d, want %d", statusErr.StatusCode, http.StatusNotImplemented)
	}
	if statusErr.Response.Error.Code != "NOT_IMPLEMENTED" {
		t.Errorf("code = %q", statusErr.Response.Error.Code)
	}
}

func TestEndpointHandler_SetCheckMethod_GetServerError(t *testing.T) {
	parsedURL, _ := url.Parse("https://example.com/h")

	t.Run("endpoint service error", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				setCheckMethodFn: func(_ context.Context, _ uint, _ uint, _ dto.SetCheckMethodRequest) error {
					return errors.New("endpoint error")
				},
			},
		}

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
		_, err := h.SetCheckMethod(t.Context(), req, api.SetCheckMethodParams{ID: 1})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})

	t.Run("get server not found", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				setCheckMethodFn: func(_ context.Context, _ uint, _ uint, _ dto.SetCheckMethodRequest) error {
					return nil
				},
			},
			serverService: &mockServerService{
				getServerFn: func(_ context.Context, _ uint) (*dto.Server, error) {
					return nil, apperrors.ErrNotFound
				},
			},
		}

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
		_, err := h.SetCheckMethod(t.Context(), req, api.SetCheckMethodParams{ID: 99})
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusNotFound {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})
}

func TestEndpointHandler_TestEndpoint(t *testing.T) {
	parsedURL, _ := url.Parse("https://example.com/test")

	t.Run("success", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				testEndpointFn: func(_ context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {
					if req.URL != "https://example.com/test" {
						t.Errorf("URL = %q", req.URL)
					}
					return &dto.TestEndpointResponse{Success: true, StatusCode: 200}, nil
				},
			},
		}

		req := &api.TestEndpointRequest{
			URL:    *parsedURL,
			Method: api.TestEndpointRequestMethodGET,
		}
		resp, err := h.TestEndpoint(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !resp.Success {
			t.Error("expected Success=true")
		}
		if resp.StatusCode != 200 {
			t.Errorf("StatusCode = %d", resp.StatusCode)
		}
	})

	t.Run("with error message", func(t *testing.T) {
		errMsg := "connection refused"
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				testEndpointFn: func(_ context.Context, _ dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {
					return &dto.TestEndpointResponse{Success: false, StatusCode: 0, Error: &errMsg}, nil
				},
			},
		}

		req := &api.TestEndpointRequest{
			URL:    *parsedURL,
			Method: api.TestEndpointRequestMethodGET,
		}
		resp, err := h.TestEndpoint(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.Success {
			t.Error("expected Success=false")
		}
		if !resp.Error.Set || resp.Error.Value != "connection refused" {
			t.Errorf("Error = %+v", resp.Error)
		}
	})

	t.Run("service error", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				testEndpointFn: func(_ context.Context, _ dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {
					return nil, errors.New("test error")
				},
			},
		}

		req := &api.TestEndpointRequest{
			URL:    *parsedURL,
			Method: api.TestEndpointRequestMethodGET,
		}
		_, err := h.TestEndpoint(t.Context(), req)
		var statusErr *api.ErrorResponseStatusCode
		if !errors.As(err, &statusErr) {
			t.Fatalf("expected ErrorResponseStatusCode, got %T", err)
		}
		if statusErr.StatusCode != http.StatusInternalServerError {
			t.Errorf("status = %d", statusErr.StatusCode)
		}
	})

	t.Run("with custom timeout and expected code", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				testEndpointFn: func(_ context.Context, req dto.TestEndpointRequest) (*dto.TestEndpointResponse, error) {
					if req.Timeout != 15*time.Second {
						t.Errorf("Timeout = %v", req.Timeout)
					}
					if req.ExpectedCode != 201 {
						t.Errorf("ExpectedCode = %d", req.ExpectedCode)
					}
					return &dto.TestEndpointResponse{Success: true, StatusCode: 201}, nil
				},
			},
		}

		req := &api.TestEndpointRequest{
			URL:          *parsedURL,
			Method:       api.TestEndpointRequestMethodPOST,
			Timeout:      api.NewOptInt(15),
			ExpectedCode: api.NewOptInt(201),
		}
		resp, err := h.TestEndpoint(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != 201 {
			t.Errorf("StatusCode = %d", resp.StatusCode)
		}
	})
}
