package handler

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/samber/do/v2"
	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	apperrors "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/errors"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/service"
)

type OntimeService interface {
	ListServersWithOntime(ctx context.Context, createdByID uuid.UUID, page, perPage int) ([]dto.ServerOntime, error)
	GetServerWithOntime(ctx context.Context, serverID uint, userID uuid.UUID) (*dto.ServerOntime, error)
}

type OntimeRangeService interface {
	CalculateUptime(ctx context.Context, in dto.CalculateUptimeInput) (*dto.UptimeResponse, error)
}

type OntimeHandler struct {
	ontimeService      OntimeService
	ontimeRangeService OntimeRangeService
}

func RegisterOntimeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*OntimeHandler, error) {
		return &OntimeHandler{
			ontimeService:      do.MustInvoke[*service.OntimeService](i),
			ontimeRangeService: do.MustInvoke[*service.OntimeRangeService](i),
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
			OntimeStats: toOntimeStats(item.DayStats),
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
		OntimeStats: toOntimeStats(result.DayStats),
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

func (h *OntimeHandler) CalculateUptime(
	ctx context.Context,
	req *api.CalculateUptimeRequest,
	params api.CalculateUptimeParams,
) (*api.UptimeResponse, error) {

	userID := authclient.GetUserID(ctx)

	resolution := time.Duration(15 * time.Minute)
	if req.Resolution.IsSet() {
		d, err := utils.ParseResolution(req.Resolution.Value)
		if err != nil {
			return nil, err
		}
		resolution = d
	}

	input := dto.CalculateUptimeInput{
		ServerID:   uint(params.ID),
		UserID:     userID,
		From:       req.From,
		To:         req.To,
		Resolution: resolution,
	}

	result, err := h.ontimeRangeService.CalculateUptime(ctx, input)

	if err != nil {
		return nil, err
	}

	return toAPIUptimeResponse(result)
}

func toOntimeStats(stats []dto.DayStats) []api.OntimeStats {
	return lo.Map(stats, func(s dto.DayStats, _ int) api.OntimeStats {
		return api.OntimeStats{
			Date:    api.NewOptDateTime(s.Date),
			Stats:   api.NewOptFloat64(s.Result.Uptime),
			HasData: api.NewOptBool(s.Result.HasData),
		}
	})
}

func toAPIUptimeResponse(r *dto.UptimeResponse) (*api.UptimeResponse, error) {
	intervals := make([]api.IntervalResult, len(r.Intervals))
	for i, iv := range r.Intervals {
		from, err := time.Parse(time.RFC3339, iv.From)
		if err != nil {
			return nil, err
		}
		to, err := time.Parse(time.RFC3339, iv.To)
		if err != nil {
			return nil, err
		}
		intervals[i] = api.IntervalResult{
			From:    api.NewOptDateTime(from),
			To:      api.NewOptDateTime(to),
			Uptime:  api.NewOptFloat64(iv.Uptime),
			HasData: api.NewOptBool(iv.HasData),
		}
	}

	from, err := time.Parse(time.RFC3339, r.From)
	if err != nil {
		return nil, err
	}
	to, err := time.Parse(time.RFC3339, r.To)
	if err != nil {
		return nil, err
	}

	return &api.UptimeResponse{
		ServerID:      api.NewOptInt(int(r.ServerID)),
		Uptime:        api.NewOptFloat64(r.Uptime),
		From:          api.NewOptDateTime(from),
		To:            api.NewOptDateTime(to),
		TotalSeconds:  api.NewOptFloat64(r.TotalSeconds),
		OnlineSeconds: api.NewOptFloat64(r.OnlineSeconds),
		HasData:       api.NewOptBool(r.HasData),
		Partial:       api.NewOptBool(r.Partial),
		Intervals:     intervals,
	}, nil
}

var (
	_ OntimeService      = (*service.OntimeService)(nil)
	_ OntimeRangeService = (*service.OntimeRangeService)(nil)
	_ api.Handler        = (*OntimeHandler)(nil)
)
