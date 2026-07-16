package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/dto"
	ontimerepo "github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/repository"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/logger"
	"github.com/minhnbnt/uptime-monitor-microservices/ontime-service/internal/infrastructure/serverclient"
)

// ---------- helpers ----------

func oDay(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func oTm(y, m, d, h, min int) time.Time {
	return time.Date(y, time.Month(m), d, h, min, 0, 0, time.UTC)
}

// ---------- buildResponse ----------

func TestBuildResponse(t *testing.T) {
	d1 := oDay(2026, 6, 1)
	d2 := oDay(2026, 6, 2)

	b := &Batcher{}

	t.Run("groups stats by server", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{
			{EndpointID: 1, Date: d1},
			{EndpointID: 1, Date: d2},
			{EndpointID: 2, Date: d1},
		}
		resultMap := map[dto.BatchGetOntimeItem]float64{
			{EndpointID: 1, Date: d1}: 99.5,
			{EndpointID: 1, Date: d2}: 100.0,
			{EndpointID: 2, Date: d1}: 50.0,
		}

		got := b.buildResponse(req, resultMap)

		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}

		// Build lookup to verify
		s1Stats := map[uint][]dto.OntimeStats{}
		for _, r := range got {
			s1Stats[r.EndpointID] = r.Result
		}

		if len(s1Stats[1]) != 2 {
			t.Errorf("server 1 result len = %d, want 2", len(s1Stats[1]))
		}
		if len(s1Stats[2]) != 1 {
			t.Errorf("server 2 result len = %d, want 1", len(s1Stats[2]))
		}

		// Verify values
		for _, stat := range s1Stats[1] {
			if stat.Date.Equal(d1) && stat.Stats != 99.5 {
				t.Errorf("server 1, d1 stats = %f, want 99.5", stat.Stats)
			}
			if stat.Date.Equal(d2) && stat.Stats != 100.0 {
				t.Errorf("server 1, d2 stats = %f, want 100.0", stat.Stats)
			}
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := b.buildResponse(nil, nil)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("missing key defaults to 0", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{
			{EndpointID: 1, Date: d1},
		}
		got := b.buildResponse(req, nil)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if len(got[0].Result) != 1 {
			t.Fatalf("result len = %d, want 1", len(got[0].Result))
		}
		if got[0].Result[0].Stats != 0 {
			t.Errorf("Stats = %f, want 0", got[0].Result[0].Stats)
		}
	})
}

// ---------- buildOntimeLookup ----------

func TestBuildOntimeLookup(t *testing.T) {
	d1 := oDay(2026, 6, 1)
	d2 := oDay(2026, 6, 2)

	t.Run("converts response slice to lookup map", func(t *testing.T) {
		results := []dto.BatchGetOntimeResponse{
			{
				EndpointID: 1,
				Result: []dto.OntimeStats{
					{Date: d1, Stats: 99.5},
					{Date: d2, Stats: 100.0},
				},
			},
			{
				EndpointID: 2,
				Result: []dto.OntimeStats{
					{Date: d1, Stats: 50.0},
				},
			},
		}

		got := buildOntimeLookup(results)

		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[1][d1] != 99.5 {
			t.Errorf("server 1, day 1 = %f, want 99.5", got[1][d1])
		}
		if got[1][d2] != 100.0 {
			t.Errorf("server 1, day 2 = %f, want 100.0", got[1][d2])
		}
		if got[2][d1] != 50.0 {
			t.Errorf("server 2, day 1 = %f, want 50.0", got[2][d1])
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := buildOntimeLookup(nil)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})
}

// ---------- resolveCache ----------

func TestResolveCache(t *testing.T) {
	d1 := oDay(2026, 6, 1)
	d2 := oDay(2026, 6, 2)

	t.Run("returns cached values", func(t *testing.T) {
		keys := []dto.BatchGetOntimeItem{
			{EndpointID: 1, Date: d1},
			{EndpointID: 1, Date: d2},
		}
		cached := map[dto.BatchGetOntimeItem]float64{
			{EndpointID: 1, Date: d1}: 99.0,
			{EndpointID: 1, Date: d2}: 100.0,
		}

		b := &Batcher{
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return cached, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got := b.resolveCache(t.Context(), keys)

		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[keys[0]] != 99.0 {
			t.Errorf("key %+v = %f, want 99.0", keys[0], got[keys[0]])
		}
	})

	t.Run("cache error returns empty map", func(t *testing.T) {
		keys := []dto.BatchGetOntimeItem{
			{EndpointID: 1, Date: d1},
		}
		log, capLog := logger.NewCapturingLogger()

		b := &Batcher{
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return nil, errors.New("redis down")
				},
			},
			logger: log,
		}

		got := b.resolveCache(t.Context(), keys)

		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
		if !capLog.HasWarn() {
			t.Error("expected Warn to be called")
		}
	})

	t.Run("empty keys returns nil", func(t *testing.T) {
		b := &Batcher{
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return nil, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got := b.resolveCache(t.Context(), nil)

		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

// ---------- BatchGetOntimeUntil ----------

func TestOntimeService_BatchGetOntimeUntil(t *testing.T) {
	d1 := oDay(2026, 6, 1)
	until := oTm(2026, 6, 1, 14, 0) // fixed "now"

	t.Run("all cached", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{{EndpointID: 1, Date: d1}}
		cacheResult := map[dto.BatchGetOntimeItem]float64{
			{EndpointID: 1, Date: d1}: 100.0,
		}
		var dbCalled bool
		var mSetCalled bool

		b := &Batcher{
			ontineRepository: &mockOntineRepo{
				batchGetOntimeFn: func(_ context.Context, _ []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error) {
					dbCalled = true
					return nil, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, _ []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return cacheResult, nil
				},
				mSetFn: func(_ context.Context, _ map[dto.BatchGetOntimeItem]float64) error {
					mSetCalled = true
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := b.BatchGetOntimeUntil(t.Context(), req, until)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if dbCalled {
			t.Error("DB should not be called when all cached")
		}
		if mSetCalled {
			t.Error("MSet should not be called when all cached")
		}
		if len(got) != 1 || len(got[0].Result) != 1 {
			t.Fatalf("unexpected result shape: %+v", got)
		}
		if got[0].Result[0].Stats != 100.0 {
			t.Errorf("Stats = %f, want 100.0", got[0].Result[0].Stats)
		}
	})

	t.Run("all miss - fills from DB", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{{EndpointID: 1, Date: d1}}
		var mSetCalled bool
		var capturedItems map[dto.BatchGetOntimeItem]float64

		b := &Batcher{
			ontineRepository: &mockOntineRepo{
				batchGetOntimeFn: func(_ context.Context, _ []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error) {
					return []ontimerepo.RawEvent{
						{EndpointID: 1, Day: d1, Status: "ON", Time: d1.Add(6 * time.Hour)},
					}, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mSetFn: func(_ context.Context, items map[dto.BatchGetOntimeItem]float64) error {
					mSetCalled = true
					capturedItems = items
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := b.BatchGetOntimeUntil(t.Context(), req, until)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !mSetCalled {
			t.Error("MSet should be called when there are misses")
		}
		if len(got) != 1 || len(got[0].Result) != 1 {
			t.Fatalf("unexpected result shape: %+v", got)
		}
		if got[0].Result[0].Stats <= 0 {
			t.Errorf("Stats = %f, want > 0", got[0].Result[0].Stats)
		}
		if capturedItems != nil && len(capturedItems) != 1 {
			t.Errorf("capturedItems len = %d, want 1", len(capturedItems))
		}
	})

	t.Run("DB error logs warning", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{{EndpointID: 1, Date: d1}}
		log, capLog := logger.NewCapturingLogger()

		b := &Batcher{
			ontineRepository: &mockOntineRepo{
				batchGetOntimeFn: func(_ context.Context, _ []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error) {
					return nil, errors.New("db error")
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                log,
		}

		_, err := b.BatchGetOntimeUntil(t.Context(), req, until)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !capLog.HasWarn() {
			t.Error("expected Warn to be called on DB error")
		}
	})

	t.Run("MSet error logs warning", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{{EndpointID: 1, Date: d1}}
		log, capLog := logger.NewCapturingLogger()

		b := &Batcher{
			ontineRepository: &mockOntineRepo{
				batchGetOntimeFn: func(_ context.Context, _ []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error) {
					return []ontimerepo.RawEvent{
						{EndpointID: 1, Day: d1, Status: "ON", Time: d1.Add(6 * time.Hour)},
					}, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mSetFn: func(_ context.Context, _ map[dto.BatchGetOntimeItem]float64) error {
					return errors.New("redis set error")
				},
			},
			logger: log,
		}

		got, err := b.BatchGetOntimeUntil(t.Context(), req, until)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !capLog.HasWarn() {
			t.Error("expected Warn to be called on MSet error")
		}
		if len(got) != 1 || len(got[0].Result) != 1 {
			t.Fatalf("result should still be returned even if MSet fails: %+v", got)
		}
	})
}

// ---------- BatchGetOntime ----------

func TestOntimeService_BatchGetOntime(t *testing.T) {
	d1 := oDay(2026, 6, 1)
	d2 := oDay(2026, 6, 2)
	d3 := oDay(2026, 6, 3)

	req := []dto.BatchGetOntimeItem{
		{EndpointID: 1, Date: d1},
		{EndpointID: 1, Date: d2},
		{EndpointID: 2, Date: d3},
	}

	t.Run("all cached", func(t *testing.T) {
		cacheResult := map[dto.BatchGetOntimeItem]float64{
			{EndpointID: 1, Date: d1}: 99.0,
			{EndpointID: 1, Date: d2}: 100.0,
			{EndpointID: 2, Date: d3}: 50.0,
		}

		b := &Batcher{
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return cacheResult, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := b.BatchGetOntime(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}

		// Group by server for verification
		byServer := map[uint]map[time.Time]float64{}
		for _, r := range got {
			mp := map[time.Time]float64{}
			for _, s := range r.Result {
				mp[s.Date] = s.Stats
			}
			byServer[r.EndpointID] = mp
		}

		if byServer[1][d1] != 99.0 {
			t.Errorf("server 1, d1 = %f, want 99.0", byServer[1][d1])
		}
		if byServer[1][d2] != 100.0 {
			t.Errorf("server 1, d2 = %f, want 100.0", byServer[1][d2])
		}
		if byServer[2][d3] != 50.0 {
			t.Errorf("server 2, d3 = %f, want 50.0", byServer[2][d3])
		}
	})

	t.Run("partially cached", func(t *testing.T) {
		cacheResult := map[dto.BatchGetOntimeItem]float64{
			{EndpointID: 1, Date: d1}: 99.0,
		}

		b := &Batcher{
			ontineRepository: &mockOntineRepo{
				batchGetOntimeFn: func(_ context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error) {
					// Only return events for the missing keys
					events := []ontimerepo.RawEvent{}
					for _, r := range req {
						if r.EndpointID == 1 && r.Date.Equal(d2) {
							events = append(events, ontimerepo.RawEvent{
								EndpointID: 1, Day: d2, Status: "ON",
								Time: d2.Add(6 * time.Hour),
							})
						}
						if r.EndpointID == 2 && r.Date.Equal(d3) {
							events = append(events, ontimerepo.RawEvent{
								EndpointID: 2, Day: d3, Status: "OFF",
								Time: d3.Add(12 * time.Hour),
							})
						}
					}
					return events, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return cacheResult, nil
				},
				mSetFn: func(_ context.Context, _ map[dto.BatchGetOntimeItem]float64) error {
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := b.BatchGetOntime(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}

		// Verify server 1 has d1=99 and d2>0
		for _, r := range got {
			if r.EndpointID == 1 {
				for _, s := range r.Result {
					if s.Date.Equal(d1) && s.Stats != 99.0 {
						t.Errorf("server 1, d1 = %f, want 99.0", s.Stats)
					}
					if s.Date.Equal(d2) && s.Stats <= 0 {
						t.Errorf("server 1, d2 = %f, want > 0", s.Stats)
					}
				}
			}
		}
	})

	t.Run("cache error falls through to DB", func(t *testing.T) {
		var dbCalled bool

		b := &Batcher{
			ontineRepository: &mockOntineRepo{
				batchGetOntimeFn: func(_ context.Context, req []ontimerepo.BatchGetOntimeRequest) ([]ontimerepo.RawEvent, error) {
					dbCalled = true
					return []ontimerepo.RawEvent{
						{EndpointID: 1, Day: d1, Status: "ON", Time: d1.Add(6 * time.Hour)},
						{EndpointID: 1, Day: d2, Status: "ON", Time: d2.Add(8 * time.Hour)},
						{EndpointID: 1, Day: d2, Status: "OFF", Time: d2.Add(12 * time.Hour)},
						{EndpointID: 2, Day: d3, Status: "ON", Time: d3.Add(8 * time.Hour)},
					}, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, _ []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
					return nil, errors.New("redis down")
				},
				mSetFn: func(_ context.Context, _ map[dto.BatchGetOntimeItem]float64) error {
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := b.BatchGetOntime(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !dbCalled {
			t.Error("expected DB to be called when cache errors")
		}
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}

		// Verify all servers have stats > 0
		for _, r := range got {
			for _, s := range r.Result {
				if s.Stats <= 0 {
					t.Errorf("server %d, date %v: stats = %f, want > 0",
						r.EndpointID, s.Date, s.Stats)
				}
			}
		}
	})

	t.Run("empty request returns empty", func(t *testing.T) {
		b := &Batcher{
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                logger.NewMockLogger(),
		}

		got, err := b.BatchGetOntime(t.Context(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("len(got) = %d, want 0", len(got))
		}
	})
}

// ---------- GetServerWithOntime ----------

func TestOntimeService_GetServerWithOntime(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	createdAt := now.AddDate(0, 0, -30)

	t.Run("success", func(t *testing.T) {
		svc := &OntimeService{
			serverClient: &mockServerClient{
				getServerFn: func(_ context.Context, serverID, userID uint) (*serverclient.ServerBrief, error) {
					return &serverclient.ServerBrief{ID: serverID, Name: "server-a", CreatedAt: createdAt}, nil
				},
			},
			batcher: &Batcher{
				ontimeCacheRepository: &mockOntimeCacheRepo{
					mGetFn: func(_ context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
						result := make(map[dto.BatchGetOntimeItem]float64, len(keys))
						for _, k := range keys {
							result[k] = 100.0
						}
						return result, nil
					},
				},
				logger: logger.NewMockLogger(),
			},
			logger: logger.NewMockLogger(),
		}

		got, err := svc.GetServerWithOntime(t.Context(), 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.ServerID != 1 {
			t.Errorf("EndpointID = %d, want 1", got.ServerID)
		}
		if len(got.OntimeStats) == 0 {
			t.Errorf("expected ontime stats, got none")
		}
	})

	t.Run("server not found", func(t *testing.T) {
		svc := &OntimeService{
			serverClient: &mockServerClient{
				getServerFn: func(_ context.Context, _, _ uint) (*serverclient.ServerBrief, error) {
					return nil, errors.New("not found")
				},
			},
			batcher: &Batcher{},
			logger:  logger.NewMockLogger(),
		}

		_, err := svc.GetServerWithOntime(t.Context(), 99, 1)
		if err == nil {
			t.Fatal("expected error for non-existent server")
		}
	})
}

// ---------- ListServersWithOntime ----------

func TestOntimeService_ListServersWithOntime(t *testing.T) {
	now := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	createdAt := now.AddDate(0, 0, -30)

	t.Run("success", func(t *testing.T) {
		svc := &OntimeService{
			serverClient: &mockServerClient{
				listServersFn: func(_ context.Context, userID uint, page, perPage int) ([]serverclient.ServerBrief, error) {
					return []serverclient.ServerBrief{
						{ID: 1, Name: "server-a", CreatedAt: createdAt},
						{ID: 2, Name: "server-b", CreatedAt: createdAt},
					}, nil
				},
			},
			batcher: &Batcher{
				ontimeCacheRepository: &mockOntimeCacheRepo{
					mGetFn: func(_ context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error) {
						result := make(map[dto.BatchGetOntimeItem]float64, len(keys))
						for _, k := range keys {
							result[k] = 100.0
						}
						return result, nil
					},
				},
				logger: logger.NewMockLogger(),
			},
			logger: logger.NewMockLogger(),
		}

		got, err := svc.ListServersWithOntime(t.Context(), 1, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}
		if got[0].ServerID != 1 && got[0].ServerID != 2 {
			t.Errorf("unexpected server id: %d", got[0].ServerID)
		}
		if len(got[0].OntimeStats) == 0 {
			t.Errorf("expected ontime stats for server %d", got[0].ServerID)
		}
	})

	t.Run("empty server list", func(t *testing.T) {
		svc := &OntimeService{
			serverClient: &mockServerClient{
				listServersFn: func(_ context.Context, _ uint, _, _ int) ([]serverclient.ServerBrief, error) {
					return nil, nil
				},
			},
			batcher: &Batcher{},
			logger:  logger.NewMockLogger(),
		}

		got, err := svc.ListServersWithOntime(t.Context(), 1, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("len(got) = %d, want 0", len(got))
		}
	})

	t.Run("server client error", func(t *testing.T) {
		svc := &OntimeService{
			serverClient: &mockServerClient{
				listServersFn: func(_ context.Context, _ uint, _, _ int) ([]serverclient.ServerBrief, error) {
					return nil, errors.New("db error")
				},
			},
			batcher: &Batcher{},
			logger:  logger.NewMockLogger(),
		}

		_, err := svc.ListServersWithOntime(t.Context(), 1, 1, 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
