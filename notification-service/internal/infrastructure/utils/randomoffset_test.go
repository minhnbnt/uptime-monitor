package utils

import (
	"testing"
	"time"
)

func TestGenerateOffset(t *testing.T) {
	t.Run("deterministic for same id and interval", func(t *testing.T) {
		a := GenerateOffset("server-1", 30*time.Second)
		b := GenerateOffset("server-1", 30*time.Second)
		if a != b {
			t.Errorf("expected same offset, got %v and %v", a, b)
		}
	})

	t.Run("within interval range", func(t *testing.T) {
		interval := 30 * time.Second
		offset := GenerateOffset("server-1", interval)
		if offset < 0 || offset >= interval {
			t.Errorf("offset %v out of range [0, %v)", offset, interval)
		}
	})

	t.Run("different ids produce different offsets", func(t *testing.T) {
		a := GenerateOffset("server-1", 1000*time.Second)
		b := GenerateOffset("server-2", 1000*time.Second)
		if a == b {
			t.Errorf("expected different offsets for different ids, got both %v", a)
		}
	})

	t.Run("works with minute intervals", func(t *testing.T) {
		offset := GenerateOffset("test", 5*time.Minute)
		if offset < 0 || offset >= 5*time.Minute {
			t.Errorf("offset %v out of range", offset)
		}
	})

	t.Run("empty string id", func(t *testing.T) {
		offset := GenerateOffset("", 10*time.Second)
		if offset < 0 || offset >= 10*time.Second {
			t.Errorf("offset %v out of range", offset)
		}
	})
}
