package service

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/logger"
)

type fakeResolver struct {
	owned  map[uint64]bool
	err    error
	called bool
}

func (f *fakeResolver) ResolveServers(_ context.Context, _ uint, ids []uint64) ([]uint64, error) {
	f.called = true
	if f.err != nil {
		return nil, f.err
	}
	resolved := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if f.owned[id] {
			resolved = append(resolved, id)
		}
	}
	return resolved, nil
}

type fakeGate struct {
	allowCalled bool
	released    bool
	allowed     bool
	next        time.Time
	err         error
}

func (f *fakeGate) Allow(_ context.Context, _ string, _ time.Duration) (time.Time, bool, error) {
	f.allowCalled = true
	if f.err != nil {
		return time.Time{}, false, f.err
	}
	return f.next, f.allowed, nil
}

func (f *fakeGate) Release(_ context.Context, _ string) error {
	f.released = true
	return nil
}

type fakeRecorder struct {
	events []*domain.ServerEvent
	fail   map[uint]bool
}

func (f *fakeRecorder) Record(_ context.Context, event *domain.ServerEvent) error {
	if f.fail[event.EndpointID] {
		return errors.New("boom")
	}
	f.events = append(f.events, event)
	return nil
}

var testNext = time.UnixMilli(1756000000000)

func newTestPushService(r *fakeResolver, g *fakeGate, rec *fakeRecorder) *PushEventService {
	return &PushEventService{
		resolver: r,
		gate:     g,
		recorder: rec,
		logger:   logger.NewMockLogger(),
	}
}

func TestPushEventHandle(t *testing.T) {

	user := uint(7)
	sid := "sess-1"

	t.Run("all valid returns accepted without errors", func(t *testing.T) {

		r := &fakeResolver{owned: map[uint64]bool{1: true, 2: true}}
		g := &fakeGate{allowed: true, next: testNext}
		rec := &fakeRecorder{}
		svc := newTestPushService(r, g, rec)

		res, err := svc.Handle(t.Context(), user, sid, []PushEventItem{
			{ID: 1, Status: "ON"},
			{ID: 2, Status: "OFF"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !slices.Equal(res.Accepted, []uint64{1, 2}) {
			t.Errorf("accepted = %v, want [1 2]", res.Accepted)
		}
		if len(res.Errors) != 0 {
			t.Errorf("errors = %v, want empty", res.Errors)
		}
		if !res.NextTime.Equal(testNext) {
			t.Errorf("nextTime = %v, want %v", res.NextTime, testNext)
		}
		if g.released {
			t.Error("gate must not be released")
		}
		if len(rec.events) != 2 {
			t.Errorf("recorded = %d events, want 2", len(rec.events))
		}
	})

	t.Run("unknown id goes to errors while rest accepted", func(t *testing.T) {

		r := &fakeResolver{owned: map[uint64]bool{1: true}}
		g := &fakeGate{allowed: true, next: testNext}
		rec := &fakeRecorder{}
		svc := newTestPushService(r, g, rec)

		res, err := svc.Handle(t.Context(), user, sid, []PushEventItem{
			{ID: 1, Status: "ON"},
			{ID: 99, Status: "OFF"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !slices.Equal(res.Accepted, []uint64{1}) {
			t.Errorf("accepted = %v, want [1]", res.Accepted)
		}
		if len(res.Errors) != 1 || res.Errors[0].ID != 99 || res.Errors[0].Error != "not found" {
			t.Errorf("errors = %v, want [{99 not found}]", res.Errors)
		}
	})

	t.Run("invalid status never reaches resolver", func(t *testing.T) {

		r := &fakeResolver{owned: map[uint64]bool{1: true}}
		g := &fakeGate{allowed: true, next: testNext}
		rec := &fakeRecorder{}
		svc := newTestPushService(r, g, rec)

		res, err := svc.Handle(t.Context(), user, sid, []PushEventItem{
			{ID: 1, Status: "MAYBE"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if r.called {
			t.Error("resolver must not be called for invalid statuses")
		}
		if g.allowCalled {
			t.Error("gate must not be consulted when nothing is valid")
		}
		if len(res.Errors) != 1 || res.Errors[0].Error != "invalid status" {
			t.Errorf("errors = %v, want invalid status entry", res.Errors)
		}
		if len(res.Accepted) != 0 {
			t.Errorf("accepted = %v, want empty", res.Accepted)
		}
	})

	t.Run("blocked gate returns RateLimitedError with next time", func(t *testing.T) {

		r := &fakeResolver{owned: map[uint64]bool{1: true}}
		g := &fakeGate{allowed: false, next: testNext}
		rec := &fakeRecorder{}
		svc := newTestPushService(r, g, rec)

		res, err := svc.Handle(t.Context(), user, sid, []PushEventItem{{ID: 1, Status: "ON"}})
		var rlErr *RateLimitedError
		if !errors.As(err, &rlErr) {
			t.Fatalf("err = %v, want RateLimitedError", err)
		}
		if !rlErr.NextTime.Equal(testNext) {
			t.Errorf("nextTime = %v, want %v", rlErr.NextTime, testNext)
		}
		if res != nil {
			t.Errorf("result = %v, want nil on rate limit", res)
		}
		if len(rec.events) != 0 {
			t.Errorf("recorded = %d events, want 0", len(rec.events))
		}
	})

	t.Run("all record failures release the milestone", func(t *testing.T) {

		r := &fakeResolver{owned: map[uint64]bool{1: true}}
		g := &fakeGate{allowed: true, next: testNext}
		rec := &fakeRecorder{fail: map[uint]bool{1: true}}
		svc := newTestPushService(r, g, rec)

		res, err := svc.Handle(t.Context(), user, sid, []PushEventItem{{ID: 1, Status: "ON"}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !g.released {
			t.Error("milestone must be released when nothing was accepted")
		}
		if !res.NextTime.IsZero() {
			t.Errorf("nextTime = %v, want zero after release", res.NextTime)
		}
		if len(res.Errors) != 1 || res.Errors[0].ID != 1 {
			t.Errorf("errors = %v, want record failure for id 1", res.Errors)
		}
	})

	t.Run("partial record failures keep the milestone", func(t *testing.T) {

		r := &fakeResolver{owned: map[uint64]bool{1: true, 2: true}}
		g := &fakeGate{allowed: true, next: testNext}
		rec := &fakeRecorder{fail: map[uint]bool{2: true}}
		svc := newTestPushService(r, g, rec)

		res, err := svc.Handle(t.Context(), user, sid, []PushEventItem{
			{ID: 1, Status: "ON"},
			{ID: 2, Status: "OFF"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if g.released {
			t.Error("milestone must be kept when at least one event was accepted")
		}
		if !res.NextTime.Equal(testNext) {
			t.Errorf("nextTime = %v, want %v", res.NextTime, testNext)
		}
		if !slices.Equal(res.Accepted, []uint64{1}) {
			t.Errorf("accepted = %v, want [1]", res.Accepted)
		}
	})
}
