package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/domain"
)

const streamKey = "endpoint:events"

type endpointLifecycleEvent struct {
	Type     string             `json:"type"`
	Endpoint streamEndpointData `json:"endpoint"`
}

type streamEndpointData struct {
	ID           uint   `json:"id"`
	ServerID     uint   `json:"server_id"`
	URL          string `json:"url"`
	Method       string `json:"method"`
	ExpectedCode int    `json:"expected_code"`
	IntervalNs   int64  `json:"interval_ns"`
	TimeoutNs    int64  `json:"timeout_ns"`
}

func streamEndpointFromDomain(ep *domain.Endpoint) streamEndpointData {
	return streamEndpointData{
		ID:           ep.ID,
		ServerID:     ep.ServerID,
		URL:          ep.URL,
		Method:       ep.Method,
		ExpectedCode: ep.ExpectedCode,
		IntervalNs:   ep.Interval.Nanoseconds(),
		TimeoutNs:    ep.Timeout.Nanoseconds(),
	}
}

type StreamEventPublisher struct {
	client *redis.Client
}

func RegisterStreamEventPublisher(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*StreamEventPublisher, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &StreamEventPublisher{client: wrapper.GetClient()}, nil
	})
}

func (p *StreamEventPublisher) Publish(ctx context.Context, eventType string, ep *domain.Endpoint) error {
	event := endpointLifecycleEvent{
		Type:     eventType,
		Endpoint: streamEndpointFromDomain(ep),
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]any{"payload": string(payload)},
	}).Err()
}
