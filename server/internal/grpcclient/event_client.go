package grpcclient

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"google.golang.org/grpc"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type StatusClient interface {
	GetCurrentStatuses(ctx context.Context, endpointIDs []uint) (map[uint]domain.ServerStatus, error)
	CountByStatus(ctx context.Context, endpointIDs []uint) (online, offline int64, err error)
}

type EventClient struct {
	client eventv1.EventServiceClient
}

func NewEventClient(cc *grpc.ClientConn) *EventClient {
	return &EventClient{client: eventv1.NewEventServiceClient(cc)}
}

func RegisterEventClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (StatusClient, error) {
		client, err := newEventClient(i)
		if err != nil {
			return nil, err
		}
		return client, nil
	})
	do.Provide(i, func(i do.Injector) (*EventClient, error) {
		return newEventClient(i)
	})
}

func newEventClient(i do.Injector) (*EventClient, error) {
	wrapper := do.MustInvokeNamed[*config.GRPCClientWrapper](i, config.GRPCClientNameEvent)
	return NewEventClient(wrapper.GetConn()), nil
}

func (c *EventClient) GetCurrentStatuses(ctx context.Context, endpointIDs []uint) (map[uint]domain.ServerStatus, error) {
	if len(endpointIDs) == 0 {
		return nil, nil
	}

	ids := lo.Map(endpointIDs, func(id uint, _ int) uint64 { return uint64(id) })
	resp, err := c.client.GetCurrentStatuses(ctx, &eventv1.GetCurrentStatusesRequest{EndpointIds: ids})
	if err != nil {
		return nil, fmt.Errorf("get current statuses: %w", err)
	}

	return lo.SliceToMap(resp.Statuses, func(status *eventv1.EndpointStatus) (uint, domain.ServerStatus) {
		return uint(status.EndpointId), domain.ServerStatus(status.Status)
	}), nil
}

func (c *EventClient) CountByStatus(ctx context.Context, endpointIDs []uint) (online, offline int64, err error) {
	if len(endpointIDs) == 0 {
		return 0, 0, nil
	}

	ids := lo.Map(endpointIDs, func(id uint, _ int) uint64 { return uint64(id) })
	resp, err := c.client.CountByStatus(ctx, &eventv1.CountByStatusRequest{EndpointIds: ids})
	if err != nil {
		return 0, 0, fmt.Errorf("count by status: %w", err)
	}

	return resp.Online, resp.Offline, nil
}
