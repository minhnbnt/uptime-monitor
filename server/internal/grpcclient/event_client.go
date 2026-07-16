package grpcclient

import (
	"context"
	"fmt"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/minhnbnt/uptime-monitor/internal/config"
	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

type StatusClient interface {
	GetCurrentStatuses(ctx context.Context, endpointIDs []uint) (map[uint]domain.ServerStatus, error)
	CountByStatus(ctx context.Context, endpointIDs []uint) (online, offline int64, err error)
}

type EventClient struct {
	client eventv1.EventServiceClient
	conn   *grpc.ClientConn
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
	cfg := do.MustInvoke[*config.Config](i)
	conn, err := grpc.NewClient(cfg.GRPC.EventAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("dial event grpc: %w", err)
	}
	return &EventClient{
		client: eventv1.NewEventServiceClient(conn),
		conn:   conn,
	}, nil
}

func (c *EventClient) Shutdown() error {
	return c.conn.Close()
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

	m := make(map[uint]domain.ServerStatus, len(resp.Statuses))
	for _, s := range resp.Statuses {
		m[uint(s.EndpointId)] = domain.ServerStatus(s.Status)
	}
	return m, nil
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
