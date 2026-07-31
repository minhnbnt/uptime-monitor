package redis

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/redis/go-redis/v9"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

func xmessage(id, value string) redis.XMessage {
	return redis.XMessage{
		ID: id,
		Values: map[string]any{
			"value": value,
		},
	}
}

func TestProcessMessage(t *testing.T) {
	ctx := context.Background()

	t.Run("missing value field warns and skips handler", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		var called bool
		p := &messageProcessor{
			handler: &mockServerEventHandler{
				onMessageFn: func(_ context.Context, _ *dto.DebeziumMessage) error {
					called = true
					return nil
				},
			},
			logger: log,
		}

		p.ProcessMessage(ctx, "uptime.public.servers", redis.XMessage{ID: "1-0", Values: map[string]any{}})
		if called {
			t.Error("handler should not be called")
		}
		if !capLog.HasWarn() {
			t.Error("expected warn log")
		}
	})

	t.Run("value not a string warns and skips handler", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		var called bool
		p := &messageProcessor{
			handler: &mockServerEventHandler{
				onMessageFn: func(_ context.Context, _ *dto.DebeziumMessage) error {
					called = true
					return nil
				},
			},
			logger: log,
		}

		p.ProcessMessage(ctx, "uptime.public.servers", redis.XMessage{ID: "1-0", Values: map[string]any{"value": 42}})
		if called {
			t.Error("handler should not be called")
		}
		if !capLog.HasWarn() {
			t.Error("expected warn log")
		}
	})

	t.Run("invalid JSON logs error and skips handler", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		var called bool
		p := &messageProcessor{
			handler: &mockServerEventHandler{
				onMessageFn: func(_ context.Context, _ *dto.DebeziumMessage) error {
					called = true
					return nil
				},
			},
			logger: log,
		}

		p.ProcessMessage(ctx, "uptime.public.servers", xmessage("1-0", "not json"))
		if called {
			t.Error("handler should not be called")
		}
		if !capLog.HasError() {
			t.Error("expected error log")
		}
	})

	t.Run("valid event parses envelope and calls handler", func(t *testing.T) {
		log, _ := logger.NewCapturingLogger()
		var got *dto.DebeziumMessage
		p := &messageProcessor{
			handler: &mockServerEventHandler{
				onMessageFn: func(_ context.Context, event *dto.DebeziumMessage) error {
					got = event
					return nil
				},
			},
			logger: log,
		}

		p.ProcessMessage(ctx, "uptime.public.servers", xmessage("1-0", `{"op":"c","after":{"id":1,"namespace":"default"}}`))
		if got == nil {
			t.Fatal("handler was not called")
		}
		if got.TopicName != "uptime.public.servers" {
			t.Errorf("TopicName = %q, want %q", got.TopicName, "uptime.public.servers")
		}
		if got.Operation != "c" {
			t.Errorf("Operation = %q, want c", got.Operation)
		}
		var sv map[string]any
		if err := json.Unmarshal(got.After, &sv); err != nil {
			t.Fatalf("unmarshal after: %v", err)
		}
		if sv["id"].(float64) != 1 {
			t.Errorf("after.id = %v, want 1", sv["id"])
		}
	})
}
