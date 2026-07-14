package ontime

import (
	"testing"
	"time"

	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/features/ontime/repository"
)

func e(day, t time.Time, status string) ontimerepo.RawEvent {
	return ontimerepo.RawEvent{Day: day, Time: t, Status: status}
}

func day(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func tm(y, m, d, h, min int) time.Time {
	return time.Date(y, time.Month(m), d, h, min, 0, 0, time.UTC)
}

func TestCalculateDayOntime(t *testing.T) {
	d := day(2026, 6, 4)
	tomorrow := d.Add(24 * time.Hour)

	tests := []struct {
		name   string
		events []ontimerepo.RawEvent
		today  time.Time
		now    time.Time
		want   float64
	}{
		{
			name:   "no events",
			events: nil,
			want:   0,
		},
		{
			name:   "single ON in past",
			events: []ontimerepo.RawEvent{e(d, tm(2026, 6, 4, 6, 0), "ON")},
			today:  tomorrow,
			now:    tomorrow.Add(1 * time.Hour),
			want:   100,
		},
		{
			name:   "single OFF in past",
			events: []ontimerepo.RawEvent{e(d, tm(2026, 6, 4, 6, 0), "OFF")},
			today:  tomorrow,
			now:    tomorrow.Add(1 * time.Hour),
			want:   0,
		},
		{
			name: "alternating ON/OFF full day",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 4, 6, 0), "ON"),
				e(d, tm(2026, 6, 4, 12, 0), "OFF"),
				e(d, tm(2026, 6, 4, 18, 0), "ON"),
			},
			today: tomorrow,
			now:   tomorrow.Add(1 * time.Hour),
			want:  100 * 18.0 / 24.0,
		},
		{
			name: "today still ON from earlier event",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 4, 3, 0), "ON"),
			},
			today: d,
			now:   tm(2026, 6, 4, 12, 0),
			want:  100,
		},
		{
			name: "today ON then OFF",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 4, 3, 0), "ON"),
				e(d, tm(2026, 6, 4, 9, 0), "OFF"),
			},
			today: d,
			now:   tm(2026, 6, 4, 12, 0),
			want:  100 * 6.0 / 9.0,
		},
		{
			name: "all OFF",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 4, 6, 0), "OFF"),
				e(d, tm(2026, 6, 4, 12, 0), "OFF"),
				e(d, tm(2026, 6, 4, 18, 0), "OFF"),
			},
			today: tomorrow,
			now:   tomorrow.Add(1 * time.Hour),
			want:  0,
		},
		{
			name: "all ON",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 4, 6, 0), "ON"),
				e(d, tm(2026, 6, 4, 12, 0), "ON"),
				e(d, tm(2026, 6, 4, 18, 0), "ON"),
			},
			today: tomorrow,
			now:   tomorrow.Add(1 * time.Hour),
			want:  100,
		},
		{
			name: "with lowerbound ON, event OFF at 10",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 3, 23, 0), "ON"),
				e(d, tm(2026, 6, 4, 10, 0), "OFF"),
			},
			today: tomorrow,
			now:   tomorrow.Add(1 * time.Hour),
			want:  100 * 10.0 / 24.0,
		},
		{
			name: "with lowerbound OFF, event ON at 8 OFF at 16",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 3, 23, 0), "OFF"),
				e(d, tm(2026, 6, 4, 8, 0), "ON"),
				e(d, tm(2026, 6, 4, 16, 0), "OFF"),
			},
			today: tomorrow,
			now:   tomorrow.Add(1 * time.Hour),
			want:  100 * 8.0 / 24.0,
		},
		{
			name: "today ON since lowerbound, then OFF",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 3, 23, 0), "ON"),
				e(d, tm(2026, 6, 4, 10, 0), "OFF"),
			},
			today: d,
			now:   tm(2026, 6, 4, 12, 0),
			want:  100 * 11.0 / 13.0,
		},
		{
			name: "dedup adjacent same-time events",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 4, 6, 0), "ON"),
				e(d, tm(2026, 6, 4, 6, 0), "ON"),
				e(d, tm(2026, 6, 4, 12, 0), "OFF"),
			},
			today: tomorrow,
			now:   tomorrow.Add(1 * time.Hour),
			want:  50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OntimeCalculator{}.CalculateDayOntime(tt.events, tt.today, tt.now)
			diff := got - tt.want
			if diff < 0 {
				diff = -diff
			}
			if diff > 1e-9 {
				t.Errorf("CalculateDayOntime = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCalculateOnlineDuration(t *testing.T) {
	d := day(2026, 6, 4)

	tests := []struct {
		name     string
		timeline Timeline
		want     float64
	}{
		{
			name: "start ON, no events",
			timeline: Timeline{
				StartTime:   d,
				EndTime:     d.Add(24 * time.Hour),
				StartStatus: "ON",
			},
			want: 86400,
		},
		{
			name: "start OFF, no events",
			timeline: Timeline{
				StartTime:   d,
				EndTime:     d.Add(24 * time.Hour),
				StartStatus: "OFF",
			},
			want: 0,
		},
		{
			name: "start ON, event OFF at noon",
			timeline: Timeline{
				StartTime:   d,
				EndTime:     d.Add(24 * time.Hour),
				StartStatus: "ON",
				Events:      []ontimerepo.RawEvent{{Time: tm(2026, 6, 4, 12, 0), Status: "OFF"}},
			},
			want: 43200,
		},
		{
			name: "multi segment: ON→OFF→ON→OFF",
			timeline: Timeline{
				StartTime:   d,
				EndTime:     d.Add(24 * time.Hour),
				StartStatus: "ON",
				Events: []ontimerepo.RawEvent{
					{Time: tm(2026, 6, 4, 8, 0), Status: "OFF"},
					{Time: tm(2026, 6, 4, 16, 0), Status: "ON"},
					{Time: tm(2026, 6, 4, 20, 0), Status: "OFF"},
				},
			},
			want: 8*3600 + 4*3600,
		},
		{
			name: "start OFF, event ON then OFF",
			timeline: Timeline{
				StartTime:   d,
				EndTime:     d.Add(24 * time.Hour),
				StartStatus: "OFF",
				Events: []ontimerepo.RawEvent{
					{Time: tm(2026, 6, 4, 10, 0), Status: "ON"},
					{Time: tm(2026, 6, 4, 18, 0), Status: "OFF"},
				},
			},
			want: 8 * 3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OntimeCalculator{}.CalculateOnlineDuration(tt.timeline)
			if got != tt.want {
				t.Errorf("CalculateOnlineDuration = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildTimelinePastDay(t *testing.T) {
	d := day(2026, 6, 4)
	events := []ontimerepo.RawEvent{
		e(d, tm(2026, 6, 4, 6, 0), "ON"),
		e(d, tm(2026, 6, 4, 12, 0), "OFF"),
	}

	tl := OntimeCalculator{}.BuildTimeline(events, d.Add(24*time.Hour), d.Add(48*time.Hour))

	if !tl.Day.Equal(d) {
		t.Errorf("Day = %v, want %v", tl.Day, d)
	}
	if !tl.StartTime.Equal(d) {
		t.Errorf("StartTime = %v, want %v", tl.StartTime, d)
	}
	if !tl.EndTime.Equal(d.Add(24 * time.Hour)) {
		t.Errorf("EndTime = %v, want %v", tl.EndTime, d.Add(24*time.Hour))
	}
	if tl.StartStatus != "ON" {
		t.Errorf("StartStatus = %v, want ON", tl.StartStatus)
	}
	if len(tl.Events) != 2 {
		t.Errorf("len(Events) = %d, want 2", len(tl.Events))
	}
}

func TestBuildTimelineToday(t *testing.T) {
	d := day(2026, 6, 4)
	now := tm(2026, 6, 4, 14, 0)
	events := []ontimerepo.RawEvent{
		e(d, tm(2026, 6, 4, 6, 0), "ON"),
		e(d, tm(2026, 6, 4, 12, 0), "OFF"),
	}

	tl := OntimeCalculator{}.BuildTimeline(events, d, now)

	if !tl.StartTime.Equal(tm(2026, 6, 4, 6, 0)) {
		t.Errorf("StartTime = %v, want 06:00", tl.StartTime)
	}
	if !tl.EndTime.Equal(now) {
		t.Errorf("EndTime = %v, want %v", tl.EndTime, now)
	}
}

func TestBuildTimelineTodayWithPrevEvents(t *testing.T) {
	prev := day(2026, 6, 3)
	d := day(2026, 6, 4)
	now := tm(2026, 6, 4, 14, 0)
	events := []ontimerepo.RawEvent{
		e(prev, tm(2026, 6, 3, 23, 0), "ON"),
		e(d, tm(2026, 6, 4, 10, 0), "OFF"),
	}

	tl := OntimeCalculator{}.BuildTimeline(events, d, now)

	if !tl.StartTime.Equal(prev) {
		t.Errorf("StartTime = %v, want %v", tl.StartTime, prev)
	}
	if tl.StartStatus != "ON" {
		t.Errorf("StartStatus = %v, want ON", tl.StartStatus)
	}
}

func TestSplitByDayBoundary(t *testing.T) {
	d := day(2026, 6, 4)

	tests := []struct {
		name       string
		events     []ontimerepo.RawEvent
		wantPrev   int
		wantInside int
	}{
		{
			name: "all inside",
			events: []ontimerepo.RawEvent{
				e(d, tm(2026, 6, 4, 6, 0), "ON"),
				e(d, tm(2026, 6, 4, 12, 0), "OFF"),
			},
			wantPrev:   0,
			wantInside: 2,
		},
		{
			name: "mixed",
			events: []ontimerepo.RawEvent{
				e(d.Add(-24*time.Hour), tm(2026, 6, 3, 23, 0), "ON"),
				e(d, tm(2026, 6, 4, 6, 0), "ON"),
				e(d, tm(2026, 6, 4, 12, 0), "OFF"),
			},
			wantPrev:   1,
			wantInside: 2,
		},
		{
			name: "all before",
			events: []ontimerepo.RawEvent{
				e(d.Add(-48*time.Hour), tm(2026, 6, 2, 12, 0), "ON"),
				e(d.Add(-24*time.Hour), tm(2026, 6, 3, 6, 0), "OFF"),
			},
			wantPrev:   2,
			wantInside: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prev, inside := OntimeCalculator{}.splitByDayBoundary(tt.events, d)
			if len(prev) != tt.wantPrev {
				t.Errorf("len(prev) = %d, want %d", len(prev), tt.wantPrev)
			}
			if len(inside) != tt.wantInside {
				t.Errorf("len(inside) = %d, want %d", len(inside), tt.wantInside)
			}
		})
	}
}

func TestDedupEvents(t *testing.T) {
	t06 := tm(2026, 6, 4, 6, 0)
	t12 := tm(2026, 6, 4, 12, 0)
	t18 := tm(2026, 6, 4, 18, 0)

	tests := []struct {
		name   string
		input  []ontimerepo.RawEvent
		output int
	}{
		{
			name:   "empty",
			input:  nil,
			output: 0,
		},
		{
			name:   "single",
			input:  []ontimerepo.RawEvent{{Time: t06}},
			output: 1,
		},
		{
			name: "no duplicates",
			input: []ontimerepo.RawEvent{
				{Time: t06}, {Time: t12}, {Time: t18},
			},
			output: 3,
		},
		{
			name: "adjacent duplicates",
			input: []ontimerepo.RawEvent{
				{Time: t06}, {Time: t06}, {Time: t12}, {Time: t12}, {Time: t18},
			},
			output: 3,
		},
		{
			name: "non-adjacent duplicates",
			input: []ontimerepo.RawEvent{
				{Time: t06}, {Time: t12}, {Time: t06},
			},
			output: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := OntimeCalculator{}.dedupEvents(tt.input)
			if len(got) != tt.output {
				t.Errorf("len(got) = %d, want %d", len(got), tt.output)
			}
		})
	}
}

func TestApplyStartState(t *testing.T) {
	d := day(2026, 6, 4)

	tests := []struct {
		name       string
		prevEvents []ontimerepo.RawEvent
		dayEvents  []ontimerepo.RawEvent
		allEvents  []ontimerepo.RawEvent
		isToday    bool
		wantStatus string
		wantTime   time.Time
	}{
		{
			name:       "prev present, not today",
			prevEvents: []ontimerepo.RawEvent{{Time: tm(2026, 6, 3, 23, 0), Status: "OFF"}},
			isToday:    false,
			wantStatus: "OFF",
			wantTime:   d,
		},
		{
			name:       "prev present, today",
			prevEvents: []ontimerepo.RawEvent{{Time: tm(2026, 6, 3, 23, 0), Status: "ON"}},
			isToday:    true,
			wantStatus: "ON",
			wantTime:   tm(2026, 6, 3, 23, 0),
		},
		{
			name:       "no prev, use first day event",
			dayEvents:  []ontimerepo.RawEvent{{Time: tm(2026, 6, 4, 8, 0), Status: "ON"}},
			allEvents:  []ontimerepo.RawEvent{{Time: tm(2026, 6, 4, 8, 0), Status: "ON"}},
			isToday:    false,
			wantStatus: "ON",
			wantTime:   d,
		},
		{
			name:       "no prev, today, use first day event",
			dayEvents:  []ontimerepo.RawEvent{{Time: tm(2026, 6, 4, 8, 0), Status: "OFF"}},
			allEvents:  []ontimerepo.RawEvent{{Time: tm(2026, 6, 4, 8, 0), Status: "OFF"}},
			isToday:    true,
			wantStatus: "OFF",
			wantTime:   tm(2026, 6, 4, 8, 0),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tl := &Timeline{StartTime: d, EndTime: d.Add(24 * time.Hour)}
			OntimeCalculator{}.applyStartState(tl, tt.prevEvents, tt.dayEvents, tt.allEvents, tt.isToday)

			if tl.StartStatus != tt.wantStatus {
				t.Errorf("StartStatus = %v, want %v", tl.StartStatus, tt.wantStatus)
			}
			if !tl.StartTime.Equal(tt.wantTime) {
				t.Errorf("StartTime = %v, want %v", tl.StartTime, tt.wantTime)
			}
		})
	}
}
