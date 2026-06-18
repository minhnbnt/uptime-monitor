package repository

import "testing"

func TestStatusKey(t *testing.T) {
	tests := []struct {
		endpointID uint
		want       string
	}{
		{0, "endpoint:0:status"},
		{1, "endpoint:1:status"},
		{42, "endpoint:42:status"},
		{999, "endpoint:999:status"},
	}

	for _, tt := range tests {
		got := statusKey(tt.endpointID)
		if got != tt.want {
			t.Errorf("statusKey(%d) = %q, want %q", tt.endpointID, got, tt.want)
		}
	}
}
