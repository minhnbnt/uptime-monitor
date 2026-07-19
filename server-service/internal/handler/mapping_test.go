package handler

import (
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/server-service/generated/api"
	"github.com/minhnbnt/uptime-monitor-microservices/server-service/internal/dto"
)

func TestToAPIServer_NoMonitorStatusValidates(t *testing.T) {
	out := ToAPIServer(&dto.Server{Name: "x"})
	if !out.MonitorStatus.IsEmpty() {
		t.Fatalf("expected empty monitor_status, got %q (set=%v null=%v)", out.MonitorStatus.Value, out.MonitorStatus.Set, out.MonitorStatus.Null)
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("response failed validation: %v", err)
	}
}

func TestToAPIServer_WithMonitorStatusValidates(t *testing.T) {
	out := ToAPIServer(&dto.Server{Name: "x", MonitorStatus: "ON"})
	if !out.MonitorStatus.IsSet() || out.MonitorStatus.Value != api.ServerObjectMonitorStatusON {
		t.Fatalf("expected monitor_status ON, got %q", out.MonitorStatus.Value)
	}
	if err := out.Validate(); err != nil {
		t.Fatalf("response failed validation: %v", err)
	}
}
