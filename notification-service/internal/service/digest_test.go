package service

import (
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/domain"
	"github.com/minhnbnt/uptime-monitor-microservices/notification-service/internal/infrastructure/utils"
)

func TestBuildReport_DropsNoDataDays(t *testing.T) {
	svc := &DigestService{}

	servers := []domain.Server{{ID: 1, Name: "server-a"}}

	d1 := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)

	ontimeMap := map[uint][]domain.OntimeStats{
		1: {
			{Date: d1, Stats: 80, HasData: true},
			{Date: d2, Stats: 0, HasData: false}, // no data → must be dropped
		},
	}

	rows := svc.buildReport(servers, ontimeMap)
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}

	stats := rows[0].Stats
	if _, ok := stats[utils.TruncateDay(d1)]; !ok {
		t.Error("data day missing from stats map")
	}
	if _, ok := stats[utils.TruncateDay(d2)]; ok {
		t.Error("no-data day should be absent so Excel renders the no-data marker, got present")
	}
}

func TestBuildReport_KeepsZeroWithData(t *testing.T) {
	svc := &DigestService{}

	servers := []domain.Server{{ID: 1, Name: "server-a"}}
	d := time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)

	ontimeMap := map[uint][]domain.OntimeStats{
		1: {{Date: d, Stats: 0, HasData: true}}, // genuinely 0% — must stay
	}

	rows := svc.buildReport(servers, ontimeMap)
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if v, ok := rows[0].Stats[utils.TruncateDay(d)]; !ok || v != 0 {
		t.Errorf("0%% day with HasData = true should be kept, got %+v", rows[0].Stats)
	}
}
