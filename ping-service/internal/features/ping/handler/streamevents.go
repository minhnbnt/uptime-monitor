package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"
	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/features/ping/scheduler"
)

const (
	streamKey        = "endpoint:events"
	consumerGroup    = "ping-service"
	consumerName     = "worker-1"
	streamReadCount  = 10
	streamBlockTime  = 5 * time.Second
)

type endpointLifecycleEvent struct {
	Type     string               `json:"type"`
	Endpoint streamEndpointData   `json:"endpoint"`
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

type StreamEventConsumer struct {
	client            *redis.Client
	scheduler         *scheduler.ZSetScheduleRepository
	logger            *slog.Logger
}

func RegisterStreamEventConsumer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*StreamEventConsumer, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &StreamEventConsumer{
			client:    wrapper.GetClient(),
			scheduler: do.MustInvoke[*scheduler.ZSetScheduleRepository](i),
			logger:    do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

func (c *StreamEventConsumer) Run(ctx context.Context) {
	c.logger.Info("starting stream consumer",
		slog.String("stream", streamKey),
		slog.String("group", consumerGroup),
	)

	if err := c.client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "$").Err(); err != nil {
		if err.Error() != "BUSYGROUP Consumer Group name already exists" {
			c.logger.Warn("create consumer group", slog.Any("error", err))
		}
	}

	for ctx.Err() == nil {
		streams, err := c.client.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumerName,
			Streams:  []string{streamKey, ">"},
			Count:    streamReadCount,
			Block:    streamBlockTime,
		}).Result()

		if err != nil {
			if err == redis.Nil {
				continue
			}
			c.logger.Error("stream read", slog.Any("error", err))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				c.processMessage(ctx, msg)
			}
		}
	}
}

func (c *StreamEventConsumer) processMessage(ctx context.Context, msg redis.XMessage) {
	payload, ok := msg.Values["payload"]
	if !ok {
		c.logger.Warn("stream message missing payload", slog.String("id", msg.ID))
		c.ack(ctx, msg.ID)
		return
	}

	payloadStr, ok := payload.(string)
	if !ok {
		c.logger.Warn("stream message payload not string", slog.String("id", msg.ID))
		c.ack(ctx, msg.ID)
		return
	}

	var event endpointLifecycleEvent
	if err := json.Unmarshal([]byte(payloadStr), &event); err != nil {
		c.logger.Warn("stream message invalid json",
			slog.String("id", msg.ID),
			slog.Any("error", err),
		)
		c.ack(ctx, msg.ID)
		return
	}

	interval := time.Duration(event.Endpoint.IntervalNs)

	switch event.Type {
	case "created", "updated":
		ep := domain.Endpoint{
			Model:        gorm.Model{ID: event.Endpoint.ID},
			ServerID:     event.Endpoint.ServerID,
			URL:          event.Endpoint.URL,
			Method:       event.Endpoint.Method,
			ExpectedCode: event.Endpoint.ExpectedCode,
			Interval:     interval,
		}
		if err := c.scheduler.Register(ctx, &ep); err != nil {
			c.logger.Error("register endpoint",
				slog.Uint64("endpoint_id", uint64(ep.ID)),
				slog.Any("error", err),
			)
			return
		}

	case "deleted":
		if err := c.scheduler.Unregister(ctx, event.Endpoint.ID); err != nil {
			c.logger.Error("unregister endpoint",
				slog.Uint64("endpoint_id", uint64(event.Endpoint.ID)),
				slog.Any("error", err),
			)
			return
		}

	default:
		c.logger.Warn("unknown event type", slog.String("type", event.Type))
	}

	c.ack(ctx, msg.ID)
}

func (c *StreamEventConsumer) ack(ctx context.Context, msgID string) {
	if err := c.client.XAck(ctx, streamKey, consumerGroup, msgID).Err(); err != nil {
		c.logger.Error("ack message",
			slog.String("msg_id", msgID),
			slog.Any("error", err),
		)
	}
}
