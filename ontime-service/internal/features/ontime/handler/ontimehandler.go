package handler

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/authclient"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	ontimedto "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/features/ontime/service"
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

func (h *OntimeHandler) ListServersOntime(ctx context.Context, params api.ListServersOntimeParams) (*api.ServerOntimeListResponse, error) {

	userID := authclient.GetUserID(ctx)
	page := params.Page.Or(1)
	perPage := params.PerPage.Or(20)

	result, err := h.ontimeService.ListServersWithOntime(ctx, userID, page, perPage)
	if err != nil {
		return nil, err
	}

	data := lo.Map(result, func(item ontimedto.ServerOntime, _ int) api.ServerOntime {
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

func toOntimeStats(stats []ontimedto.OntimeStats) []api.OntimeStats {
	return lo.Map(stats, func(s ontimedto.OntimeStats, _ int) api.OntimeStats {
		return api.OntimeStats{
			Date:  api.NewOptDateTime(s.Date),
			Stats: api.NewOptFloat64(s.Stats),
		}
	})
}

func init() {
	// ponytail: ensure compile-time interface satisfaction
	var _ OntimeService = (*ontime.OntimeService)(nil)
	var _ api.Handler = (*OntimeHandler)(nil)
}
