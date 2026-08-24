package repository

import (
	"sync"
	"testing"
	"time"

	"github.com/samber/lo"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure/testcontainers"
)

func startRateLimiter(tb testing.TB) *PushRateLimiter {
	tb.Helper()

	testcontainers.SkipIfShort(tb)
	ctx := tb.Context()

	container, addr := testcontainers.StartRedisAddr(ctx)
	tb.Cleanup(func() { _ = container.Terminate(ctx) })

	client := testcontainers.NewTestRedis(tb, addr)
	return NewPushRateLimiter(client)
}

func TestPushRateLimiter(t *testing.T) {

	const interval = 30 * time.Second
	sid := "sess-ratelimit"

	t.Run("first send passes and sets milestone", func(t *testing.T) {

		limiter := startRateLimiter(t)

		next, allowed, err := limiter.Allow(t.Context(), sid, interval)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("first send must pass")
		}
		if next.Before(time.Now()) {
			t.Errorf("next = %v, want future milestone", next)
		}

		ttl, err := limiter.client.TTL(t.Context(), pushNextKey(sid)).Result()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ttl <= 0 || ttl > interval*2 {
			t.Errorf("ttl = %v, want within (0, %v]", ttl, interval*2)
		}
	})

	t.Run("early resend blocked at same milestone", func(t *testing.T) {

		limiter := startRateLimiter(t)

		first, allowed, err := limiter.Allow(t.Context(), sid, interval)
		if err != nil || !allowed {
			t.Fatalf("first send: allowed=%v err=%v", allowed, err)
		}

		next, allowed, err := limiter.Allow(t.Context(), sid, interval)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("early resend must be blocked")
		}
		if !next.Equal(first) {
			t.Errorf("blocked next = %v, want first milestone %v", next, first)
		}
	})

	t.Run("past milestone passes again", func(t *testing.T) {

		limiter := startRateLimiter(t)

		past := time.Now().Add(-time.Minute)
		err := limiter.client.Set(t.Context(), pushNextKey(sid), past.UnixMilli(), interval).Err()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		_, allowed, err := limiter.Allow(t.Context(), sid, interval)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("send after milestone must pass")
		}
	})

	t.Run("expired milestone passes without waiting full ttl", func(t *testing.T) {

		limiter := startRateLimiter(t)

		err := limiter.client.Set(
			t.Context(),
			pushNextKey(sid),
			time.Now().Add(time.Minute).UnixMilli(),
			30*time.Millisecond,
		).Err()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		time.Sleep(50 * time.Millisecond)

		_, allowed, err := limiter.Allow(t.Context(), sid, interval)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("send after expiry must pass")
		}
	})

	t.Run("parallel same session single admission", func(t *testing.T) {

		limiter := startRateLimiter(t)

		const n = 10
		results := make([]bool, n)
		waitgroup := sync.WaitGroup{}
		start := make(chan struct{})

		for i := range n {
			waitgroup.Go(func() {
				<-start
				_, allowed, err := limiter.Allow(t.Context(), sid, interval)
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				results[i] = allowed
			})
		}

		close(start)
		waitgroup.Wait()

		admitted := lo.Count(results, true)
		if admitted != 1 {
			t.Errorf("admitted = %d, want exactly 1", admitted)
		}
	})

	t.Run("release clears the milestone", func(t *testing.T) {

		limiter := startRateLimiter(t)

		if _, allowed, err := limiter.Allow(t.Context(), sid, interval); err != nil || !allowed {
			t.Fatalf("first send: allowed=%v err=%v", allowed, err)
		}

		if _, allowed, _ := limiter.Allow(t.Context(), sid, interval); allowed {
			t.Fatal("second immediate send must be blocked before release")
		}

		if err := limiter.Release(t.Context(), sid); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		exists, err := limiter.client.Exists(t.Context(), pushNextKey(sid)).Result()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if exists != 0 {
			t.Fatal("milestone key must be gone after release")
		}

		if _, allowed, err := limiter.Allow(t.Context(), sid, interval); err != nil || !allowed {
			t.Fatalf("send after release: allowed=%v err=%v", allowed, err)
		}
	})
}
