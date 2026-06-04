package handler

import (
	"net/http"
	"testing"

	"github.com/go-playground/validator/v10"

	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
)

func TestRequestValidator_Validate(t *testing.T) {
	rv := &RequestValidator{v: validator.New()}

	t.Run("valid struct", func(t *testing.T) {
		c, _ := newGinContext("POST", "/", `{}`)
		ok := rv.Validate(c, dto.RegisterRequest{
			Email:    "a@b.com",
			Username: "user123",
			Password: "password1",
			Name:     "Test",
		})
		if !ok {
			t.Error("expected valid")
		}
	})

	t.Run("invalid struct", func(t *testing.T) {
		c, w := newGinContext("POST", "/", `{}`)
		ok := rv.Validate(c, dto.RegisterRequest{
			Email:    "bad",
			Username: "u",
			Password: "short",
			Name:     "",
		})
		if ok {
			t.Error("expected invalid")
		}
		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}
