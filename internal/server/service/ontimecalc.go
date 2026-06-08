package service

import (
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
)

type Timeline struct {
	Day         time.Time
	StartTime   time.Time
	EndTime     time.Time
	StartStatus string
	Events      []serverrepo.RawEvent
}

type OntimeCalculator struct{}

func (OntimeCalculator) CalculateDayOntime(events []serverrepo.RawEvent, today time.Time, now time.Time) float64 {

	if len(events) == 0 {
		return 0
	}

	t := OntimeCalculator{}.BuildTimeline(events, today, now)
	online := OntimeCalculator{}.CalculateOnlineDuration(t)
	coverage := t.EndTime.Sub(t.StartTime).Seconds()

	if coverage > 0 {
		return online / coverage * 100
	}

	if domain.ServerStatus(t.StartStatus) == domain.StatusOn {
		return 100
	}
	return 0
}

func (o OntimeCalculator) BuildTimeline(events []serverrepo.RawEvent, today time.Time, now time.Time) Timeline {
	day := events[0].Day

	t := Timeline{
		Day:       day,
		StartTime: day,
		EndTime:   day.Add(24 * time.Hour),
	}

	if day.Equal(today) {
		t.EndTime = now
	}

	prevEvents, dayEvents := o.splitByDayBoundary(events, day)
	o.applyStartState(&t, prevEvents, dayEvents, events, day.Equal(today))
	t.Events = o.dedupEvents(dayEvents)

	return t
}

func (o OntimeCalculator) splitByDayBoundary(events []serverrepo.RawEvent, day time.Time) (prev, inside []serverrepo.RawEvent) {

	dayEnd := day.Add(24 * time.Hour)

	for _, e := range events {
		if e.Time.Before(day) {
			prev = append(prev, e)
		} else if e.Time.Before(dayEnd) {
			inside = append(inside, e)
		}
	}

	return
}

func (o OntimeCalculator) applyStartState(t *Timeline, prevEvents, dayEvents, allEvents []serverrepo.RawEvent, isToday bool) {

	if len(prevEvents) > 0 {

		last := prevEvents[len(prevEvents)-1]

		t.StartStatus = last.Status
		if isToday {
			t.StartTime = last.Time
		}

		return
	}

	if len(allEvents) == 0 {
		return
	}

	fallback := allEvents[0]

	switch {
	case len(dayEvents) > 0:
		fallback = dayEvents[0]
	}

	t.StartStatus = fallback.Status
	if isToday {
		t.StartTime = fallback.Time
	}
}

func (o OntimeCalculator) dedupEvents(events []serverrepo.RawEvent) []serverrepo.RawEvent {

	if len(events) <= 1 {
		return events
	}

	unique := []serverrepo.RawEvent{events[0]}
	for i := 1; i < len(events); i++ {
		if !events[i].Time.Equal(events[i-1].Time) {
			unique = append(unique, events[i])
		}
	}

	return unique
}

func (o OntimeCalculator) CalculateOnlineDuration(t Timeline) float64 {

	prevTime := t.StartTime
	prevStatus := t.StartStatus
	var total float64

	for _, e := range t.Events {
		if domain.ServerStatus(prevStatus) == domain.StatusOn {
			total += e.Time.Sub(prevTime).Seconds()
		}
		prevStatus = e.Status
		prevTime = e.Time
	}

	if domain.ServerStatus(prevStatus) == domain.StatusOn {
		dur := t.EndTime.Sub(prevTime).Seconds()
		if dur > 0 {
			total += dur
		}
	}

	return total
}
