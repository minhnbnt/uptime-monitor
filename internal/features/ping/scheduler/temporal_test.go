package scheduler

import "testing"

func TestToScheduleID(t *testing.T) {
	tests := []struct {
		serverID string
		want     string
	}{
		{"1", "ping-schedule-1"},
		{"42", "ping-schedule-42"},
		{"0", "ping-schedule-0"},
		{"999999", "ping-schedule-999999"},
		{"abc", "ping-schedule-abc"},
		{"", "ping-schedule-"},
	}

	for _, tt := range tests {
		got := toScheduleID(tt.serverID)
		if got != tt.want {
			t.Errorf("toScheduleID(%q) = %q, want %q", tt.serverID, got, tt.want)
		}
	}
}
