package consumer

import "testing"

func TestRedisOffsetStoreIsNewer(t *testing.T) {
	s := &RedisOffsetStore{}

	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{"greater ms", "2-0", "1-0", true},
		{"equal ms greater seq", "1-2", "1-1", true},
		{"equal", "1-1", "1-1", false},
		{"less", "1-0", "2-0", false},
		{"zero base", "1-0", "0-0", true},
	}

	for _, c := range cases {
		got, err := s.IsNewer(c.a, c.b)
		if err != nil {
			t.Fatalf("%s: IsNewer(%q,%q): %v", c.name, c.a, c.b, err)
		}
		if got != c.want {
			t.Errorf("%s: IsNewer(%q,%q)=%v, want %v", c.name, c.a, c.b, got, c.want)
		}
	}
}

func TestParseOffsetInvalid(t *testing.T) {
	if _, _, err := parseOffset("not-an-id"); err == nil {
		t.Error("expected error for invalid offset")
	}
}
