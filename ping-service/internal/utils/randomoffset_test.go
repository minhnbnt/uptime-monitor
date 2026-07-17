package utils

import (
	"testing"
	"time"
)

func TestGenerateOffset(t *testing.T) {
	t.Run("returns value in [0, interval)", func(t *testing.T) {
		tests := []struct {
			id       string
			interval time.Duration
		}{
			{"", 10 * time.Second},
			{"1", 30 * time.Second},
			{"999", 60 * time.Second},
			{"", 1<<63 - 1},
		}
		for _, tt := range tests {
			got := GenerateOffset(tt.id, tt.interval)
			if got < 0 || got >= tt.interval {
				t.Errorf("GenerateOffset(%q, %v) = %v, want [0, %v)", tt.id, tt.interval, got, tt.interval)
			}
		}
	})

	t.Run("deterministic", func(t *testing.T) {
		a := GenerateOffset("42", 10*time.Second)
		b := GenerateOffset("42", 10*time.Second)
		if a != b {
			t.Errorf("expected same offset for same input, got %v != %v", a, b)
		}
	})

	t.Run("zero interval panics", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("expected panic for zero interval")
			}
		}()
		GenerateOffset("1", 0)
	})
}
