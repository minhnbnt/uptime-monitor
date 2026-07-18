package infrastructure

import "testing"

func TestBodyChecker_Check(t *testing.T) {
	checker := &BodyChecker{}

	tests := []struct {
		name       string
		body       string
		expression string
		wantOK     bool
		wantErr    bool
	}{
		{"contains true", `{"status":"ok"}`, `body contains "ok"`, true, false},
		{"contains false", `{"status":"fail"}`, `body contains "ok"`, false, false},
		{"matches true", "hello world", `body matches "world"`, true, false},
		{"matches false", "hello", `body matches "world"`, false, false},
		{"invalid syntax", "anything", `body contains`, false, true},
		{"runtime error", "x", `body + 1`, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, err := checker.Check(tt.body, tt.expression)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got %v", tt.wantOK, ok)
			}
		})
	}
}
