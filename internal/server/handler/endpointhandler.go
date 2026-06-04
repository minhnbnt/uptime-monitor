package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
)

type EndpointHandler struct {
	endpointService *service.EndpointService
	validator       *RequestValidator
}

func RegisterEndpointHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EndpointHandler, error) {
		return &EndpointHandler{
			endpointService: do.MustInvoke[*service.EndpointService](i),
			validator:       do.MustInvoke[*RequestValidator](i),
		}, nil
	})
}

func (h *EndpointHandler) SetCheckMethod(c *gin.Context, id int) {

	var req api.SetCheckMethodRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("INVALID_REQUEST", err.Error()))
		return
	}

	ctx := c.Request.Context()
	dtoReq := dto.SetCheckMethodRequest{
		Method:       dto.CheckMethodType(req.Method),
		HTTPMethod:   req.Endpoint.Method,
		Interval:     time.Duration(req.Endpoint.Interval) * time.Second,
		Timeout:      time.Duration(req.Endpoint.Timeout) * time.Second,
		URL:          req.Endpoint.Url,
		ExpectedCode: req.Endpoint.ExpectedCode,
	}

	if !h.validator.Validate(c, dtoReq) {
		return
	}

	if err := h.endpointService.SetCheckMethod(ctx, uint(id), dtoReq); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			errResponse("INTERNAL_ERROR", err.Error()),
		)
		return
	}

	c.Status(http.StatusOK)
}
