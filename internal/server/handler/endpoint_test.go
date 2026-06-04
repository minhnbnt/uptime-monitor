package handler

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

func TestEndpointHandler_SetCheckMethod(t *testing.T) {
	val := &RequestValidator{v: validator.New()}

	t.Run("success", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				setCheckMethodFn: func(_ context.Context, id uint, req dto.SetCheckMethodRequest) error {
					return nil
				},
			},
			validator: val,
		}
		body := `{"method":"pull","endpoint":{"url":"https://example.com/h","method":"GET","interval":30,"timeout":10,"expected_code":200}}`
		c, w := newGinContext("PUT", "/api/v1/servers/1/endpoint", body)
		h.SetCheckMethod(c, 1)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("bad json", func(t *testing.T) {
		h := &EndpointHandler{validator: val}
		c, w := newGinContext("PUT", "/api/v1/servers/1/endpoint", `{bad`)
		h.SetCheckMethod(c, 1)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("internal error", func(t *testing.T) {
		h := &EndpointHandler{
			endpointService: &mockEndpointService{
				setCheckMethodFn: func(_ context.Context, _ uint, _ dto.SetCheckMethodRequest) error {
					return errors.New("upsert failed")
				},
			},
			validator: val,
		}
		body := `{"method":"pull","endpoint":{"url":"https://example.com/h","method":"GET","interval":30,"timeout":10,"expected_code":200}}`
		c, w := newGinContext("PUT", "/api/v1/servers/1/endpoint", body)
		h.SetCheckMethod(c, 1)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
		}
	})
}
