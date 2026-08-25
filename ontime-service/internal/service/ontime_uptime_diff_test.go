package service

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/testcontainers"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/utils"
)

// Differential spike: the SQL-native OntimeUptimeRepository must produce the
// exact same per-(endpoint, day) uptime as the existing Batcher pipeline
// (raw events streamed to Go, OntimeCalculator) on identical data. Any drift
// here means the SQL rewrite changed semantics somewhere.
func TestIntegration_UptimeSQL_MatchesGoPipeline(t *testing.T) {
	testcontainers.SkipIfShort(t)
	db := initTestDB(t)

	rng := rand.New(rand.NewSource(42))

	base := oDay(2026, 5, 1)
	days := 12

	flip := func() domain.ServerStatus {
		if rng.Intn(2) == 0 {
			return domain.StatusOn
		}
		return domain.StatusOff
	}

	// Endpoints 1–3 get pseudo-random event chains; 4 gets a single event
	// long before the window; 5 gets nothing at all.
	for ep := uint(1); ep <= 4; ep++ {
		status := domain.StatusOn
		if ep == 4 {
			seedEvent(t, db, ep, status, base.Add(-72*time.Hour))
			continue
		}

		next := base.Add(time.Duration(rng.Intn(72)) * time.Hour)
		end := base.AddDate(0, 0, days)
		for next.Before(end) {
			seedEvent(t, db, ep, status, next)
			status = flip()
			next = next.Add(time.Duration(1+rng.Intn(20)) * time.Hour)
		}
	}

	// Every endpoint × every day in the window, plus one day before anything
	// happened (no history → shrunk window / no-data cases).
	var items []dto.BatchGetOntimeItem
	var reqs []ontimerepo.BatchGetOntimeRequest
	for ep := uint(1); ep <= 5; ep++ {
		for d := -1; d < days; d++ {
			day := base.AddDate(0, 0, d)
			items = append(items, dto.BatchGetOntimeItem{EndpointID: ep, Date: day})
			reqs = append(reqs, ontimerepo.BatchGetOntimeRequest{EndpointID: ep, Date: day})
		}
	}

	// Clamp only the LAST day: until sits mid-afternoon of day `days-1`,
	// so earlier days use their full 24h window and the last one ends early.
	until := base.AddDate(0, 0, days-1).Add(15 * time.Hour)

	b := newBatcher(t, db)
	oldResults, err := b.BatchGetOntimeUntil(t.Context(), items, until)
	if err != nil {
		t.Fatalf("old pipeline: %v", err)
	}

	oldByKey := map[uint]map[time.Time]float64{}
	oldHasData := map[uint]map[time.Time]bool{}
	for _, r := range oldResults {
		for _, s := range r.Result {
			if oldByKey[r.EndpointID] == nil {
				oldByKey[r.EndpointID] = map[time.Time]float64{}
				oldHasData[r.EndpointID] = map[time.Time]bool{}
			}
			oldByKey[r.EndpointID][utils.TruncateDay(s.Date)] = s.Stats
			oldHasData[r.EndpointID][utils.TruncateDay(s.Date)] = s.HasData
		}
	}

	repo := ontimerepo.NewOntimeUptimeRepository(db)
	newRows, err := repo.BatchGetUptime(t.Context(), reqs, until)
	if err != nil {
		t.Fatalf("new sql repo: %v", err)
	}

	newByKey := map[uint]map[time.Time]float64{}
	for _, row := range newRows {
		if newByKey[row.EndpointID] == nil {
			newByKey[row.EndpointID] = map[time.Time]float64{}
		}
		newByKey[row.EndpointID][utils.TruncateDay(row.Day)] = row.Uptime
	}

	compared := 0
	for ep := uint(1); ep <= 5; ep++ {
		for d := -1; d < days; d++ {
			day := utils.TruncateDay(base.AddDate(0, 0, d))

			oldUptime, oldData := oldByKey[ep][day], oldHasData[ep][day]
			newUptime, hadNew := newByKey[ep][day]

			if !oldData {
				if hadNew {
					t.Errorf("ep %d %s: SQL returned %.4f but Go says HasData=false", ep, day.Format("2006-01-02"), newUptime)
				}
				continue
			}
			if !hadNew {
				t.Errorf("ep %d %s: Go says %.4f (HasData=true) but SQL returned no row", ep, day.Format("2006-01-02"), oldUptime)
				continue
			}

			if math.Abs(oldUptime-newUptime) > 0.01 {
				t.Errorf("ep %d %s: uptime drift — Go %.6f vs SQL %.6f",
					ep, day.Format("2006-01-02"), oldUptime, newUptime)
			}
			compared++
		}
	}

	if compared == 0 {
		t.Fatal("no rows compared — generator produced nothing usable")
	}
	t.Logf("compared %d (endpoint, day) pairs: SQL matches Go pipeline", compared)
}
