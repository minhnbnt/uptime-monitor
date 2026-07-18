package grpcclient

import (
	"context"
	"fmt"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type EventRecorderClient struct {
	client  eventv1.EventRecorderServiceClient
	wrapper *config.GRPCClientWrapper
}

func NewEventRecorderClient(wrapper *config.GRPCClientWrapper) *EventRecorderClient {
	return &EventRecorderClient{
		client:  eventv1.NewEventRecorderServiceClient(wrapper.GetConn()),
		wrapper: wrapper,
	}
}

func RegisterEventRecorderClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventRecorderClient, error) {
		cfg := do.MustInvoke[*config.Config](i)
		wrapper, err := config.NewGRPCClientWrapper(cfg.GRPC.EventAddr)
		if err != nil {
			return nil, fmt.Errorf("dial event grpc: %w", err)
		}
		return NewEventRecorderClient(wrapper), nil
	})
}

func (c *EventRecorderClient) Shutdown() error {
	return c.wrapper.Shutdown()
}

func (c *EventRecorderClient) RecordEvent(ctx context.Context, endpointID uint, status domain.ServerStatus) error {

	_, err := c.client.RecordEvent(ctx, &eventv1.RecordEventRequest{
		EndpointId: uint64(endpointID),
		Status:     string(status),
	})

	if err != nil {
		return fmt.Errorf("record event: %w", err)
	}

	return nil
}
