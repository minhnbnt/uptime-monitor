package utils

import (
	"testing"
	"time"
)

func TestParseResolution(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"15 minutes", "15m", 15 * time.Minute, false},
		{"1 hour", "1h", time.Hour, false},
		{"5 minutes", "5m", 5 * time.Minute, false},
		{"30 minutes", "30m", 30 * time.Minute, false},
		{"6 hours", "6h", 6 * time.Hour, false},
		{"too small ms", "500ms", 0, true},
		{"too small s", "30s", 0, true},
		{"invalid string", "abc", 0, true},
		{"empty string", "", 0, true},
		{"negative", "-5m", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseResolution(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseResolution(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseResolution(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitIntervals(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		from       time.Time
		to         time.Time
		resolution time.Duration
		wantLen    int
		wantFirst  Interval
		wantLast   Interval
	}{
		{
			name:       "1h with 15m resolution = 4 intervals",
			from:       base,
			to:         base.Add(time.Hour),
			resolution: 15 * time.Minute,
			wantLen:    4,
			wantFirst:  Interval{base, base.Add(15 * time.Minute)},
			wantLast:   Interval{base.Add(45 * time.Minute), base.Add(time.Hour)},
		},
		{
			name:       "30m with 1h resolution = 1 interval (cutoff)",
			from:       base,
			to:         base.Add(30 * time.Minute),
			resolution: time.Hour,
			wantLen:    1,
			wantFirst:  Interval{base, base.Add(30 * time.Minute)},
			wantLast:   Interval{base, base.Add(30 * time.Minute)},
		},
		{
			name:       "same from/to = 0 intervals",
			from:       base,
			to:         base,
			resolution: 15 * time.Minute,
			wantLen:    0,
		},
		{
			name:       "2h with 1h resolution = 2 intervals",
			from:       base,
			to:         base.Add(2 * time.Hour),
			resolution: time.Hour,
			wantLen:    2,
			wantFirst:  Interval{base, base.Add(time.Hour)},
			wantLast:   Interval{base.Add(time.Hour), base.Add(2 * time.Hour)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitIntervals(tt.from, tt.to, tt.resolution)
			if len(got) != tt.wantLen {
				t.Errorf("SplitIntervals() len = %d, want %d", len(got), tt.wantLen)
				return
			}
			if tt.wantLen == 0 {
				return
			}
			if got[0] != tt.wantFirst {
				t.Errorf("first interval = %v, want %v", got[0], tt.wantFirst)
			}
			if got[len(got)-1] != tt.wantLast {
				t.Errorf("last interval = %v, want %v", got[len(got)-1], tt.wantLast)
			}
		})
	}
}
