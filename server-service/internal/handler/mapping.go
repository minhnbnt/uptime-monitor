package handler

import (
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

func toHttpConfig(v interface {
	GetPort() api.OptInt
	GetEndpointPath() api.OptString
	GetExpectedCode() api.OptInt
	GetBodyCheckExpr() api.OptString
	GetMethod() api.OptString
}) *dto.HttpConfig {
	cfg := &dto.HttpConfig{}
	if p, ok := v.GetPort().Get(); ok {
		cfg.Port = p
	}
	if ep, ok := v.GetEndpointPath().Get(); ok {
		cfg.EndpointPath = ep
	}
	if ec, ok := v.GetExpectedCode().Get(); ok {
		cfg.ExpectedCode = ec
	}
	if be, ok := v.GetBodyCheckExpr().Get(); ok {
		cfg.BodyCheckExpr = be
	}
	if m, ok := v.GetMethod().Get(); ok {
		cfg.Method = m
	}
	return cfg
}

func ToCreateServerRequest(req *api.CreateServerRequest) dto.CreateServerRequest {
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
	if v, ok := req.HTTPConfig.Get(); ok {
		dtoReq.HttpConfig = toHttpConfig(&v)
	}
	return dtoReq
}

func ToUpdateServerRequest(req *api.UpdateServerRequest) dto.UpdateServerRequest {
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
	if v, ok := req.HTTPConfig.Get(); ok {
		dtoReq.HttpConfig = toHttpConfig(&v)
	} else if req.HTTPConfig.IsSet() {
		dtoReq.ClearHttpConfig = true
	}
	return dtoReq
}

func ToTestEndpointRequest(req *api.TestEndpointRequest) dto.TestEndpointRequest {
	timeout := req.Timeout.Or(10)

	dtoReq := dto.TestEndpointRequest{
		Namespace: req.Namespace,
		ObjectID:  req.ObjectID,
		Kind:      string(req.Kind),
		Timeout:   time.Duration(timeout) * time.Second,
	}
	if v, ok := req.ContainerName.Get(); ok {
		dtoReq.ContainerName = v
	}
	if v, ok := req.HTTPConfig.Get(); ok {
		dtoReq.HttpConfig = toHttpConfig(&v)
	}
	return dtoReq
}

func ToAPIServer(s *dto.Server) api.ServerObject {

	if s == nil {
		return api.ServerObject{}
	}

	monitorStatus := api.OptNilServerObjectMonitorStatus{}
	if s.MonitorStatus != "" {
		monitorStatus.SetTo(api.ServerObjectMonitorStatus(s.MonitorStatus))
	}

	httpConfig := api.OptNilServerObjectHTTPConfig{}
	if s.HttpConfig != nil {
		httpConfig.SetTo(api.ServerObjectHTTPConfig{
			Port:          api.NewOptInt(s.HttpConfig.Port),
			EndpointPath:  api.NewOptString(s.HttpConfig.EndpointPath),
			ExpectedCode:  api.NewOptInt(s.HttpConfig.ExpectedCode),
			BodyCheckExpr: api.NewOptString(s.HttpConfig.BodyCheckExpr),
			Method:        api.NewOptString(s.HttpConfig.Method),
		})
	}

	return api.ServerObject{
		ID:            int(s.ID),
		Name:          s.Name,
		Namespace:     s.Namespace,
		Kind:          api.ServerObjectKind(s.Kind),
		ObjectID:      s.ObjectID,
		ContainerName: api.NewOptString(s.ContainerName),
		MonitorStatus: monitorStatus,
		HTTPConfig:    httpConfig,
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
