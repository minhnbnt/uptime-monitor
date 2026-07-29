package service

import (
	"sort"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
)

type Timeline struct {
	Day         time.Time
	StartTime   time.Time
	EndTime     time.Time
	StartStatus string
	Events      []ontimerepo.RawEvent
}

type OntimeCalculator struct{}

func calcUptimePercent(online, total float64) float64 {
	if total > 0 {
		return online / total * 100
	}
	return 0
}

func CalculateOntime(events []ontimerepo.RawEvent, startTime, endTime time.Time) float64 {

	if len(events) == 0 {
		return 0
	}

	t := OntimeCalculator{}.BuildTimeline(events, startTime, endTime)
	online := OntimeCalculator{}.CalculateOnlineDuration(t)

	return calcUptimePercent(online, t.EndTime.Sub(t.StartTime).Seconds())
}

func (OntimeCalculator) CalculateDayOntime(events []ontimerepo.RawEvent, today time.Time, now time.Time) float64 {

	if len(events) == 0 {
		return 0
	}

	day := events[0].Day
	startTime := day
	endTime := day.Add(24 * time.Hour)
	if day.Equal(today) {
		endTime = now

		for _, e := range events {
			if e.Time.Before(day) {
				startTime = e.Time
			}
		}

		if startTime.Equal(day) && len(events) > 0 {
			startTime = events[0].Time
		}
	}

	uptime := CalculateOntime(events, startTime, endTime)
	if uptime > 0 {
		return uptime
	}

	if domain.ServerStatus(events[0].Status) == domain.StatusOn {
		return 100
	}
	return 0
}

func (o OntimeCalculator) BuildTimeline(events []ontimerepo.RawEvent, startTime, endTime time.Time) Timeline {

	t := Timeline{
		StartTime: startTime,
		EndTime:   endTime,
	}

	if len(events) > 0 {
		t.Day = events[0].Day
	}

	prevEvents, dayEvents := o.splitEventsByRange(events, startTime, endTime)
	o.applyStartState(&t, prevEvents, dayEvents)
	t.Events = o.dedupEvents(dayEvents)

	return t
}

func (o OntimeCalculator) splitEventsByRange(events []ontimerepo.RawEvent, startTime, endTime time.Time) (prev, inside []ontimerepo.RawEvent) {

	firstInside := sort.Search(len(events), func(i int) bool {
		return !events[i].Time.Before(startTime)
	})

	pastEnd := sort.Search(len(events), func(i int) bool {
		return events[i].Time.After(endTime)
	})

	return events[:firstInside], events[firstInside:pastEnd]
}

func (o OntimeCalculator) applyStartState(t *Timeline, prevEvents, dayEvents []ontimerepo.RawEvent) {

	if len(prevEvents) > 0 {
		t.StartStatus = prevEvents[len(prevEvents)-1].Status
		return
	}

	if len(dayEvents) > 0 {
		t.StartStatus = dayEvents[0].Status
	}
}

func (o OntimeCalculator) dedupEvents(events []ontimerepo.RawEvent) []ontimerepo.RawEvent {

	if len(events) <= 1 {
		return events
	}

	unique := []ontimerepo.RawEvent{events[0]}
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
	total := 0.0

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
