package grpcclient

import (
	"context"
	"fmt"

	eventv1 "github.com/minhnbnt/uptime-monitor-microservices/common/proto/generated/event/v1"
	"github.com/samber/do/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type EventRecorderClient struct {
	client eventv1.EventServiceClient
	conn   *grpc.ClientConn
}

func RegisterEventRecorderClient(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*EventRecorderClient, error) {
		cfg := do.MustInvoke[*config.Config](i)
		conn, err := grpc.NewClient(cfg.GRPC.EventAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, fmt.Errorf("dial event grpc: %w", err)
		}
		client := eventv1.NewEventServiceClient(conn)
		return &EventRecorderClient{client: client, conn: conn}, nil
	})
}

func (c *EventRecorderClient) Shutdown() error {
	return c.conn.Close()
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
