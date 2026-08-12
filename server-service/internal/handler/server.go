package handler

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
)

type ServerHandler struct {
	serverReader ServerReader
	serverWriter ServerWriter
}

func RegisterServerHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerHandler, error) {
		return &ServerHandler{
			serverReader: do.MustInvoke[*service.ServerReader](i),
			serverWriter: do.MustInvoke[*service.ServerService](i),
		}, nil
	})
}

func (h *ServerHandler) ListServers(
	ctx context.Context,
	params api.ListServersParams,
) (*api.ServerListResponse, error) {

	page, perPage := params.Page.Or(1), params.PerPage.Or(20)

	userID := authclient.GetUserID(ctx)
	result, total, err := h.serverReader.ListServers(ctx, userID, page, perPage)
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

func (h *ServerHandler) CreateServer(
	ctx context.Context,
	req *api.CreateServerRequest,
) (*api.ServerResponse, error) {

	userID := authclient.GetUserID(ctx)
	result, err := h.serverWriter.CreateServer(ctx, ToCreateServerRequest(req), userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerResponse{Data: ToAPIServer(result)}, nil
}

func (h *ServerHandler) UpdateServer(
	ctx context.Context,
	req *api.UpdateServerRequest,
	params api.UpdateServerParams,
) (*api.ServerResponse, error) {

	userID := authclient.GetUserID(ctx)

	request := ToUpdateServerRequest(req, uint(params.ID))
	result, err := h.serverWriter.UpdateServer(ctx, userID, request)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerResponse{Data: ToAPIServer(result)}, nil
}

func (h *ServerHandler) DeleteServer(
	ctx context.Context,
	params api.DeleteServerParams,
) error {

	userID := authclient.GetUserID(ctx)
	if err := h.serverWriter.DeleteServer(ctx, uint(params.ID), userID); err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}

func (h *ServerHandler) GetServer(
	ctx context.Context,
	params api.GetServerParams,
) (*api.ServerResponse, error) {

	userID := authclient.GetUserID(ctx)
	result, err := h.serverReader.GetServer(ctx, uint(params.ID))
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	if result.CreatedByID != userID {
		return nil, apperrors.ErrForbidden
	}

	return &api.ServerResponse{Data: ToAPIServer(result)}, nil
}

func (h *ServerHandler) CountServersByStatus(
	ctx context.Context,
) (*api.ServerCountResponse, error) {

	userID := authclient.GetUserID(ctx)
	total, online, offline, err := h.serverReader.CountByStatus(ctx, userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	return &api.ServerCountResponse{
		Total:   int(total),
		Online:  int(online),
		Offline: int(offline),
	}, nil
}

func (h *ServerHandler) SearchServers(
	ctx context.Context,
	params api.SearchServersParams,
) (*api.ServerListResponse, error) {

	page, perPage := params.Page.Or(1), params.PerPage.Or(20)

	searchParams := dto.SearchParams{
		Q:         params.Q,
		From:      (page - 1) * perPage,
		To:        perPage,
		SortBy:    string(params.SortBy.Or(api.SearchServersSortByScore)),
		SortOrder: string(params.SortOrder.Or(api.SearchServersSortOrderDesc)),
	}

	userID := authclient.GetUserID(ctx)
	result, total, err := h.serverReader.SearchServers(ctx, searchParams, userID)
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

var _ ServerReader = (*service.ServerReader)(nil)
var _ ServerWriter = (*service.ServerService)(nil)
