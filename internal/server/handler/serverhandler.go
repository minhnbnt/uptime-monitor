package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/oapi-codegen/runtime/types"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

type ServerHandler struct {
	service       *service.ServerService
	ontimeService *service.OntimeService
	pageValidator *utils.PageValidator
	validator     *RequestValidator
}

func RegisterServerHandler(i do.Injector) {

	do.Provide(i, func(i do.Injector) (*ServerHandler, error) {
		return &ServerHandler{
			service:       do.MustInvoke[*service.ServerService](i),
			ontimeService: do.MustInvoke[*service.OntimeService](i),
			pageValidator: utils.NewPageValidator(30),
			validator:     do.MustInvoke[*RequestValidator](i),
		}, nil
	})
}

func (m *ServerHandler) ListServers(c *gin.Context, params api.ListServersParams) {

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
			return toAPIServer(&item)
		}),
		Meta: api.PaginationMeta{
			Page:    &page,
			PerPage: &perPage,
			Total:   new(len(result)),
		},
	})
}

func (m *ServerHandler) CreateServer(c *gin.Context) {

	var req api.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("INVALID_REQUEST", err.Error()))
		return
	}

	ctx := c.Request.Context()
	dtoReq := dto.CreateServerRequest{Name: req.Name}
	if !m.validator.Validate(c, dtoReq) {
		return
	}

	result, err := m.service.CreateServer(ctx, dtoReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	c.JSON(http.StatusCreated, api.ServerResponse{Data: toAPIServer(result)})
}

func (m *ServerHandler) GetServer(c *gin.Context, id int) {

	ctx := c.Request.Context()
	result, err := m.service.GetServer(ctx, uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "Server not found"))
		return
	}

	c.JSON(http.StatusOK, api.ServerResponse{Data: toAPIServer(result)})
}

func (m *ServerHandler) UpdateServer(c *gin.Context, id int) {

	var req api.UpdateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errResponse("INVALID_REQUEST", err.Error()))
		return
	}

	ctx := c.Request.Context()
	dtoReq := dto.UpdateServerRequest{Name: req.Name}
	if !m.validator.Validate(c, dtoReq) {
		return
	}

	result, err := m.service.UpdateServer(ctx, uint(id), dtoReq)
	if err != nil {
		c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "Server not found"))
		return
	}

	c.JSON(http.StatusOK, api.ServerResponse{Data: toAPIServer(result)})
}

func (m *ServerHandler) DeleteServer(c *gin.Context, id int) {

	ctx := c.Request.Context()
	if err := m.service.DeleteServer(ctx, uint(id)); err != nil {
		c.JSON(http.StatusNotFound, errResponse("NOT_FOUND", "Server not found"))
		return
	}

	c.Status(http.StatusNoContent)
}

func (m *ServerHandler) ListServersOntime(c *gin.Context, params api.ListServersOntimeParams) {

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
	result, total, err := m.ontimeService.ListServersWithOntime(ctx, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errResponse("INTERNAL_ERROR", err.Error()))
		return
	}

	data := lo.Map(result, func(item dto.ServerWithOntime, _ int) api.ServerWithOntime {
		return api.ServerWithOntime{
			Server: toAPIServer(&item.Server),
			OntimeStats: lo.Map(item.OntimeStats, func(os dto.OntimeStats, _ int) api.OntimeStats {
				return api.OntimeStats{
					Date:  types.Date{Time: os.Date},
					Stats: os.Stats,
				}
			}),
		}
	})

	totalInt := int(total)
	c.JSON(http.StatusOK, api.ServerOntimeListResponse{
		Data: data,
		Meta: api.PaginationMeta{
			Page:    &page,
			PerPage: &perPage,
			Total:   &totalInt,
		},
	})
}

func toAPIEndpoint(e *dto.Endpoint) *api.Endpoint {

	if e == nil {
		return nil
	}

	return &api.Endpoint{
		Url:          e.URL,
		Interval:     int(e.Interval.Seconds()),
		Timeout:      int(e.Timeout.Seconds()),
		Method:       e.Method,
		ExpectedCode: e.ExpectedCode,
	}
}

func toAPIServer(s *dto.Server) api.Server {
	return api.Server{
		Id:        int(s.ID),
		Name:      s.Name,
		Status:    api.ServerStatus(s.Status),
		Endpoint:  toAPIEndpoint(s.Endpoint),
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
