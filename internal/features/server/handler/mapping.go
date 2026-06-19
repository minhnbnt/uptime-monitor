package handler

import (
	"net/url"

	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor/generated/api"
	ontimedto "github.com/minhnbnt/uptime-monitor/internal/features/ontime/dto"
	"github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

func ToAPIServer(s *dto.Server) api.ServerObject {
	if s == nil {
		return api.ServerObject{}
	}
	return api.ServerObject{
		ID:        int(s.ID),
		Name:      s.Name,
		Status:    api.ServerStatus(s.Status),
		Endpoint:  toAPIEndpoint(s.Endpoint),
		CreatedAt: s.CreatedAt,
		UpdatedAt: s.UpdatedAt,
	}
}

func toAPIEndpoint(e *dto.Endpoint) api.OptEndpoint {
	if e == nil {
		return api.OptEndpoint{}
	}
	var u url.URL
	if parsed, err := url.Parse(e.URL); err == nil {
		u = *parsed
	}
	return api.NewOptEndpoint(api.Endpoint{
		URL:          u,
		Interval:     int(e.Interval.Seconds()),
		Timeout:      int(e.Timeout.Seconds()),
		Method:       api.EndpointMethod(e.Method),
		ExpectedCode: e.ExpectedCode,
	})
}

func ToOntimeStats(stats []ontimedto.OntimeStats) []api.OntimeStats {
	return lo.Map(stats, func(os ontimedto.OntimeStats, _ int) api.OntimeStats {
		return api.OntimeStats{
			Date:  os.Date,
			Stats: os.Stats,
		}
	})
}

func ToPaginationMeta(page, perPage int, total int64) api.PaginationMeta {
	t := int(total)
	return api.PaginationMeta{
		Page:    api.NewOptInt(page),
		PerPage: api.NewOptInt(perPage),
		Total:   api.NewOptInt(t),
	}
}
