package service

import (
	"sort"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

// OntimeResult is the outcome of computing uptime over a requested window.
//
// HasData reports whether the server's state was actually known at some
// point in the window. When false, Uptime/OnlineSeconds carry no meaning
// and must not be shown as "0%".
//
// Partial reports that the window had to be shrunk because state was
// unknown at the very start (no event happened before it) — ObservedFrom
// is where real data begins. Callers should surface this so a user isn't
// shown a number that silently ignores part of the requested range.
type OntimeResult struct {
	Uptime        float64
	OnlineSeconds float64
	TotalSeconds  float64
	HasData       bool
	Partial       bool
	ObservedFrom  time.Time
}

type OntimeCalculator struct{}

type timeline struct {
	StartTime   time.Time
	EndTime     time.Time
	StartStatus domain.ServerStatus
	HasStart    bool
	Partial     bool
	Events      []ontimerepo.Event
}

func (o OntimeCalculator) CalculateOntime(events []ontimerepo.Event, from, to time.Time) OntimeResult {

	t := o.newTimeline(events, from, to)

	if !t.HasStart {
		return OntimeResult{TotalSeconds: to.Sub(from).Seconds()}
	}

	online := onlineSeconds(t)
	total := t.EndTime.Sub(t.StartTime).Seconds()

	return OntimeResult{
		Uptime:        calcUptimePercent(online, total),
		OnlineSeconds: online,
		TotalSeconds:  total,
		HasData:       true,
		Partial:       t.Partial,
		ObservedFrom:  t.StartTime,
	}
}

// CalculateDayOntime is now a thin wrapper around CalculateOntime — no more
// special-cased fallback that peeked at events[0] (the earliest event ever
// recorded for the server, unrelated to "today").
func (o OntimeCalculator) CalculateDayOntime(events []ontimerepo.Event, today, now time.Time) OntimeResult {

	dayEnd := today.Add(24 * time.Hour)
	if today.Equal(utils.TruncateDay(now)) {
		dayEnd = now
	}

	return o.CalculateOntime(events, today, dayEnd)
}

func calcUptimePercent(online, total float64) float64 {
	if total > 0 {
		return online / total * 100
	}
	return 0
}

func (o OntimeCalculator) newTimeline(events []ontimerepo.Event, from, to time.Time) timeline {

	t := timeline{StartTime: from, EndTime: to}

	prevEvents, dayEvents := splitByRange(events, from, to)

	if len(prevEvents) > 0 {
		t.StartStatus, t.HasStart = domain.ToServerStatus(prevEvents[len(prevEvents)-1].Status)
		t.Events = dedupExact(dayEvents)
		return t
	}

	// No real event happened before `from` (or the repository's boundary
	// row was a NULL-joined placeholder with no status). Scan forward for
	// the first event in range with a known status instead of guessing —
	// state before that point is genuinely unknown, so the window shrinks
	// to start there and gets flagged Partial.
	for i, e := range dayEvents {
		status, known := domain.ToServerStatus(e.Status)
		if !known {
			continue
		}

		t.StartTime = e.Time
		t.StartStatus = status
		t.HasStart = true
		t.Partial = !e.Time.Equal(from)
		t.Events = dedupExact(dayEvents[i+1:])
		return t
	}

	return t // HasStart stays false: no data anywhere in the range
}

func splitByRange(events []ontimerepo.Event, from, to time.Time) (prev, inside []ontimerepo.Event) {

	firstInside := sort.Search(len(events), func(i int) bool {
		return !events[i].Time.Before(from)
	})

	pastEnd := sort.Search(len(events), func(i int) bool {
		return events[i].Time.After(to)
	})

	return events[:firstInside], events[firstInside:pastEnd]
}

// dedupExact merges consecutive events identical in both time and status —
// true duplicates. Events sharing a timestamp but disagreeing on status are
// kept, not dropped: that pattern is a sign of a race between two ping
// workers upstream, contributes a harmless zero-width interval here, and
// stays visible for debugging rather than vanishing silently. Deterministic
// ordering for that case should ultimately come from a secondary sort key
// (event id) added to the repository's SQL, not from this function.
func dedupExact(events []ontimerepo.Event) []ontimerepo.Event {

	if len(events) <= 1 {
		return events
	}

	unique := events[:1]
	for i := 1; i < len(events); i++ {

		prev := unique[len(unique)-1]
		isSame := events[i].Time.Equal(prev.Time) && events[i].Status == prev.Status

		if !isSame {
			unique = append(unique, events[i])
		}
	}

	return unique
}

func onlineSeconds(t timeline) float64 {

	total := 0.0
	prevTime, prevStatus := t.StartTime, t.StartStatus

	for _, e := range t.Events {

		status, known := domain.ToServerStatus(e.Status)

		// Unknown status (e.g. an empty/NULL no-data row) is intentionally
		// skipped WITHOUT advancing the boundary time: the server's known
		// state must not change on a meaningless row, or uptime would be
		// skewed. The next known event still measures from the last known
		// boundary. (asStatus is the single gate deciding what is "known",
		// so unknown here always means no-data, never a typo'd status.)
		if !known {
			continue
		}

		if prevStatus == domain.StatusOn {
			total += e.Time.Sub(prevTime).Seconds()
		}

		prevTime, prevStatus = e.Time, status
	}

	if prevStatus == domain.StatusOn {
		if dur := t.EndTime.Sub(prevTime).Seconds(); dur > 0 {
			total += dur
		}
	}

	return total
}
