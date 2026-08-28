package redis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

func TestStreamEventConsumerRunDeadLettersPoisonAndProcessesValid(t *testing.T) {
	testcontainers.SkipIfShort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := testcontainers.NewTestRedis(t, testRedisAddr)

	// anchor the consumer group at "$" before publishing so new messages are read
	if err := client.XGroupCreateMkStream(ctx, streamKey, consumerGroup, "$").Err(); err != nil {
		t.Fatalf("create group: %v", err)
	}

	var mu sync.Mutex
	var created domain.Endpoint
	var handled int
	handler := &mockEndpointEventHandler{
		onCreateFn: func(_ context.Context, ep domain.Endpoint) error {
			mu.Lock()
			created = ep
			handled++
			mu.Unlock()
			return nil
		},
	}

	consumer := &StreamEventConsumer{client: client, logger: logger.NewMockLogger()}

	go consumer.Run(ctx, handler)

	if err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]any{"value": "not json"},
	}).Err(); err != nil {
		t.Fatalf("publish poison: %v", err)
	}
	if err := client.XAdd(ctx, &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]any{"value": `{"op":"c","after":{"id":7,"url":"https://x.com","method":"GET","expected_code":200,"interval":5000000000,"timeout":2000000000}}`},
	}).Err(); err != nil {
		t.Fatalf("publish valid: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	for {
		mu.Lock()
		gotCreated := created
		gotHandled := handled
		mu.Unlock()

		dlq, err := client.XLen(ctx, dlqStreamKey).Result()
		if err != nil {
			t.Fatalf("xlen dlq: %v", err)
		}

		if dlq == 1 && gotHandled >= 1 && gotCreated.ID == 7 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("timeout: dlq=%d handled=%d createdID=%d", dlq, gotHandled, gotCreated.ID)
		}
		time.Sleep(100 * time.Millisecond)
	}

	entries, err := client.XRange(ctx, dlqStreamKey, "-", "+").Result()
	if err != nil {
		t.Fatalf("xrange dlq: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("dlq entries = %d, want 1 (only the poison message)", len(entries))
	}
	if entries[0].Values["original_id"] == "" {
		t.Error("dlq entry missing original_id")
	}
	if _, ok := entries[0].Values["error"]; !ok {
		t.Error("dlq entry missing error field")
	}
}
