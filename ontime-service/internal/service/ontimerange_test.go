package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/logger"
)

type mockUptimeRepo struct {
	batchGetUptimeFn func(ctx context.Context, req []repository.BatchGetOntimeRequest) ([]repository.UptimeRow, error)
}

func (m *mockUptimeRepo) BatchGetUptime(ctx context.Context, req []repository.BatchGetOntimeRequest) ([]repository.UptimeRow, error) {
	if m.batchGetUptimeFn == nil {
		return nil, nil
	}
	return m.batchGetUptimeFn(ctx, req)
}

var _ UptimeRepository = (*mockUptimeRepo)(nil)

func TestCalculateUptime(t *testing.T) {
	log := logger.NewMockLogger()
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("success with data", func(t *testing.T) {
		svc := &OntimeRangeService{
			uptimeRepo: &mockUptimeRepo{
				batchGetUptimeFn: func(_ context.Context, req []repository.BatchGetOntimeRequest) ([]repository.UptimeRow, error) {
					if len(req) == 1 {
						return []repository.UptimeRow{{
							EndpointID:    1,
							From:          base,
							To:            base.Add(24 * time.Hour),
							ObservedFrom:  base,
							ObservedTo:    base.Add(24 * time.Hour),
							OnlineSeconds: 82800,
						}}, nil
					}
					results := make([]repository.UptimeRow, len(req))
					for i, r := range req {
						results[i] = repository.UptimeRow{
							EndpointID:    r.EndpointID,
							From:          r.From,
							To:            r.To,
							ObservedFrom:  r.From,
							ObservedTo:    r.To,
							OnlineSeconds: r.To.Sub(r.From).Seconds(),
						}
					}
					return results, nil
				},
			},
			ownerRepo: &mockServerOwnerRepo{
				ownedServersFn: func(_ context.Context, _ uint, _ []uint) ([]repository.OwnedServer, error) {
					return []repository.OwnedServer{{ServerID: 1}}, nil
				},
			},
			logger: log,
		}

		resp, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{
			ServerID:   1,
			UserID:     1,
			From:       base,
			To:         base.Add(24 * time.Hour),
			Resolution: 15 * time.Minute,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.ServerID != 1 {
			t.Errorf("server_id = %d, want 1", resp.ServerID)
		}
		if !resp.HasData {
			t.Error("expected has_data = true")
		}
		if len(resp.Intervals) == 0 {
			t.Error("expected intervals")
		}
	})

	t.Run("forbidden", func(t *testing.T) {
		svc := &OntimeRangeService{
			uptimeRepo: &mockUptimeRepo{},
			ownerRepo: &mockServerOwnerRepo{
				ownedServersFn: func(_ context.Context, _ uint, _ []uint) ([]repository.OwnedServer, error) {
					return nil, nil
				},
			},
			logger: log,
		}

		_, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{
			ServerID:   1,
			UserID:     1,
			From:       base,
			To:         base.Add(time.Hour),
			Resolution: 15 * time.Minute,
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid range from >= to", func(t *testing.T) {
		svc := &OntimeRangeService{
			uptimeRepo: &mockUptimeRepo{},
			ownerRepo: &mockServerOwnerRepo{
				ownedServersFn: func(_ context.Context, _ uint, _ []uint) ([]repository.OwnedServer, error) {
					return []repository.OwnedServer{{ServerID: 1}}, nil
				},
			},
			logger: log,
		}

		_, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{
			ServerID:   1,
			UserID:     1,
			From:       base.Add(time.Hour),
			To:         base,
			Resolution: 15 * time.Minute,
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("range exceeds 90 days", func(t *testing.T) {
		svc := &OntimeRangeService{
			uptimeRepo: &mockUptimeRepo{},
			ownerRepo: &mockServerOwnerRepo{
				ownedServersFn: func(_ context.Context, _ uint, _ []uint) ([]repository.OwnedServer, error) {
					return []repository.OwnedServer{{ServerID: 1}}, nil
				},
			},
			logger: log,
		}

		_, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{
			ServerID:   1,
			UserID:     1,
			From:       base,
			To:         base.Add(91 * 24 * time.Hour),
			Resolution: 15 * time.Minute,
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("repo error", func(t *testing.T) {
		svc := &OntimeRangeService{
			uptimeRepo: &mockUptimeRepo{
				batchGetUptimeFn: func(_ context.Context, _ []repository.BatchGetOntimeRequest) ([]repository.UptimeRow, error) {
					return nil, errors.New("db error")
				},
			},
			ownerRepo: &mockServerOwnerRepo{
				ownedServersFn: func(_ context.Context, _ uint, _ []uint) ([]repository.OwnedServer, error) {
					return []repository.OwnedServer{{ServerID: 1}}, nil
				},
			},
			logger: log,
		}

		_, err := svc.CalculateUptime(t.Context(), dto.CalculateUptimeInput{
			ServerID:   1,
			UserID:     1,
			From:       base,
			To:         base.Add(time.Hour),
			Resolution: 15 * time.Minute,
		})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestMergeIntervals(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		got := mergeIntervals(nil)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("single", func(t *testing.T) {
		in := []dto.IntervalResult{{From: "a", To: "b", Uptime: 100, HasData: true}}
		got := mergeIntervals(in)
		if len(got) != 1 {
			t.Errorf("len = %d, want 1", len(got))
		}
	})

	t.Run("merge same uptime", func(t *testing.T) {
		in := []dto.IntervalResult{
			{From: "a", To: "b", Uptime: 100, HasData: true},
			{From: "b", To: "c", Uptime: 100, HasData: true},
		}
		got := mergeIntervals(in)
		if len(got) != 1 {
			t.Errorf("len = %d, want 1", len(got))
		}
		if got[0].From != "a" || got[0].To != "c" {
			t.Errorf("interval = %v-%v, want a-c", got[0].From, got[0].To)
		}
	})

	t.Run("no merge different uptime", func(t *testing.T) {
		in := []dto.IntervalResult{
			{From: "a", To: "b", Uptime: 100, HasData: true},
			{From: "b", To: "c", Uptime: 50, HasData: true},
		}
		got := mergeIntervals(in)
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
	})

	t.Run("no merge different has_data", func(t *testing.T) {
		in := []dto.IntervalResult{
			{From: "a", To: "b", Uptime: 0, HasData: false},
			{From: "b", To: "c", Uptime: 0, HasData: true},
		}
		got := mergeIntervals(in)
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
	})
}

func TestValidateRange(t *testing.T) {
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	t.Run("valid", func(t *testing.T) {
		if err := validateRange(base, base.Add(time.Hour)); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("from after to", func(t *testing.T) {
		if err := validateRange(base.Add(time.Hour), base); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("from equals to", func(t *testing.T) {
		if err := validateRange(base, base); err == nil {
			t.Error("expected error")
		}
	})

	t.Run("exceeds 90 days", func(t *testing.T) {
		if err := validateRange(base, base.Add(91*24*time.Hour)); err == nil {
			t.Error("expected error")
		}
	})
}
