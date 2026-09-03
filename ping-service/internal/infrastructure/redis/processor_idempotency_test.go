package redis

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

func TestProcessMessageSkipsStaleAndAppliesFresh(t *testing.T) {
	testcontainers.SkipIfShort(t)
	ctx := context.Background()
	client := testcontainers.NewTestRedis(t, testRedisAddr)

	store := NewOffsetStore(client, 30*time.Minute)
	if err := store.SetOffset(ctx, 1, "2-0"); err != nil {
		t.Fatalf("set offset: %v", err)
	}

	var mu sync.Mutex
	var created domain.Endpoint
	h := &mockEndpointEventHandler{
		onCreateFn: func(_ context.Context, ep domain.Endpoint) error {
			mu.Lock()
			created = ep
			mu.Unlock()
			return nil
		},
	}
	p := &messageProcessor{
		handler: h,
		logger:  logger.NewMockLogger(),
		client:  client,
		offsets: store,
	}

	// stale message (id 1-0 < applied 2-0) -> acked, handler NOT called
	if !p.ProcessMessage(ctx, xmessage("1-0", `{"payload":{"op":"c","after":{"id":1,"url":"https://x.com","method":"GET","expected_code":200,"interval":5000000000,"timeout":2000000000}}}`)) {
		t.Error("expected canAck=true for stale message")
	}
	mu.Lock()
	if created.ID != 0 {
		t.Errorf("stale message must not call handler, got endpoint id %d", created.ID)
	}
	mu.Unlock()

	// fresh message (id 3-0 > applied 2-0) -> acked, handler called
	if !p.ProcessMessage(ctx, xmessage("3-0", `{"payload":{"op":"c","after":{"id":1,"url":"https://x.com","method":"GET","expected_code":200,"interval":5000000000,"timeout":2000000000}}}`)) {
		t.Error("expected canAck=true for fresh message")
	}
	mu.Lock()
	if created.ID != 1 {
		t.Errorf("fresh message must call handler with endpoint id 1, got %d", created.ID)
	}
	mu.Unlock()
}

func TestOffsetStoreIsNewer(t *testing.T) {
	s := &OffsetStore{}

	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"greater ms", "2-0", "1-0", true},
		{"equal ms greater seq", "1-2", "1-1", true},
		{"equal", "1-1", "1-1", false},
		{"less", "1-0", "2-0", false},
	}

	for _, c := range cases {
		got, err := s.IsNewer(c.a, c.b)
		if err != nil {
			t.Fatalf("%s: IsNewer(%q,%q): %v", c.name, c.a, c.b, err)
		}
		if got != c.want {
			t.Errorf("%s: IsNewer(%q,%q)=%v, want %v", c.name, c.a, c.b, got, c.want)
		}
	}
}
