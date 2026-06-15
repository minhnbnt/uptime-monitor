package handler

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/server/middleware"
	"github.com/minhnbnt/uptime-monitor/internal/server/service"
	ontime "github.com/minhnbnt/uptime-monitor/internal/server/service/ontime"
)

type ServerHandler struct {
	serverService ServerService
	ontimeService OntimeService
}

func RegisterServerHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerHandler, error) {
		return &ServerHandler{
			serverService: do.MustInvoke[*service.ServerService](i),
			ontimeService: do.MustInvoke[*ontime.OntimeService](i),
		}, nil
	})
}

func (h *ServerHandler) ListServers(ctx context.Context, params api.ListServersParams) (*api.ServerListResponse, error) {

	page := params.Page.Or(1)
	perPage := params.PerPage.Or(20)

	userID := middleware.GetUserID(ctx)
	result, err := h.serverService.ListServers(ctx, userID, page, perPage)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerListResponse{
		Data: lo.Map(result, func(item dto.Server, _ int) api.ServerObject {
			return toAPIServer(&item)
		}),
		Meta: toPaginationMeta(page, perPage, int64(len(result))),
	}, nil
}

func (h *ServerHandler) CreateServer(ctx context.Context, req *api.CreateServerRequest) (*api.ServerResponse, error) {

	dtoReq := dto.CreateServerRequest{Name: req.Name}

	userID := middleware.GetUserID(ctx)
	result, err := h.serverService.CreateServer(ctx, dtoReq, userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerResponse{Data: toAPIServer(result)}, nil
}

func (h *ServerHandler) GetServer(ctx context.Context, params api.GetServerParams) (*api.ServerResponse, error) {

	result, err := h.ontimeService.GetServerWithOntime(ctx, uint(params.ID))
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	obj := toAPIServer(&result.Server)
	obj.SetOntimeStats(toOntimeStats(result.OntimeStats))

	return &api.ServerResponse{Data: obj}, nil
}

func (h *ServerHandler) UpdateServer(ctx context.Context, req *api.UpdateServerRequest, params api.UpdateServerParams) (*api.ServerResponse, error) {

	dtoReq := dto.UpdateServerRequest{}
	if name, ok := req.Name.Get(); ok {
		dtoReq.Name = &name
	}

	result, err := h.serverService.UpdateServer(ctx, uint(params.ID), dtoReq)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerResponse{Data: toAPIServer(result)}, nil
}

func (h *ServerHandler) DeleteServer(ctx context.Context, params api.DeleteServerParams) error {
	if err := h.serverService.DeleteServer(ctx, uint(params.ID)); err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}

func (h *ServerHandler) SearchServers(ctx context.Context, params api.SearchServersParams) (*api.ServerListResponse, error) {

	searchParams := dto.SearchParams{
		Q:         params.Q,
		Page:      params.Page.Or(1),
		PerPage:   params.PerPage.Or(20),
		SortBy:    string(params.SortBy.Or(api.SearchServersSortByScore)),
		SortOrder: string(params.SortOrder.Or(api.SearchServersSortOrderDesc)),
	}

	userID := middleware.GetUserID(ctx)
	result, total, err := h.serverService.SearchServers(ctx, searchParams, userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	data := lo.Map(result, func(item dto.Server, _ int) api.ServerObject {
		return toAPIServer(&item)
	})

	return &api.ServerListResponse{
		Meta: toPaginationMeta(searchParams.Page, searchParams.PerPage, total),
		Data: data,
	}, nil
}

func (h *ServerHandler) ListServersOntime(ctx context.Context, params api.ListServersOntimeParams) (*api.ServerOntimeListResponse, error) {

	page := params.Page.Or(1)
	perPage := params.PerPage.Or(20)

	userID := middleware.GetUserID(ctx)
	result, total, err := h.ontimeService.ListServersWithOntime(ctx, userID, page, perPage)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	data := lo.Map(result, func(item dto.ServerWithOntime, _ int) api.ServerWithOntime {
		return api.ServerWithOntime{
			Server:      toAPIServer(&item.Server),
			OntimeStats: toOntimeStats(item.OntimeStats),
		}
	})

	return &api.ServerOntimeListResponse{
		Data: data,
		Meta: toPaginationMeta(page, perPage, total),
	}, nil
}
