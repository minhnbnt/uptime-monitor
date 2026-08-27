package service

import (
	"context"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
)

type captureEventRecorder struct {
	saved *domain.ServerEvent
}

func (c *captureEventRecorder) Save(_ context.Context, event *domain.ServerEvent) error {
	c.saved = event
	return nil
}

var _ EventRecorder = (*captureEventRecorder)(nil)

func TestRecordEventUsesProvidedTime(t *testing.T) {
	want := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	rec := &captureEventRecorder{}
	svc := NewEventService(rec, nil, &fakeOwnerRepo{})

	err := svc.RecordEvent(context.Background(), dto.RecordEventRequest{
		EndpointID: 7,
		Status:     dto.ServerStatusOff,
		Time:       want,
	})
	if err != nil {
		t.Fatalf("RecordEvent: %v", err)
	}
	if rec.saved == nil {
		t.Fatal("expected an event to be saved")
	}
	if !rec.saved.Time.Equal(want) {
		t.Errorf("saved time = %v, want %v", rec.saved.Time, want)
	}
	if rec.saved.Status != domain.StatusOff {
		t.Errorf("saved status = %q, want %q", rec.saved.Status, domain.StatusOff)
	}
}
