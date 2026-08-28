package handler

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/service"
)

type OntimeService interface {
	ListServersWithOntime(ctx context.Context, createdByID uint, page, perPage int) ([]dto.ServerOntime, error)
	GetServerWithOntime(ctx context.Context, serverID uint, userID uint) (*dto.ServerOntime, error)
	GetServersWithOntime(ctx context.Context, userID uint, ids []uint) ([]dto.ServerOntime, error)
}

type OntimeHandler struct {
	ontimeService OntimeService
	eventService  *service.EventService
}

func RegisterOntimeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeHandler, error) {
		return &OntimeHandler{
			ontimeService: do.MustInvoke[*service.OntimeService](i),
			eventService:  do.MustInvoke[*service.EventService](i),
		}, nil
	})
}

func (h *OntimeHandler) ListServersOntime(ctx context.Context, params api.ListServersOntimeParams) (*api.ServerOntimeListResponse, error) {

	userID := authclient.GetUserID(ctx)
	page := params.Page.Or(1)
	perPage := params.PerPage.Or(20)

	result, err := h.ontimeService.ListServersWithOntime(ctx, userID, page, perPage)
	if err != nil {
		return nil, err
	}

	data := lo.Map(result, func(item dto.ServerOntime, _ int) api.ServerOntime {
		return api.ServerOntime{
			ServerID:    api.NewOptInt(int(item.ServerID)),
			OntimeStats: toOntimeStats(item.OntimeStats),
		}
	})

	return &api.ServerOntimeListResponse{Data: data}, nil
}

func (h *OntimeHandler) GetServerOntime(ctx context.Context, params api.GetServerOntimeParams) (*api.ServerOntimeResponse, error) {

	userID := authclient.GetUserID(ctx)

	result, err := h.ontimeService.GetServerWithOntime(ctx, uint(params.ID), userID)
	if err != nil {
		return nil, err
	}

	so := api.ServerOntime{
		ServerID:    api.NewOptInt(int(result.ServerID)),
		OntimeStats: toOntimeStats(result.OntimeStats),
	}

	return &api.ServerOntimeResponse{Data: api.NewOptServerOntime(so)}, nil
}

func (h *OntimeHandler) GetServersStatuses(ctx context.Context, params *api.ServerStatusesRequest) (*api.ServerStatusesResponse, error) {

	userID := authclient.GetUserID(ctx)

	ids := lo.Map(params.Ids, func(id int, _ int) uint { return uint(id) })

	statuses, err := h.eventService.GetServersStatuses(ctx, userID, ids)
	if err != nil {
		return nil, err
	}

	data := lo.Map(statuses, func(st dto.EndpointStatus, _ int) api.ServerStatusItem {
		return api.ServerStatusItem{
			ServerID: int(st.EndpointID),
			Status:   st.Status.String(),
		}
	})

	return &api.ServerStatusesResponse{Data: data}, nil
}

func (h *OntimeHandler) ListServersOntimeByIDs(ctx context.Context, params *api.ServerOntimeByIDsRequest) (*api.ServerOntimeListResponse, error) {

	userID := authclient.GetUserID(ctx)

	ids := lo.Map(params.Ids, func(id int, _ int) uint { return uint(id) })
	if len(ids) > 100 {
		return nil, apperrors.ErrBadRequest
	}

	result, err := h.ontimeService.GetServersWithOntime(ctx, userID, ids)
	if err != nil {
		return nil, err
	}

	data := lo.Map(result, func(item dto.ServerOntime, _ int) api.ServerOntime {
		return api.ServerOntime{
			ServerID:    api.NewOptInt(int(item.ServerID)),
			OntimeStats: toOntimeStats(item.OntimeStats),
		}
	})

	return &api.ServerOntimeListResponse{Data: data}, nil
}

func (h *OntimeHandler) CountServersByStatus(ctx context.Context) (*api.ServerCountResponse, error) {

	userID := authclient.GetUserID(ctx)
	total, online, offline, err := h.eventService.CountServersByStatus(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &api.ServerCountResponse{
		Total:   api.NewOptInt(int(total)),
		Online:  api.NewOptInt(int(online)),
		Offline: api.NewOptInt(int(offline)),
	}, nil
}

func (h *OntimeHandler) NewError(_ context.Context, err error) *api.ErrorResponseStatusCode {

	status, body := apperrors.ToAPIError(err)

	return &api.ErrorResponseStatusCode{
		StatusCode: status,
		Response: api.ErrorResponse{
			Error:   api.NewOptString(body["error"].(string)),
			Message: api.NewOptString(body["message"].(string)),
		},
	}
}

func toOntimeStats(stats []dto.OntimeStats) []api.OntimeStats {
	return lo.Map(stats, func(s dto.OntimeStats, _ int) api.OntimeStats {
		return api.OntimeStats{
			Date:           api.NewOptDateTime(s.Date),
			Stats:          api.NewOptFloat64(s.Stats),
			HasData:        api.NewOptBool(s.HasData),
			UnknownSeconds: api.NewOptFloat64(s.UnknownSeconds),
		}
	})
}

var (
	_ OntimeService = (*service.OntimeService)(nil)
	_ api.Handler   = (*OntimeHandler)(nil)
)
