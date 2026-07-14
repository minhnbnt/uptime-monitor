package handler

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	"github.com/minhnbnt/uptime-monitor/internal/authclient"
	apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	"github.com/minhnbnt/uptime-monitor/internal/features/ontime/service"
	serverhandler "github.com/minhnbnt/uptime-monitor/internal/features/server/handler"
)

type OntimeHandler struct {
	ontimeService OntimeService
}

func RegisterOntimeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeHandler, error) {
		return &OntimeHandler{
			ontimeService: do.MustInvoke[*ontime.OntimeService](i),
		}, nil
	})
}

func (h *OntimeHandler) GetServer(ctx context.Context, params api.GetServerParams) (*api.ServerResponse, error) {

	userID := authclient.GetUserID(ctx)
	result, err := h.ontimeService.GetServerWithOntime(ctx, uint(params.ID), userID)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	obj := serverhandler.ToAPIServer(&result.Server)
	obj.SetOntimeStats(serverhandler.ToOntimeStats(result.OntimeStats))

	return &api.ServerResponse{Data: obj}, nil
}

func (h *OntimeHandler) ListServersOntime(ctx context.Context, params api.ListServersOntimeParams) (*api.ServerOntimeListResponse, error) {
	page, perPage := params.Page.Or(1), params.PerPage.Or(20)

	userID := authclient.GetUserID(ctx)
	result, total, online, offline, err := h.ontimeService.ListServersWithOntime(ctx, userID, page, perPage)
	if err != nil {
		return nil, apperrors.ToAPIError(err)
	}

	data := lo.Map(result, func(item ontimedto.ServerWithOntime, _ int) api.ServerWithOntime {
		return api.ServerWithOntime{
			Server:      serverhandler.ToAPIServer(&item.Server),
			OntimeStats: serverhandler.ToOntimeStats(item.OntimeStats),
		}
	})

	return &api.ServerOntimeListResponse{
		Meta:         serverhandler.ToPaginationMeta(page, perPage, total),
		Data:         data,
		OnlineCount:  int(online),
		OfflineCount: int(offline),
	}, nil
}
