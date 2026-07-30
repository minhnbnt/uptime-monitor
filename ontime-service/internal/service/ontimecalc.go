package service

import (
	"sort"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

type Timeline struct {
	Day         time.Time
	StartTime   time.Time
	EndTime     time.Time
	StartStatus string
	Events      []ontimerepo.Event
}

type OntimeCalculator struct{}

func calcUptimePercent(online, total float64) float64 {

	if total > 0 {
		return online / total * 100
	}

	return 0
}

func (o OntimeCalculator) CalculateOntime(events []ontimerepo.Event, startTime, endTime time.Time) float64 {

	if len(events) == 0 {
		return 0
	}

	t := o.buildTimeline(events, startTime, endTime)
	online := o.calculateOnlineDuration(t)

	return calcUptimePercent(online, t.EndTime.Sub(t.StartTime).Seconds())
}

func (o OntimeCalculator) CalculateDayOntime(events []ontimerepo.Event, today time.Time, now time.Time) float64 {

	if len(events) == 0 {
		return 0
	}

	day := utils.TruncateDay(events[0].AnchorTime)
	startTime, endTime := day, day.Add(24*time.Hour)
	if startTime.Equal(today) {

		endTime = now

		index := sort.Search(len(events), func(i int) bool {
			return !events[i].Time.Before(day)
		})

		if index > 0 {
			startTime = events[index-1].Time
		}

		if startTime.Equal(day) && len(events) > 0 {
			startTime = events[0].Time
		}
	}

	uptime := o.CalculateOntime(events, startTime, endTime)
	if uptime > 0 {
		return uptime
	}

	if domain.ServerStatus(events[0].Status) == domain.StatusOn {
		return 100
	}

	return 0
}

func (o OntimeCalculator) buildTimeline(events []ontimerepo.Event, startTime, endTime time.Time) Timeline {

	t := Timeline{
		StartTime: startTime,
		EndTime:   endTime,
	}

	if len(events) > 0 {
		t.Day = utils.TruncateDay(events[0].AnchorTime)
	}

	prevEvents, dayEvents := o.splitEventsByRange(events, startTime, endTime)
	o.applyStartState(&t, prevEvents, dayEvents)
	t.Events = o.dedupEvents(dayEvents)

	return t
}

func (o OntimeCalculator) splitEventsByRange(events []ontimerepo.Event, startTime, endTime time.Time) (prev, inside []ontimerepo.Event) {

	firstInside := sort.Search(len(events), func(i int) bool {
		return !events[i].Time.Before(startTime)
	})

	pastEnd := sort.Search(len(events), func(i int) bool {
		return events[i].Time.After(endTime)
	})

	return events[:firstInside], events[firstInside:pastEnd]
}

func (o OntimeCalculator) applyStartState(t *Timeline, prevEvents, dayEvents []ontimerepo.Event) {

	if len(prevEvents) > 0 {
		t.StartStatus = prevEvents[len(prevEvents)-1].Status
		return
	}

	if len(dayEvents) > 0 {
		t.StartStatus = dayEvents[0].Status
	}
}

func (o OntimeCalculator) dedupEvents(events []ontimerepo.Event) []ontimerepo.Event {

	if len(events) <= 1 {
		return events
	}

	unique := []ontimerepo.Event{events[0]}
	for i := 1; i < len(events); i++ {
		if !events[i].Time.Equal(events[i-1].Time) {
			unique = append(unique, events[i])
		}
	}

	return unique
}

func (o OntimeCalculator) calculateOnlineDuration(t Timeline) float64 {

	total := 0.0

	prevTime, prevStatus := t.StartTime, t.StartStatus
	for _, e := range t.Events {

		if domain.ServerStatus(prevStatus) == domain.StatusOn {
			total += e.Time.Sub(prevTime).Seconds()
		}

		prevStatus, prevTime = e.Status, e.Time
	}

	if domain.ServerStatus(prevStatus) == domain.StatusOn {

		dur := t.EndTime.Sub(prevTime).Seconds()

		if dur > 0 {
			total += dur
		}
	}

	return total
}
