package utils

import (
	"strings"
	"testing"
)

func TestValidateServerName(t *testing.T) {
	t.Run("valid name", func(t *testing.T) {
		if err := ValidateServerName("My Server"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("trims spaces", func(t *testing.T) {
		if err := ValidateServerName("  My Server  "); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty name", func(t *testing.T) {
		if err := ValidateServerName(""); err != ErrNameRequired {
			t.Errorf("got %v, want %v", err, ErrNameRequired)
		}
	})

	t.Run("whitespace only", func(t *testing.T) {
		if err := ValidateServerName("   "); err != ErrNameRequired {
			t.Errorf("got %v, want %v", err, ErrNameRequired)
		}
	})

	t.Run("too long", func(t *testing.T) {
		long := strings.Repeat("a", 256)
		if err := ValidateServerName(long); err != ErrNameTooLong {
			t.Errorf("got %v, want %v", err, ErrNameTooLong)
		}
	})

	t.Run("exactly 255 chars", func(t *testing.T) {
		name := strings.Repeat("a", 255)
		if err := ValidateServerName(name); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestValidateURL(t *testing.T) {
	t.Run("valid http", func(t *testing.T) {
		if err := ValidateURL("http://example.com"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid https", func(t *testing.T) {
		if err := ValidateURL("https://example.com/path"); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("trims spaces", func(t *testing.T) {
		if err := ValidateURL("  https://example.com  "); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty is valid", func(t *testing.T) {
		if err := ValidateURL(""); err != nil {
			t.Errorf("expected nil for empty, got %v", err)
		}
	})
	t.Run("invalid scheme", func(t *testing.T) {
		if err := ValidateURL("ftp://example.com"); err != ErrURLInvalid {
			t.Errorf("got %v, want %v", err, ErrURLInvalid)
		}
	})

	t.Run("no scheme", func(t *testing.T) {
		if err := ValidateURL("example.com"); err != ErrURLInvalid {
			t.Errorf("got %v, want %v", err, ErrURLInvalid)
		}
	})

	t.Run("invalid url string", func(t *testing.T) {
		if err := ValidateURL("http://\x00"); err != ErrURLParse {
			t.Errorf("got %v, want %v", err, ErrURLParse)
		}
	})
}

func TestValidateMethod(t *testing.T) {
	t.Run("valid method", func(t *testing.T) {
		got, err := ValidateMethod("POST")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "POST" {
			t.Errorf("got %q, want %q", got, "POST")
		}
	})

	t.Run("case insensitive", func(t *testing.T) {
		got, err := ValidateMethod("get")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "GET" {
			t.Errorf("got %q, want %q", got, "GET")
		}
	})

	t.Run("empty defaults to GET", func(t *testing.T) {
		got, err := ValidateMethod("")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "GET" {
			t.Errorf("got %q, want %q", got, "GET")
		}
	})

	t.Run("unsupported method", func(t *testing.T) {
		_, err := ValidateMethod("INVALID")
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestValidateInterval(t *testing.T) {
	t.Run("valid interval", func(t *testing.T) {
		if err := ValidateInterval(30); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("zero interval", func(t *testing.T) {
		if err := ValidateInterval(0); err != ErrIntervalInvalid {
			t.Errorf("got %v, want %v", err, ErrIntervalInvalid)
		}
	})

	t.Run("negative", func(t *testing.T) {
		if err := ValidateInterval(-1); err != ErrIntervalInvalid {
			t.Errorf("got %v, want %v", err, ErrIntervalInvalid)
		}
	})
}

func TestValidateTimeout(t *testing.T) {
	t.Run("valid timeout", func(t *testing.T) {
		if err := ValidateTimeout(10); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("zero timeout", func(t *testing.T) {
		if err := ValidateTimeout(0); err != ErrTimeoutInvalid {
			t.Errorf("got %v, want %v", err, ErrTimeoutInvalid)
		}
	})

	t.Run("negative", func(t *testing.T) {
		if err := ValidateTimeout(-5); err != ErrTimeoutInvalid {
			t.Errorf("got %v, want %v", err, ErrTimeoutInvalid)
		}
	})
}

func TestValidateExpectedCode(t *testing.T) {
	t.Run("valid 200", func(t *testing.T) {
		if err := ValidateExpectedCode(200); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid 100", func(t *testing.T) {
		if err := ValidateExpectedCode(100); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("valid 599", func(t *testing.T) {
		if err := ValidateExpectedCode(599); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("below range", func(t *testing.T) {
		if err := ValidateExpectedCode(99); err != ErrCodeOutOfRange {
			t.Errorf("got %v, want %v", err, ErrCodeOutOfRange)
		}
	})

	t.Run("above range", func(t *testing.T) {
		if err := ValidateExpectedCode(600); err != ErrCodeOutOfRange {
			t.Errorf("got %v, want %v", err, ErrCodeOutOfRange)
		}
	})

	t.Run("zero", func(t *testing.T) {
		if err := ValidateExpectedCode(0); err != ErrCodeOutOfRange {
			t.Errorf("got %v, want %v", err, ErrCodeOutOfRange)
		}
	})
}
