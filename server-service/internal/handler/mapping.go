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

func ToCreateK8sObjectRequest(req *api.CreateK8sObjectRequest) dto.CreateK8sObjectRequest {
	dtoReq := dto.CreateK8sObjectRequest{
		Name:      req.Name,
		Namespace: req.Namespace,
		ObjectID:  req.ObjectID,
	}
	for _, c := range req.Containers {
		dtoReq.Containers = append(dtoReq.Containers, dto.Container{Name: c.Name, Image: c.Image})
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
		cfg := api.ServerObjectHTTPConfig{
			Port:         api.NewOptInt(s.HttpConfig.Port),
			EndpointPath: api.NewOptString(s.HttpConfig.EndpointPath),
		}
		if s.HttpConfig.ExpectedCode > 0 {
			cfg.ExpectedCode = api.NewOptInt(s.HttpConfig.ExpectedCode)
		}
		if s.HttpConfig.BodyCheckExpr != "" {
			cfg.BodyCheckExpr = api.NewOptString(s.HttpConfig.BodyCheckExpr)
		}
		if s.HttpConfig.Method != "" {
			cfg.Method = api.NewOptString(s.HttpConfig.Method)
		}
		httpConfig.SetTo(cfg)
	}

	return api.ServerObject{
		ID:            int(s.ID),
		Name:          s.Name,
		Namespace:     s.Namespace,
		Kind:          api.ServerObjectKind(s.Kind),
		ObjectID:      s.ObjectID,
		ContainerName: api.NewOptString(s.ContainerName),
		Managed:       api.NewOptBool(s.Managed),
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
