package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

type mockPingWorker struct {
	checkObjectStatusFn func(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error)
}

func (m *mockPingWorker) CheckObjectStatus(ctx context.Context, params *dto.K8sObjectCheckParams) (bool, error) {
	return m.checkObjectStatusFn(ctx, params)
}

type mockURLResolver struct {
	resolveFn       func(ctx context.Context, params *dto.CheckParams) (*url.URL, error)
	resolveDomainFn func(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error)
}

func (m *mockURLResolver) ResolveURL(ctx context.Context, params *dto.CheckParams) (*url.URL, error) {
	return m.resolveFn(ctx, params)
}

func (m *mockURLResolver) ResolveDomain(ctx context.Context, params *dto.K8sObjectCheckParams) (string, error) {
	if m.resolveDomainFn == nil {
		return "", nil
	}
	return m.resolveDomainFn(ctx, params)
}

type mockDomainCache struct {
	deleteFn func(ctx context.Context, id uint) error
}

func (m *mockDomainCache) Delete(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

type mockRecordWorker struct {
	recordFn func(ctx context.Context, event *domain.ServerEvent) error
}

func (m *mockRecordWorker) Record(ctx context.Context, event *domain.ServerEvent) error {
	return m.recordFn(ctx, event)
}

func newTestPingLoop(worker pingWorker) *PingLoopService {
	return &PingLoopService{
		pingWorker: worker,
		urlResolver: &mockURLResolver{
			resolveFn: func(_ context.Context, _ *dto.CheckParams) (*url.URL, error) {
				return nil, errors.New("unexpected url resolve")
			},
		},
		pingClient:      infrastructure.NewPingClient(&http.Client{}),
		responseChecker: &ResponseChecker{bodyChecker: &infrastructure.BodyChecker{}},
		logger:          logger.NewMockLogger(),
	}
}

func TestPingAndRecordServer(t *testing.T) {
	sv := &domain.Server{
		ID:        1,
		Namespace: "default",
		Kind:      "Pod",
		ObjectID:  "web-app",
		Interval:  30 * time.Second,
	}

	t.Run("successful ping sets StatusOn and updates score", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		var updatedScore int64
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		})
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, nextScore int64) error {
				updatedScore = nextScore
				return nil
			},
		}

		task := PingTask{Server: sv}
		s.pingAndRecordServer(t.Context(), task)
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOn {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOn)
		}
		if recordedEvent.ServerID != 1 {
			t.Errorf("serverID = %d, want 1", recordedEvent.ServerID)
		}
		if updatedScore <= 0 {
			t.Errorf("expected positive updated score, got %d", updatedScore)
		}
	})

	t.Run("ping error sets StatusOff", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		log, capLog := logger.NewCapturingLogger()
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return false, errors.New("connection refused")
			},
		})
		s.logger = log
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOff {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOff)
		}
		if !capLog.HasWarn() {
			t.Error("expected warn log for ping failure")
		}
	})

	t.Run("pod not running sets StatusOff", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return false, nil
			},
		})
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, event *domain.ServerEvent) error {
				recordedEvent = event
				return nil
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOff {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOff)
		}
	})

	t.Run("record error is logged but not returned", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		})
		s.logger = log
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, _ *domain.ServerEvent) error {
				return errors.New("grpc error")
			},
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if !capLog.HasError() {
			t.Error("expected error log for record failure")
		}
	})

	t.Run("score update error is logged but not returned", func(t *testing.T) {
		log, capLog := logger.NewCapturingLogger()
		s := newTestPingLoop(&mockPingWorker{
			checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
				return true, nil
			},
		})
		s.logger = log
		s.recordStatusWorker = &mockRecordWorker{
			recordFn: func(_ context.Context, _ *domain.ServerEvent) error { return nil },
		}
		s.scoreUpdater = &mockScoreUpdater{
			updateFn: func(_ context.Context, _ uint, _ int64) error {
				return errors.New("redis error")
			},
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if !capLog.HasError() {
			t.Error("expected error log for score update failure")
		}
	})

	t.Run("http-dns server pings HTTP endpoint and checks response", func(t *testing.T) {
		var recordedEvent *domain.ServerEvent
		var updatedScore int64

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		}))
		defer server.Close()

		sv := &domain.Server{
			ID:        2,
			Namespace: "default",
			Kind:      "Service",
			ObjectID:  "my-api",
			Interval:  30 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{
				Port:          8080,
				EndpointPath:  "/health",
				ExpectedCode:  200,
				BodyCheckExpr: `status == "ok"`,
				Method:        "GET",
			},
		}

		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
					t.Fatal("pod status check should not run for http-dns server")
					return false, nil
				},
			},
			urlResolver: &mockURLResolver{
				resolveFn: func(_ context.Context, params *dto.CheckParams) (*url.URL, error) {
					if params.HTTPCheckParams.Port != 8080 || params.HTTPCheckParams.EndpointPath != "/health" || params.HTTPCheckParams.Method != "GET" {
						t.Errorf("unexpected http params: %+v", params.HTTPCheckParams)
					}
					u, err := url.Parse(server.URL)
					if err != nil {
						t.Fatalf("parse server url: %v", err)
					}
					return u, nil
				},
			},
			pingClient:      infrastructure.NewPingClient(&http.Client{}),
			responseChecker: &ResponseChecker{bodyChecker: &infrastructure.BodyChecker{}},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, nextScore int64) error {
					updatedScore = nextScore
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOn {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOn)
		}
		if updatedScore <= 0 {
			t.Errorf("expected positive updated score, got %d", updatedScore)
		}
	})

	t.Run("http-dns stale pod ip changed invalidates cache and skips event", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		sv := &domain.Server{
			ID:        7,
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "web-app",
			K8s:      &domain.K8sRuntime{Domain: "10.0.0.1"},
			Interval:  30 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{
				Port:         8080,
				EndpointPath: "/health",
				ExpectedCode: 200,
				Method:       "GET",
			},
		}

		var recordedEvent *domain.ServerEvent
		var scoreUpdated bool
		deletedIDs := make([]uint, 0)

		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
					return true, nil
				},
			},
			urlResolver: &mockURLResolver{
				resolveFn: func(_ context.Context, _ *dto.CheckParams) (*url.URL, error) {
					u, err := url.Parse(server.URL)
					if err != nil {
						t.Fatalf("parse server url: %v", err)
					}
					return u, nil
				},
				resolveDomainFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					return "10.0.0.2", nil
				},
			},
			pingClient:      infrastructure.NewPingClient(&http.Client{}),
			responseChecker: &ResponseChecker{bodyChecker: &infrastructure.BodyChecker{}},
			metaCache: &mockDomainCache{
				deleteFn: func(_ context.Context, id uint) error {
					deletedIDs = append(deletedIDs, id)
					return nil
				},
			},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, _ int64) error {
					scoreUpdated = true
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent != nil {
			t.Errorf("expected no event to be recorded, got %+v", recordedEvent)
		}
		if scoreUpdated {
			t.Error("expected score not to be updated")
		}
		if len(deletedIDs) != 1 || deletedIDs[0] != 7 {
			t.Errorf("expected metaCache.Delete(7), got %v", deletedIDs)
		}
	})

	t.Run("http-dns stale pod ip but pod gone records off", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		sv := &domain.Server{
			ID:        8,
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "web-app",
			K8s:      &domain.K8sRuntime{Domain: "10.0.0.1"},
			Interval:  30 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{
				Port:         8080,
				EndpointPath: "/health",
				ExpectedCode: 200,
				Method:       "GET",
			},
		}

		var recordedEvent *domain.ServerEvent
		var resolveDomainCalled bool

		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
					return false, errors.New("pod not found")
				},
			},
			urlResolver: &mockURLResolver{
				resolveFn: func(_ context.Context, _ *dto.CheckParams) (*url.URL, error) {
					u, err := url.Parse(server.URL)
					if err != nil {
						t.Fatalf("parse server url: %v", err)
					}
					return u, nil
				},
				resolveDomainFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					resolveDomainCalled = true
					return "10.0.0.2", nil
				},
			},
			pingClient:      infrastructure.NewPingClient(&http.Client{}),
			responseChecker: &ResponseChecker{bodyChecker: &infrastructure.BodyChecker{}},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
			},
			logger: logger.NewMockLogger(),
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOff {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOff)
		}
		if resolveDomainCalled {
			t.Error("expected no domain re-resolution when pod is gone")
		}
	})

	t.Run("http-dns stale pod ip unchanged records off", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		sv := &domain.Server{
			ID:        9,
			Namespace: "default",
			Kind:      "Pod",
			ObjectID:  "web-app",
			K8s:      &domain.K8sRuntime{Domain: "10.0.0.1"},
			Interval:  30 * time.Second,
			HTTPConfig: &domain.ServerHTTPConfig{
				Port:         8080,
				EndpointPath: "/health",
				ExpectedCode: 200,
				Method:       "GET",
			},
		}

		var recordedEvent *domain.ServerEvent

		s := &PingLoopService{
			pingWorker: &mockPingWorker{
				checkObjectStatusFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (bool, error) {
					return true, nil
				},
			},
			urlResolver: &mockURLResolver{
				resolveFn: func(_ context.Context, _ *dto.CheckParams) (*url.URL, error) {
					u, err := url.Parse(server.URL)
					if err != nil {
						t.Fatalf("parse server url: %v", err)
					}
					return u, nil
				},
				resolveDomainFn: func(_ context.Context, _ *dto.K8sObjectCheckParams) (string, error) {
					return "10.0.0.1", nil
				},
			},
			pingClient:      infrastructure.NewPingClient(&http.Client{}),
			responseChecker: &ResponseChecker{bodyChecker: &infrastructure.BodyChecker{}},
			recordStatusWorker: &mockRecordWorker{
				recordFn: func(_ context.Context, event *domain.ServerEvent) error {
					recordedEvent = event
					return nil
				},
			},
			scoreUpdater: &mockScoreUpdater{
				updateFn: func(_ context.Context, _ uint, _ int64) error { return nil },
			},
			logger: logger.NewMockLogger(),
		}

		s.pingAndRecordServer(t.Context(), PingTask{Server: sv})
		if recordedEvent == nil {
			t.Fatal("expected event to be recorded")
		}
		if recordedEvent.Status != domain.StatusOff {
			t.Errorf("status = %q, want %q", recordedEvent.Status, domain.StatusOff)
		}
	})
}
