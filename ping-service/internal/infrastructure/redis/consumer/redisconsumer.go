package consumer

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
	consumerGroup     = "ping-service"
	consumerName      = "worker-1"
	streamReadCount   = 10
	streamBlockTime   = 5 * time.Second
	streamIdleReclaim = time.Minute
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

	streams := make([]string, 0, 2*len(streamKeys))
	streams = append(streams, streamKeys...)
	for range streamKeys {
		streams = append(streams, ">")
	}

	args := redis.XReadGroupArgs{
		Streams:  streams,
		Consumer: consumerName,
		Group:    consumerGroup,
		Count:    streamReadCount,
		Block:    streamBlockTime,
	}

	for ctx.Err() == nil {
		c.claim(ctx, &args, processor)
		c.reclaimIdle(ctx, streamKeys, processor)
	}
}

func (c *StreamEventConsumer) claim(ctx context.Context, args *redis.XReadGroupArgs, processor *messageProcessor) {

	streams, err := c.client.XReadGroup(ctx, args).Result()
	if errors.Is(err, redis.Nil) {
		return
	}

	if ctx.Err() != nil {
		return
	}

	if err != nil {
		c.logger.Error("stream read", slog.Any("error", err))
		time.Sleep(time.Second)
		return
	}

	for _, stream := range streams {
		for _, msg := range stream.Messages {
			processor.ProcessMessage(ctx, stream.Stream, msg)
		}
	}
}

func (c *StreamEventConsumer) reclaimIdle(ctx context.Context, streamKeys []string, processor *messageProcessor) {

	for _, streamKey := range streamKeys {

		args := redis.XAutoClaimArgs{
			Stream:   streamKey,
			Group:    consumerGroup,
			Consumer: consumerName,
			MinIdle:  streamIdleReclaim,
			Start:    "0",
			Count:    streamReadCount,
		}

		claimed, _, err := c.client.XAutoClaim(ctx, &args).Result()

		if err != nil {

			c.logger.Warn(
				"reclaim idle messages",
				slog.String("stream", streamKey),
				slog.Any("error", err),
			)

			continue
		}

		for _, msg := range claimed {
			processor.ProcessMessage(ctx, streamKey, msg)
		}
	}
}

func (c *StreamEventConsumer) Ack(ctx context.Context, message *dto.DebeziumMessage) error {
	cmd := c.client.XAck(ctx, message.TopicName, consumerGroup, message.ID)
	return cmd.Err()
}
