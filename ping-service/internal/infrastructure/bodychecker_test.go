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
		{"contains true", `{"status":"ok"}`, `status matches "ok"`, true, false},
		{"contains false", `{"status":"fail"}`, `status matches "ok"`, false, false},
		{"matches true", `{"value":"hello world"}`, `value matches "hello.*"`, true, false},
		{"matches false", `{"value":"hello"}`, `value matches "world"`, false, false},
		{"invalid json", "anything", `true`, false, true},
		{"invalid syntax", `{"x":"y"}`, `invalid[`, false, true},
		{"runtime error", `{"x":"y"}`, `nonexistent(x)`, false, true},
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
