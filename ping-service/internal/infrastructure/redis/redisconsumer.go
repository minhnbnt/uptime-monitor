package redis

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/samber/do/v2"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/config"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

const (
	consumerGroup   = "ping-service"
	consumerName    = "worker-1"
	streamReadCount = 10
	streamBlockTime = 5 * time.Second
)

type StreamEventConsumer struct {
	client *redis.Client
	logger *slog.Logger
}

func RegisterStreamEventConsumer(i do.Injector) {
	do.Provide(i, func(i do.Injector) (*StreamEventConsumer, error) {
		wrapper := do.MustInvoke[*config.RedisClientWrapper](i)
		return &StreamEventConsumer{
			client: wrapper.GetClient(),
			logger: do.MustInvoke[*slog.Logger](i),
		}, nil
	})
}

type ServerEventHandler interface {
	OnMessage(context.Context, *dto.DebeziumMessage) error
}

func (c *StreamEventConsumer) Run(ctx context.Context, streamKeys []string, handler ServerEventHandler) {

	c.logger.Info(
		"starting stream consumer",
		slog.String("stream", strings.Join(streamKeys, ", ")),
		slog.String("group", consumerGroup),
	)

	for _, streamKey := range streamKeys {
		err := c.client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "$").Err()
		if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
			c.logger.Warn("create consumer group", slog.Any("error", err))
		}
	}

	processor := &messageProcessor{
		handler: handler,
		logger:  c.logger,
	}

	streams := make([]string, 0, len(streamKeys))
	for _, streamKey := range streamKeys {
		streams = append(streams, streamKey, ">")
	}

	args := redis.XReadGroupArgs{
		Group:    consumerGroup,
		Consumer: consumerName,
		Streams:  streams,
		Count:    streamReadCount,
		Block:    streamBlockTime,
	}

	for ctx.Err() == nil {

		streams, err := c.client.XReadGroup(ctx, &args).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}

		if err != nil {
			c.logger.Error("stream read", slog.Any("error", err))
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				processor.ProcessMessage(ctx, stream.Stream, msg)
			}
		}
	}
}

func (c *StreamEventConsumer) Ack(ctx context.Context, message *dto.DebeziumMessage) error {
	cmd := c.client.XAck(ctx, message.TopicName, consumerGroup, message.ID)
	return cmd.Err()
}
