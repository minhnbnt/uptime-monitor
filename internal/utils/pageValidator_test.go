package utils

import "testing"

func TestPageValidator_Validate(t *testing.T) {
	v := NewPageValidator(30)

	tests := []struct {
		name       string
		pageNumber int
		pageSize   int
		wantErr    bool
	}{
		{"valid", 1, 10, false},
		{"valid edge min", 1, 29, false},
		{"page 0", 0, 10, true},
		{"page negative", -1, 10, true},
		{"size equals max", 1, 30, false},
		{"size exceeds max", 1, 100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.pageNumber, tt.pageSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate(%d, %d) = %v, wantErr=%v",
					tt.pageNumber, tt.pageSize, err, tt.wantErr)
			}
		})
	}
}
