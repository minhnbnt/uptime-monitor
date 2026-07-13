package repository

import "testing"

func TestStatusKeyConstant(t *testing.T) {
	want := "endpoint:status"
	if statusKey != want {
		t.Errorf("statusKey = %q, want %q", statusKey, want)
	}
}
