package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/samber/do/v2"
)

type RequestValidator struct {
	v *validator.Validate
}

func RegisterRequestValidator(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*RequestValidator, error) {
		return &RequestValidator{v: validator.New()}, nil
	})
}

func (rv *RequestValidator) Validate(c *gin.Context, data any) bool {
	if err := rv.v.Struct(data); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("VALIDATION_ERROR", err.Error()))
		return false
	}
	return true
}
