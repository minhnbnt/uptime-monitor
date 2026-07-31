package service

import (
	"testing"

	"github.com/minhnbnt/uptime-monitor-microservices/ping-service/internal/dto"
)

func TestBuildURL(t *testing.T) {
	tests := []struct {
		name       string
		host       string
		httpParams *dto.HTTPCheckParams
		want       string
	}{
		{
			name:       "path with leading slash",
			host:       "my-api.default.svc.cluster.local",
			httpParams: &dto.HTTPCheckParams{Port: 8080, EndpointPath: "/health"},
			want:       "http://my-api.default.svc.cluster.local:8080/health",
		},
		{
			name:       "path without leading slash",
			host:       "my-api.default.svc.cluster.local",
			httpParams: &dto.HTTPCheckParams{Port: 8080, EndpointPath: "health"},
			want:       "http://my-api.default.svc.cluster.local:8080/health",
		},
		{
			name:       "empty path",
			host:       "my-api.default.svc.cluster.local",
			httpParams: &dto.HTTPCheckParams{Port: 8080, EndpointPath: ""},
			want:       "http://my-api.default.svc.cluster.local:8080",
		},
		{
			name:       "pod ip host",
			host:       "10.0.0.5",
			httpParams: &dto.HTTPCheckParams{Port: 80, EndpointPath: "/health"},
			want:       "http://10.0.0.5:80/health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildURL(tt.host, tt.httpParams)
			if got.String() != tt.want {
				t.Errorf("buildURL() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}
