package handler

import (
	"net/url"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

func ToAPIServer(s *dto.Server) api.ServerObject {
	if s == nil {
		return api.ServerObject{}
	}

	monitorStatus := api.MonitorStatus("")
	switch s.MonitorStatus {
	case domain.StatusOn:
		monitorStatus = api.MonitorStatusOnline
	case domain.StatusOff:
		monitorStatus = api.MonitorStatusOffline
	}

	return api.ServerObject{
		ID:            int(s.ID),
		Name:          s.Name,
		MonitorStatus: monitorStatus,
		Endpoint:      toAPIEndpoint(s.Endpoint),
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
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
	monitorStatus := api.OptMonitorStatus{}
	switch e.MonitorStatus {
	case domain.StatusOn:
		monitorStatus = api.NewOptMonitorStatus(api.MonitorStatusOnline)
	case domain.StatusOff:
		monitorStatus = api.NewOptMonitorStatus(api.MonitorStatusOffline)
	}
	return api.NewOptEndpoint(api.Endpoint{
		URL:           u,
		MonitorStatus: monitorStatus,
		Interval:      int(e.Interval.Seconds()),
		Timeout:       int(e.Timeout.Seconds()),
		Method:        api.EndpointMethod(e.Method),
		ExpectedCode:  e.ExpectedCode,
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
