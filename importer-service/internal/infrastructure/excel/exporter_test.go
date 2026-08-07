package excel

import (
	"testing"
	"time"

	serverdto "github.com/minhnbnt/uptime-monitor-microservices/importer-service/internal/dto"
)

func TestToRowMap(t *testing.T) {

	sv := serverdto.Server{
		Name:       "web",
		Namespace:  "default",
		Kind:       "Deployment",
		ObjectID:   "web",
		Interval:   15 * time.Second,
		Timeout:    7 * time.Second,
		HTTPConfig: &serverdto.HTTPConfig{Port: 8080, EndpointPath: "/healthz", ExpectedCode: 200, Method: "GET"},
	}

	got := toRowMap(2, sv)

	if got[cellRef("server_name", 2)] != "web" {
		t.Errorf("server_name cell = %q", got[cellRef("server_name", 2)])
	}
	if got[cellRef("interval_sec", 2)] != "15" {
		t.Errorf("interval cell = %q, want 15", got[cellRef("interval_sec", 2)])
	}
	if got[cellRef("timeout_sec", 2)] != "7" {
		t.Errorf("timeout cell = %q, want 7", got[cellRef("timeout_sec", 2)])
	}
	if got[cellRef("http_port", 2)] != "8080" {
		t.Errorf("http_port cell = %q, want 8080", got[cellRef("http_port", 2)])
	}
	if got[cellRef("http_path", 2)] != "/healthz" {
		t.Errorf("http_path cell = %q, want /healthz", got[cellRef("http_path", 2)])
	}

	if sv := (serverdto.Server{Name: "no-http"}); toRowMap(3, sv)[cellRef("http_port", 3)] != "" {
		t.Error("server without http config must not write http cells")
	}

	zero := serverdto.Server{Name: "zero", Interval: 0, Timeout: 0}
	if got := toRowMap(4, zero); got[cellRef("interval_sec", 4)] != "" || got[cellRef("timeout_sec", 4)] != "" {
		t.Error("zero interval/timeout must be skipped, got:", got)
	}
}
