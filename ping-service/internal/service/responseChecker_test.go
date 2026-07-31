package service

import (
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
	infra "github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/infrastructure"
)

func TestResponseChecker_CheckResponse(t *testing.T) {
	checker := &ResponseChecker{bodyChecker: &infra.BodyChecker{}}

	tests := []struct {
		name    string
		http    *dto.HTTPCheckParams
		resp    infra.Response
		wantErr bool
	}{
		{
			name: "default no check",
			http: &dto.HTTPCheckParams{},
			resp: infra.Response{StatusCode: 500, Body: "anything"},
		},
		{
			name:    "status mismatch",
			http:    &dto.HTTPCheckParams{ExpectedCode: 200},
			resp:    infra.Response{StatusCode: 500, Body: ""},
			wantErr: true,
		},
		{
			name: "status ok no expr",
			http: &dto.HTTPCheckParams{ExpectedCode: 200},
			resp: infra.Response{StatusCode: 200, Body: "anything"},
		},
		{
			name: "status ok expr true",
			http: &dto.HTTPCheckParams{ExpectedCode: 200, BodyCheckExpr: `status == "ok"`},
			resp: infra.Response{StatusCode: 200, Body: `{"status":"ok"}`},
		},
		{
			name:    "status ok expr false",
			http:    &dto.HTTPCheckParams{ExpectedCode: 200, BodyCheckExpr: `status == "ok"`},
			resp:    infra.Response{StatusCode: 200, Body: `{"status":"fail"}`},
			wantErr: true,
		},
		{
			name:    "expr error is fail-safe",
			http:    &dto.HTTPCheckParams{ExpectedCode: 200, BodyCheckExpr: `status =`},
			resp:    infra.Response{StatusCode: 200, Body: "x"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.CheckResponse(tt.http, tt.resp)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
