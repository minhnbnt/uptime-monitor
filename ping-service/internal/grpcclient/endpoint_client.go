package grpcclient

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"

	endpointv1 "github.com/minhnbnt/uptime-monitor-microservices/proto/gen/endpoint/v1"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
)

type EndpointClient struct {
	conn   *grpc.ClientConn
	client endpointv1.EndpointServiceClient
}

func NewEndpointClient(host string) (*EndpointClient, error) {

	conn, err := grpc.NewClient(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}

	return &EndpointClient{
		conn:   conn,
		client: endpointv1.NewEndpointServiceClient(conn),
	}, nil
}

func (c *EndpointClient) Close() error {
	return c.conn.Close()
}

func (c *EndpointClient) UpdateMonitorStatus(ctx context.Context, endpointID uint, status domain.ServerStatus) error {

	_, err := c.client.UpdateMonitorStatus(ctx, &endpointv1.UpdateMonitorStatusRequest{
		EndpointId: uint64(endpointID),
		Status:     string(status),
	})

	if err != nil {
		return fmt.Errorf("update monitor status: %w", err)
	}

	return nil
}

func (c *EndpointClient) GetBatch(ctx context.Context, ids []uint) (map[uint]*domain.Endpoint, error) {

	endpointIDs := make([]uint64, len(ids))
	for i, id := range ids {
		endpointIDs[i] = uint64(id)
	}

	resp, err := c.client.GetEndpoints(ctx, &endpointv1.GetEndpointsRequest{
		EndpointIds: endpointIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get endpoints: %w", err)
	}

	result := make(map[uint]*domain.Endpoint, len(resp.Endpoints))
	for _, ep := range resp.Endpoints {
		result[uint(ep.Id)] = &domain.Endpoint{
			Model:        gorm.Model{ID: uint(ep.Id)},
			ServerID:     uint(ep.ServerId),
			URL:          ep.Url,
			Method:       ep.Method,
			ExpectedCode: int(ep.ExpectedCode),
			Interval:     time.Duration(ep.IntervalMs) * time.Millisecond,
			Timeout:      time.Duration(ep.TimeoutMs) * time.Millisecond,
			MonitorStatus: domain.ServerStatus(ep.MonitorStatus),
		}
	}

	return result, nil
}
