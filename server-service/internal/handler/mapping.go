package handler

import (
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

func ToAPIServer(s *dto.Server) api.ServerObject {

	if s == nil {
		return api.ServerObject{}
	}

	monitorStatus := api.OptNilServerObjectMonitorStatus{}
	if s.MonitorStatus != "" {
		monitorStatus.SetTo(api.ServerObjectMonitorStatus(s.MonitorStatus))
	}

	return api.ServerObject{
		ID:            int(s.ID),
		Name:          s.Name,
		Namespace:     s.Namespace,
		Kind:          api.ServerObjectKind(s.Kind),
		ObjectID:      s.ObjectID,
		ContainerName: api.NewOptString(s.ContainerName),
		MonitorStatus: monitorStatus,
		Interval:      api.NewOptInt(int(s.Interval.Seconds())),
		Timeout:       api.NewOptInt(int(s.Timeout.Seconds())),
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

func ToPaginationMeta(page, perPage int, total int64) api.PaginationMeta {

	t := int(total)

	return api.PaginationMeta{
		Page:    api.NewOptInt(page),
		PerPage: api.NewOptInt(perPage),
		Total:   api.NewOptInt(t),
	}
}
