package grpcclient

import (
	"context"
	"fmt"

	"github.com/samber/do/v2"
	"github.com/samber/lo"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
)

type StatusClient interface {
	GetCurrentStatuses(ctx context.Context, endpointIDs []uint) (
		map[uint]domain.ServerStatus, error,
	)
	CountByStatus(ctx context.Context, userID uint) (
		online, offline int64, err error,
	)
}

type EventClient struct {
	client eventv1.StatusServiceClient
}

func NewEventClient(cc *config.GRPCClientWrapper) *EventClient {
	return &EventClient{client: eventv1.NewStatusServiceClient(cc.GetConn())}
}

func newEventClient(i do.Injector) (*EventClient, error) {

	cfg := do.MustInvoke[*config.Config](i)
	addr := cfg.GRPC.EventAddr
	if addr == "" {
		addr = "localhost:50052"
	}

	wrapper, err := config.NewGRPCClientWrapper(addr)
	if err != nil {
		return nil, fmt.Errorf("event gRPC client: %w", err)
	}

	return NewEventClient(wrapper), nil
}

func RegisterEventClient(i do.Injector) {

	do.Provide(i, func(i do.Injector) (StatusClient, error) {
		return newEventClient(i)
	})

	do.Provide(i, func(i do.Injector) (*EventClient, error) {
		return newEventClient(i)
	})
}

func (c *EventClient) GetCurrentStatuses(
	ctx context.Context,
	endpointIDs []uint,
) (map[uint]domain.ServerStatus, error) {

	if len(endpointIDs) == 0 {
		return nil, nil
	}

	ids := lo.Map(endpointIDs, func(id uint, _ int) uint64 { return uint64(id) })
	request := eventv1.GetCurrentStatusesRequest{EndpointIds: ids}
	resp, err := c.client.GetCurrentStatuses(ctx, &request)
	if err != nil {
		return nil, fmt.Errorf("get current statuses: %w", err)
	}

	return lo.SliceToMap(
		resp.Statuses,
		func(status *eventv1.EndpointStatus) (uint, domain.ServerStatus) {
			return uint(status.EndpointId), domain.ServerStatus(status.Status)
		},
	), nil
}

func (c *EventClient) CountByStatus(ctx context.Context, userID uint) (online, offline int64, err error) {

	request := eventv1.CountByStatusRequest{UserId: uint64(userID)}
	resp, err := c.client.CountByStatus(ctx, &request)
	if err != nil {
		return 0, 0, fmt.Errorf("count by status: %w", err)
	}

	return resp.Online, resp.Offline, nil
}
