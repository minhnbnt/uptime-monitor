package service

import (
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/domain"
	infra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
)

func strptr(s string) *string { return &s }

func TestResponseChecker_CheckResponse(t *testing.T) {
	checker := &ResponseChecker{bodyChecker: &infra.BodyChecker{}}

	tests := []struct {
		name    string
		ep      domain.Endpoint
		resp    infra.Response
		wantErr bool
	}{
		{
			name:    "status mismatch",
			ep:      domain.Endpoint{ExpectedCode: 200},
			resp:    infra.Response{StatusCode: 500, Body: ""},
			wantErr: true,
		},
		{
			name:    "status ok no expr",
			ep:      domain.Endpoint{ExpectedCode: 200},
			resp:    infra.Response{StatusCode: 200, Body: "anything"},
			wantErr: false,
		},
		{
			name:    "status ok expr true",
			ep:      domain.Endpoint{ExpectedCode: 200, BodyCheckExpr: strptr(`status == "ok"`)},
			resp:    infra.Response{StatusCode: 200, Body: `{"status":"ok"}`},
			wantErr: false,
		},
		{
			name:    "status ok expr false",
			ep:      domain.Endpoint{ExpectedCode: 200, BodyCheckExpr: strptr(`status == "ok"`)},
			resp:    infra.Response{StatusCode: 200, Body: `{"status":"fail"}`},
			wantErr: true,
		},
		{
			name:    "expr error is fail-safe",
			ep:      domain.Endpoint{ExpectedCode: 200, BodyCheckExpr: strptr(`status =`)},
			resp:    infra.Response{StatusCode: 200, Body: "x"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.CheckResponse(tt.ep, tt.resp)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
