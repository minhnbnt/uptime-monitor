package handler

import (
	"context"
	"io"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/features/auth/middleware"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/infrastructure"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/service"
)

type ServerHandler struct {
	serverService ServerService
	excelExporter *infrastructure.ExcelExporter
}

func RegisterServerHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerHandler, error) {
		return &ServerHandler{
			serverService: do.MustInvoke[*service.ServerService](i),
			excelExporter: do.MustInvoke[*infrastructure.ExcelExporter](i),
		}, nil
	})
}

func (h *ServerHandler) ListServers(ctx context.Context, params api.ListServersParams) (*api.ServerListResponse, error) {

	page, perPage := params.Page.Or(1), params.PerPage.Or(20)

	userID := middleware.GetUserID(ctx)
	result, total, err := h.serverService.ListServers(ctx, userID, page, perPage)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerListResponse{
		Meta: ToPaginationMeta(page, perPage, total),
		Data: lo.Map(result, func(item dto.Server, _ int) api.ServerObject {
			return ToAPIServer(&item)
		}),
	}, nil
}

func (h *ServerHandler) CreateServer(ctx context.Context, req *api.CreateServerRequest) (*api.ServerResponse, error) {

	dtoReq := dto.CreateServerRequest{Name: req.Name}

	userID := middleware.GetUserID(ctx)
	result, err := h.serverService.CreateServer(ctx, dtoReq, userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerResponse{Data: ToAPIServer(result)}, nil
}

func (h *ServerHandler) UpdateServer(ctx context.Context, req *api.UpdateServerRequest, params api.UpdateServerParams) (*api.ServerResponse, error) {

	dtoReq := dto.UpdateServerRequest{}
	if name, ok := req.Name.Get(); ok {
		dtoReq.Name = &name
	}

	userID := middleware.GetUserID(ctx)
	result, err := h.serverService.UpdateServer(ctx, uint(params.ID), userID, dtoReq)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerResponse{Data: ToAPIServer(result)}, nil
}

func (h *ServerHandler) DeleteServer(ctx context.Context, params api.DeleteServerParams) error {

	userID := middleware.GetUserID(ctx)
	if err := h.serverService.DeleteServer(ctx, uint(params.ID), userID); err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}

func (h *ServerHandler) SearchServers(ctx context.Context, params api.SearchServersParams) (*api.ServerListResponse, error) {

	page, perPage := params.Page.Or(1), params.PerPage.Or(20)

	searchParams := dto.SearchParams{
		Q:         params.Q,
		From:      (page - 1) * perPage,
		To:        perPage,
		SortBy:    string(params.SortBy.Or(api.SearchServersSortByScore)),
		SortOrder: string(params.SortOrder.Or(api.SearchServersSortOrderDesc)),
	}

	userID := middleware.GetUserID(ctx)
	result, total, err := h.serverService.SearchServers(ctx, searchParams, userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	data := lo.Map(result, func(item dto.Server, _ int) api.ServerObject {
		return ToAPIServer(&item)
	})

	return &api.ServerListResponse{
		Meta: ToPaginationMeta(page, perPage, total),
		Data: data,
	}, nil
}

func (h *ServerHandler) ExportServers(ctx context.Context, params api.ExportServersParams) (*api.ExportServersOKHeaders, error) {

	searchParams := dto.SearchParams{
		Q:         params.Q.Or(""),
		From:      params.From.Or(0),
		To:        params.To.Or(100),
		SortBy:    string(params.SortBy.Or(api.ExportServersSortByName)),
		SortOrder: string(params.SortOrder.Or(api.ExportServersSortOrderAsc)),
	}

	userID := middleware.GetUserID(ctx)
	result, _, err := h.serverService.SearchServers(ctx, searchParams, userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	pr, pw := io.Pipe()
	go func() {
		if err := h.excelExporter.GenerateExportFile(pw, result); err != nil {
			_ = pw.CloseWithError(err)
		} else {
			pw.Close()
		}
	}()

	return &api.ExportServersOKHeaders{
		ContentDisposition: api.NewOptString(`attachment; filename="servers.xlsx"`),
		Response:           api.ExportServersOK{Data: pr},
	}, nil
}

var (
	_ ServerService = (*service.ServerService)(nil)
)
