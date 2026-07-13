package scheduler

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
)

func TestMetaCacheKey(t *testing.T) {
	tests := []struct {
		id   uint
		want string
	}{
		{0, "scheduler:meta:0"},
		{1, "scheduler:meta:1"},
		{42, "scheduler:meta:42"},
		{999999, "scheduler:meta:999999"},
	}

	for _, tt := range tests {
		got := metaCacheKey(tt.id)
		if got != tt.want {
			t.Errorf("metaCacheKey(%d) = %q, want %q", tt.id, got, tt.want)
		}
	}
}

func TestMapToEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		id      uint
		data    map[string]string
		want    *domain.Endpoint
		wantErr bool
	}{
		{
			name: "all fields valid",
			id:   42,
			data: map[string]string{
				"url":           "https://example.com",
				"method":        "GET",
				"expected_code": "200",
				"interval_ns":   "30000000000",
			},
			want: &domain.Endpoint{
				Model:        gorm.Model{ID: 42},
				URL:          "https://example.com",
				Method:       "GET",
				ExpectedCode: 200,
				Interval:     30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "POST method with non-default expected code",
			id:   7,
			data: map[string]string{
				"url":           "https://api.example.com/create",
				"method":        "POST",
				"expected_code": "201",
				"interval_ns":   "60000000000",
			},
			want: &domain.Endpoint{
				Model:        gorm.Model{ID: 7},
				URL:          "https://api.example.com/create",
				Method:       "POST",
				ExpectedCode: 201,
				Interval:     60 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "zero id and interval",
			id:   0,
			data: map[string]string{
				"url":           "https://test.com",
				"method":        "GET",
				"expected_code": "200",
				"interval_ns":   "0",
			},
			want: &domain.Endpoint{
				Model:        gorm.Model{ID: 0},
				URL:          "https://test.com",
				Method:       "GET",
				ExpectedCode: 200,
				Interval:     0,
			},
			wantErr: false,
		},
		{
			name: "interval_ns not a number",
			id:   1,
			data: map[string]string{
				"url":           "https://example.com",
				"method":        "GET",
				"expected_code": "200",
				"interval_ns":   "notanumber",
			},
			wantErr: true,
		},
		{
			name: "expected_code not a number",
			id:   1,
			data: map[string]string{
				"url":           "https://example.com",
				"method":        "GET",
				"expected_code": "twelve",
				"interval_ns":   "30000000000",
			},
			wantErr: true,
		},
		{
			name: "missing interval_ns",
			id:   1,
			data: map[string]string{
				"url":           "https://example.com",
				"method":        "GET",
				"expected_code": "200",
			},
			wantErr: true,
		},
		{
			name: "missing expected_code",
			id:   1,
			data: map[string]string{
				"url":         "https://example.com",
				"method":      "GET",
				"interval_ns": "30000000000",
			},
			wantErr: true,
		},
		{
			name: "missing url and method",
			id:   1,
			data: map[string]string{
				"expected_code": "200",
				"interval_ns":   "30000000000",
			},
			want: &domain.Endpoint{
				Model:        gorm.Model{ID: 1},
				URL:          "",
				Method:       "",
				ExpectedCode: 200,
				Interval:     30 * time.Second,
			},
			wantErr: false,
		},
		{
			name:    "empty data map",
			id:      1,
			data:    map[string]string{},
			wantErr: true,
		},
		{
			name: "extra keys in data are ignored",
			id:   5,
			data: map[string]string{
				"url":           "https://example.com",
				"method":        "GET",
				"expected_code": "200",
				"interval_ns":   "30000000000",
				"extra_field":   "ignored",
			},
			want: &domain.Endpoint{
				Model:        gorm.Model{ID: 5},
				URL:          "https://example.com",
				Method:       "GET",
				ExpectedCode: 200,
				Interval:     30 * time.Second,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mapToEndpoint(tt.id, tt.data)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.want.ID {
				t.Errorf("ID = %d, want %d", got.ID, tt.want.ID)
			}
			if got.URL != tt.want.URL {
				t.Errorf("URL = %q, want %q", got.URL, tt.want.URL)
			}
			if got.Method != tt.want.Method {
				t.Errorf("Method = %q, want %q", got.Method, tt.want.Method)
			}
			if got.ExpectedCode != tt.want.ExpectedCode {
				t.Errorf("ExpectedCode = %d, want %d", got.ExpectedCode, tt.want.ExpectedCode)
			}
			if got.Interval != tt.want.Interval {
				t.Errorf("Interval = %v, want %v", got.Interval, tt.want.Interval)
			}
		})
	}
}
