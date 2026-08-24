package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/common/authclient"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
	pingservice "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/service"
)

type stubResolver struct {
	owned map[uint64]bool
}

func (s *stubResolver) ResolveServers(_ context.Context, _ uint, ids []uint64) ([]uint64, error) {

	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if s.owned[id] {
			result = append(result, id)
		}
	}

	return result, nil
}

type stubGate struct {
	allowed bool
	next    time.Time
}

func (g *stubGate) Allow(_ context.Context, _ string, _ time.Duration) (time.Time, bool, error) {
	return g.next, g.allowed, nil
}

func (g *stubGate) Release(_ context.Context, _ string) error {
	return nil
}

type stubRecorder struct{}

func (r *stubRecorder) Record(_ context.Context, _ *domain.ServerEvent) error { return nil }

func newPushTestServer(tb testing.TB, resolver *stubResolver, gate *stubGate) *httptest.Server {
	tb.Helper()

	svc := pingservice.NewPushEventService(resolver, gate, &stubRecorder{}, logger.NewMockLogger())

	apiServer, err := api.NewServer(&PushEventHandler{
		pushService: svc,
		logger:      logger.NewMockLogger(),
	})
	if err != nil {
		tb.Fatalf("new api server: %v", err)
	}

	middleware := authclient.NewAuthMiddleware(logger.NewMockLogger())
	ts := httptest.NewServer(
		middleware.XUserIDMiddleware(middleware.RequireScope("ping")(apiServer)),
	)
	tb.Cleanup(ts.Close)

	return ts
}

func postEvents(tb testing.TB, ts *httptest.Server, userID, sid, scopes, body string) (int, string) {
	tb.Helper()

	req, err := http.NewRequestWithContext(
		tb.Context(),
		http.MethodPost,
		ts.URL+"/api/v1/ping/events",
		strings.NewReader(body),
	)
	if err != nil {
		tb.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if userID != "" {
		req.Header.Set("X-User-ID", userID)
	}
	if sid != "" {
		req.Header.Set("X-Session-ID", sid)
	}
	if scopes != "" {
		req.Header.Set("X-Scopes", scopes)
	}

	res, err := ts.Client().Do(req)
	if err != nil {
		tb.Fatalf("do request: %v", err)
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		tb.Fatalf("read body: %v", err)
	}

	return res.StatusCode, string(raw)
}

const pushHeaders = "7"

var futureNext = time.Now().Add(time.Minute)

func TestPushEventsEndpoint(t *testing.T) {

	t.Run("missing scope rejected with 403", func(t *testing.T) {

		ts := newPushTestServer(t, &stubResolver{}, &stubGate{allowed: true})

		code, _ := postEvents(t, ts, pushHeaders, "sess-1", "app", `[{"id":1,"status":"ON"}]`)
		if code != http.StatusForbidden {
			t.Errorf("code = %d, want 403", code)
		}
	})

	t.Run("missing session id rejected with 403", func(t *testing.T) {

		ts := newPushTestServer(t, &stubResolver{}, &stubGate{allowed: true})

		code, _ := postEvents(t, ts, pushHeaders, "", "ping", `[{"id":1,"status":"ON"}]`)
		if code != http.StatusForbidden {
			t.Errorf("code = %d, want 403", code)
		}
	})

	t.Run("valid batch accepted with 200", func(t *testing.T) {

		ts := newPushTestServer(
			t,
			&stubResolver{owned: map[uint64]bool{1: true}},
			&stubGate{allowed: true, next: futureNext},
		)

		code, body := postEvents(t, ts, pushHeaders, "sess-1", "ping", `[{"id":1,"status":"ON"}]`)
		if code != http.StatusOK {
			t.Fatalf("code = %d, want 200, body %s", code, body)
		}

		if !strings.Contains(body, `"accepted":[1]`) || !strings.Contains(body, `"errors":[]`) {
			t.Errorf("body = %s, want accepted [1] and empty errors", body)
		}
	})

	t.Run("early resend rejected with 429 and next_time", func(t *testing.T) {

		ts := newPushTestServer(t, &stubResolver{owned: map[uint64]bool{1: true}}, &stubGate{next: futureNext})

		code, body := postEvents(t, ts, pushHeaders, "sess-1", "ping", `[{"id":1,"status":"ON"}]`)
		if code != http.StatusTooManyRequests {
			t.Fatalf("code = %d, want 429, body %s", code, body)
		}

		want := `"next_time":` + strconv.FormatInt(futureNext.UnixMilli(), 10)
		if !strings.Contains(body, want) {
			t.Errorf("body = %s, want containing %s", body, want)
		}
	})

	t.Run("unknown id reported via 207", func(t *testing.T) {

		ts := newPushTestServer(
			t,
			&stubResolver{owned: map[uint64]bool{1: true}},
			&stubGate{allowed: true, next: futureNext},
		)

		code, body := postEvents(
			t, ts, pushHeaders, "sess-1", "ping",
			`[{"id":1,"status":"ON"},{"id":99,"status":"OFF"}]`,
		)
		if code != http.StatusMultiStatus {
			t.Fatalf("code = %d, want 207, body %s", code, body)
		}

		if !strings.Contains(body, `"error":"not found"`) {
			t.Errorf("body = %s, want not found entry", body)
		}
	})

	t.Run("empty array rejected with 400", func(t *testing.T) {

		ts := newPushTestServer(t, &stubResolver{}, &stubGate{allowed: true})

		code, _ := postEvents(t, ts, pushHeaders, "sess-1", "ping", `[]`)
		if code != http.StatusBadRequest {
			t.Errorf("code = %d, want 400", code)
		}
	})
}
