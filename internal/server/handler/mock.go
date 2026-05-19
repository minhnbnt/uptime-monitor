package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/domain"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

type MockServer struct {
	service       *service.ServerService
	pageValidator *utils.PageValidator
}

func RegisterMockServer(i do.Injector) {

	do.Provide(i, func(i do.Injector) (*MockServer, error) {
		return &MockServer{
			service:       do.MustInvoke[*service.ServerService](i),
			pageValidator: utils.NewPageValidator(30),
		}, nil
	})
}

func (m *MockServer) ListServers(c *gin.Context, params api.ListServersParams) {

	page := 1
	if params.Page != nil {
		page = *params.Page
	}

	perPage := 20
	if params.PerPage != nil {
		perPage = *params.PerPage
	}

	if err := m.pageValidator.Validate(page, perPage); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("INVALID_REQUEST", err.Error()))
		return
	}

	ctx := c.Request.Context()
	result, err := m.service.ListServers(ctx, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusOK, api.ServerListResponse{
		Data: lo.Map(result, func(item dto.Server, _ int) api.Server {
			return toAPIServer(item)
		}),
		Meta: api.PaginationMeta{
			Page:    &page,
			PerPage: &perPage,
			Total:   new(len(result)),
		},
	})
}

func (m *MockServer) CreateServer(c *gin.Context) {

	var req api.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("INVALID_REQUEST", err.Error()))
		return
	}

	ctx := c.Request.Context()
	result, err := m.service.CreateServer(ctx, dto.CreateServerRequest{
		Name: req.Name,
		URL:  req.Url,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, api.ServerResponse{Data: toAPIServer(*result)})
}

func (m *MockServer) GetServer(c *gin.Context, id openapi_types.UUID) {

	ctx := c.Request.Context()
	result, err := m.service.GetServer(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "Server not found"))
		return
	}

	c.JSON(http.StatusOK, api.ServerResponse{Data: toAPIServer(*result)})
}

func (m *MockServer) UpdateServer(c *gin.Context, id openapi_types.UUID) {

	var req api.UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("INVALID_REQUEST", err.Error()))
		return
	}

	status := (*domain.Status)(req.Status)

	ctx := c.Request.Context()
	result, err := m.service.UpdateServer(ctx, id, dto.UpdateServerRequest{
		Name:   req.Name,
		URL:    req.Url,
		Status: status,
	})

	if err != nil {
		c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "Server not found"))
		return
	}

	c.JSON(http.StatusOK, api.ServerResponse{Data: toAPIServer(*result)})
}

func (m *MockServer) DeleteServer(c *gin.Context, id openapi_types.UUID) {

	ctx := c.Request.Context()
	if err := m.service.DeleteServer(ctx, id); err != nil {
		c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "Server not found"))
		return
	}

	c.Status(http.StatusNoContent)
}

func toAPIServer(s dto.Server) api.Server {
	return api.Server{
		Id:        s.ID,
		Name:      s.Name,
		Url:       s.URL,
		Status:    api.ServerStatus(s.Status),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func errResponse(code, msg string) api.ErrorResponse {

	return api.ErrorResponse{
		Error: struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}{
			Code:    code,
			Message: msg,
		},
	}
}
