package repository

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/domain"
)

func countEvents(tb testing.TB, db *gorm.DB, endpointID uint) int64 {
	tb.Helper()
	var n int64
	if err := db.Model(&domain.ServerEvent{}).
		Where("endpoint_id = ?", endpointID).
		Count(&n).Error; err != nil {
		tb.Fatalf("count events: %v", err)
	}
	return n
}

func TestServerEventSaveDedupe(t *testing.T) {
	ctx := context.Background()

	t.Run("predecessor same status is skipped", func(t *testing.T) {
		db := initUptimeTestDB(t)
		repo := NewServerEventRepository(db, slog.Default())
		id := uint(1)

		seedUptimeRow(t, db, id, domain.StatusOn, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		if err := repo.Save(ctx, &domain.ServerEvent{
			EndpointID: id, Status: domain.StatusOn,
			Time: time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC),
		}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		if n := countEvents(t, db, id); n != 1 {
			t.Errorf("rows = %d, want 1 (redundant ON skipped)", n)
		}
	})

	t.Run("inserting in the middle of two same-status events is skipped", func(t *testing.T) {
		db := initUptimeTestDB(t)
		repo := NewServerEventRepository(db, slog.Default())
		id := uint(2)

		seedUptimeRow(t, db, id, domain.StatusOn, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		seedUptimeRow(t, db, id, domain.StatusOn, time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC))
		if err := repo.Save(ctx, &domain.ServerEvent{
			EndpointID: id, Status: domain.StatusOn,
			Time: time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC),
		}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		if n := countEvents(t, db, id); n != 2 {
			t.Errorf("rows = %d, want 2 (middle ON skipped)", n)
		}
	})

	t.Run("successor same status is collapsed", func(t *testing.T) {
		db := initUptimeTestDB(t)
		repo := NewServerEventRepository(db, slog.Default())
		id := uint(3)

		seedUptimeRow(t, db, id, domain.StatusOff, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		seedUptimeRow(t, db, id, domain.StatusOn, time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC))
		if err := repo.Save(ctx, &domain.ServerEvent{
			EndpointID: id, Status: domain.StatusOn,
			Time: time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC),
		}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		// OFF(t0) + ON(t0+30); the later ON(t0+60) is collapsed away.
		if n := countEvents(t, db, id); n != 2 {
			t.Fatalf("rows = %d, want 2 (successor collapsed)", n)
		}
		var remaining domain.ServerEvent
		if err := db.Where("endpoint_id = ?", id).Order("time DESC").First(&remaining).Error; err != nil {
			t.Fatalf("last: %v", err)
		}
		if remaining.Status != domain.StatusOn || !remaining.Time.Equal(time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC)) {
			t.Errorf("last event = %s@%v, want ON@t0+30", remaining.Status, remaining.Time)
		}
	})

	t.Run("status change is inserted", func(t *testing.T) {
		db := initUptimeTestDB(t)
		repo := NewServerEventRepository(db, slog.Default())
		id := uint(4)

		seedUptimeRow(t, db, id, domain.StatusOff, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		if err := repo.Save(ctx, &domain.ServerEvent{
			EndpointID: id, Status: domain.StatusOn,
			Time: time.Date(2026, 1, 1, 0, 0, 30, 0, time.UTC),
		}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		if n := countEvents(t, db, id); n != 2 {
			t.Errorf("rows = %d, want 2 (status change inserted)", n)
		}
	})

	t.Run("retry at the same timestamp is not duplicated", func(t *testing.T) {
		db := initUptimeTestDB(t)
		repo := NewServerEventRepository(db, slog.Default())
		id := uint(5)

		tm := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		seedUptimeRow(t, db, id, domain.StatusOn, tm)
		if err := repo.Save(ctx, &domain.ServerEvent{
			EndpointID: id, Status: domain.StatusOn, Time: tm,
		}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		if n := countEvents(t, db, id); n != 1 {
			t.Errorf("rows = %d, want 1 (same-T retry skipped)", n)
		}
	})
}
