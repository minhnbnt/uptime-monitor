package handler

import (
	"context"
	"time"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/service"
)

type ServerHandler struct {
	serverService ServerService
}

func RegisterServerHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*ServerHandler, error) {
		return &ServerHandler{
			serverService: do.MustInvoke[*service.ServerService](i),
		}, nil
	})
}

func (h *ServerHandler) ListServers(
	ctx context.Context,
	params api.ListServersParams,
) (*api.ServerListResponse, error) {

	page, perPage := params.Page.Or(1), params.PerPage.Or(20)

	userID := authclient.GetUserID(ctx)
	result, total, err := h.serverService.ListServers(ctx, userID, page, perPage)
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

	dtoReq := dto.CreateServerRequest{
		Name:      req.Name,
		Namespace: req.Namespace,
		Kind:      string(req.Kind),
		ObjectID:  req.ObjectID,
	}

	if v, ok := req.ContainerName.Get(); ok {
		dtoReq.ContainerName = v
	}
	if v, ok := req.Interval.Get(); ok {
		dtoReq.Interval = time.Duration(v) * time.Second
	}
	if v, ok := req.Timeout.Get(); ok {
		dtoReq.Timeout = time.Duration(v) * time.Second
	}

	userID := authclient.GetUserID(ctx)
	result, err := h.serverService.CreateServer(ctx, dtoReq, userID)
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

	dtoReq := dto.UpdateServerRequest{}
	if name, ok := req.Name.Get(); ok {
		dtoReq.Name = &name
	}
	if ns, ok := req.Namespace.Get(); ok {
		dtoReq.Namespace = &ns
	}
	if kind, ok := req.Kind.Get(); ok {
		k := string(kind)
		dtoReq.Kind = &k
	}
	if oid, ok := req.ObjectID.Get(); ok {
		dtoReq.ObjectID = &oid
	}
	if cn, ok := req.ContainerName.Get(); ok {
		dtoReq.ContainerName = &cn
	}
	if v, ok := req.Interval.Get(); ok {
		d := time.Duration(v) * time.Second
		dtoReq.Interval = &d
	}
	if v, ok := req.Timeout.Get(); ok {
		d := time.Duration(v) * time.Second
		dtoReq.Timeout = &d
	}

	userID := authclient.GetUserID(ctx)
	result, err := h.serverService.UpdateServer(ctx, uint(params.ID), userID, dtoReq)
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
	if err := h.serverService.DeleteServer(ctx, uint(params.ID), userID); err != nil {
		return apperrors.ToAPIError(err)
	}

	return nil
}

func (h *ServerHandler) GetServer(
	ctx context.Context,
	params api.GetServerParams,
) (*api.ServerResponse, error) {

	userID := authclient.GetUserID(ctx)
	result, err := h.serverService.GetServer(ctx, uint(params.ID))
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
	total, online, offline, err := h.serverService.CountByStatus(ctx, userID)
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

var _ ServerService = (*service.ServerService)(nil)
