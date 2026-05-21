package server

import (
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor/internal/server/handler"
)

type CompositeHandler struct {
	*handler.ServerHandler
	*handler.EndpointHandler
}

func RegisterCompositeHandler(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*CompositeHandler, error) {
		return &CompositeHandler{
			ServerHandler:   do.MustInvoke[*handler.ServerHandler](i),
			EndpointHandler: do.MustInvoke[*handler.EndpointHandler](i),
		}, nil
	})
}
