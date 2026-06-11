package ontime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/minhnbnt/uptime-monitor/internal/domain"
	"github.com/minhnbnt/uptime-monitor/internal/logger"
	ontimerepo "github.com/minhnbnt/uptime-monitor/internal/repository/ontime"
	serverrepo "github.com/minhnbnt/uptime-monitor/internal/repository/server"
	"github.com/minhnbnt/uptime-monitor/internal/server/dto"
	"github.com/minhnbnt/uptime-monitor/internal/utils"
)

// ---------- helpers ----------

func oDay(y, m, d int) time.Time {
	return time.Date(y, time.Month(m), d, 0, 0, 0, 0, time.UTC)
}

func oTm(y, m, d, h, min int) time.Time {
	return time.Date(y, time.Month(m), d, h, min, 0, 0, time.UTC)
}

// ---------- buildCacheKeys ----------

func TestBuildCacheKeys(t *testing.T) {
	d1 := oDay(2026, 6, 1)
	d2 := oDay(2026, 6, 2)
	d3 := oDay(2026, 6, 3)

	svc := &OntimeService{}

	t.Run("deduplicates keys", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{
			{ServerID: 1, Date: d1},
			{ServerID: 1, Date: d1},
			{ServerID: 2, Date: d2},
			{ServerID: 2, Date: d2},
			{ServerID: 1, Date: d3},
		}

		got := svc.buildCacheKeys(req)

		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		if got[0].ServerID != 1 || !got[0].Day.Equal(d1) {
			t.Errorf("got[0] = %+v, want {1, d1}", got[0])
		}
		if got[1].ServerID != 2 || !got[1].Day.Equal(d2) {
			t.Errorf("got[1] = %+v, want {2, d2}", got[1])
		}
		if got[2].ServerID != 1 || !got[2].Day.Equal(d3) {
			t.Errorf("got[2] = %+v, want {1, d3}", got[2])
		}
	})

	t.Run("empty input", func(t *testing.T) {
		got := svc.buildCacheKeys(nil)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("single item", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{
			{ServerID: 5, Date: d1},
		}

		got := svc.buildCacheKeys(req)

		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		if got[0].ServerID != 5 || !got[0].Day.Equal(d1) {
			t.Errorf("got[0] = %+v, want {5, d1}", got[0])
		}
	})
}

// ---------- buildResponse ----------

func TestBuildResponse(t *testing.T) {
	d1 := oDay(2026, 6, 1)
	d2 := oDay(2026, 6, 2)

	svc := &OntimeService{}

	t.Run("groups stats by server", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{
			{ServerID: 1, Date: d1},
			{ServerID: 1, Date: d2},
			{ServerID: 2, Date: d1},
		}
		resultMap := map[ontimerepo.OntimeCacheKey]float64{
			{ServerID: 1, Day: d1}: 99.5,
			{ServerID: 1, Day: d2}: 100.0,
			{ServerID: 2, Day: d1}: 50.0,
		}

		got := svc.buildResponse(req, resultMap)

		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}

		// Build lookup to verify
		s1Stats := map[uint][]dto.OntimeStats{}
		for _, r := range got {
			s1Stats[r.ServerID] = r.Result
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
		got := svc.buildResponse(nil, nil)
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("missing key defaults to 0", func(t *testing.T) {
		req := []dto.BatchGetOntimeItem{
			{ServerID: 1, Date: d1},
		}
		got := svc.buildResponse(req, nil)
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
				ServerID: 1,
				Result: []dto.OntimeStats{
					{Date: d1, Stats: 99.5},
					{Date: d2, Stats: 100.0},
				},
			},
			{
				ServerID: 2,
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
		keys := []ontimerepo.OntimeCacheKey{
			{ServerID: 1, Day: d1},
			{ServerID: 1, Day: d2},
		}
		cached := map[ontimerepo.OntimeCacheKey]float64{
			{ServerID: 1, Day: d1}: 99.0,
			{ServerID: 1, Day: d2}: 100.0,
		}

		svc := &OntimeService{
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					return cached, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got := svc.resolveCache(t.Context(), keys)

		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[keys[0]] != 99.0 {
			t.Errorf("key %+v = %f, want 99.0", keys[0], got[keys[0]])
		}
	})

	t.Run("cache error returns empty map", func(t *testing.T) {
		keys := []ontimerepo.OntimeCacheKey{
			{ServerID: 1, Day: d1},
		}
		log := logger.NewMockLogger()

		svc := &OntimeService{
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					return nil, errors.New("redis down")
				},
			},
			logger: log,
		}

		got := svc.resolveCache(t.Context(), keys)

		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
		if !log.WarnCalled {
			t.Error("expected Warn to be called")
		}
	})

	t.Run("empty keys returns nil", func(t *testing.T) {
		svc := &OntimeService{
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					return nil, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got := svc.resolveCache(t.Context(), nil)

		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

// ---------- fillMisses ----------

func TestFillMisses(t *testing.T) {
	d1 := oDay(2026, 6, 1)

	t.Run("no miss - nothing to fill", func(t *testing.T) {
		keys := []ontimerepo.OntimeCacheKey{
			{ServerID: 1, Day: d1},
		}
		resultMap := map[ontimerepo.OntimeCacheKey]float64{
			{ServerID: 1, Day: d1}: 100.0,
		}
		var mSetCalled bool

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				batchGetOntimeFn: func(_ context.Context, _ []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error) {
					return nil, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mSetFn: func(_ context.Context, _ map[ontimerepo.OntimeCacheKey]float64) error {
					mSetCalled = true
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		svc.fillMisses(t.Context(), resultMap, keys, time.Now())

		if mSetCalled {
			t.Error("MSet should not be called when there are no misses")
		}
	})

	t.Run("all miss - fills from DB", func(t *testing.T) {
		keys := []ontimerepo.OntimeCacheKey{
			{ServerID: 1, Day: d1},
		}
		resultMap := map[ontimerepo.OntimeCacheKey]float64{}
		var mSetCalled bool
		var capturedItems map[ontimerepo.OntimeCacheKey]float64

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				batchGetOntimeFn: func(_ context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error) {
					return []serverrepo.RawEvent{
						{ServerID: 1, Day: d1, Status: "ON", Time: oTm(2026, 6, 1, 6, 0)},
					}, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mSetFn: func(_ context.Context, items map[ontimerepo.OntimeCacheKey]float64) error {
					mSetCalled = true
					capturedItems = items
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		svc.fillMisses(t.Context(), resultMap, keys, time.Now())

		if !mSetCalled {
			t.Error("MSet should be called when there are misses")
		}
		if len(resultMap) != 1 {
			t.Fatalf("resultMap len = %d, want 1", len(resultMap))
		}
		if resultMap[keys[0]] <= 0 {
			t.Errorf("resultMap[%+v] = %f, want > 0", keys[0], resultMap[keys[0]])
		}
		if capturedItems != nil && len(capturedItems) != 1 {
			t.Errorf("capturedItems len = %d, want 1", len(capturedItems))
		}
	})

	t.Run("db error logs warning", func(t *testing.T) {
		keys := []ontimerepo.OntimeCacheKey{
			{ServerID: 1, Day: d1},
		}
		resultMap := map[ontimerepo.OntimeCacheKey]float64{}
		log := logger.NewMockLogger()

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				batchGetOntimeFn: func(_ context.Context, _ []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error) {
					return nil, errors.New("db error")
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                log,
		}

		svc.fillMisses(t.Context(), resultMap, keys, time.Now())

		if !log.WarnCalled {
			t.Error("expected Warn to be called on DB error")
		}
	})

	t.Run("MSet error logs warning", func(t *testing.T) {
		keys := []ontimerepo.OntimeCacheKey{
			{ServerID: 1, Day: d1},
		}
		resultMap := map[ontimerepo.OntimeCacheKey]float64{}
		log := logger.NewMockLogger()

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				batchGetOntimeFn: func(_ context.Context, _ []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error) {
					return []serverrepo.RawEvent{
						{ServerID: 1, Day: d1, Status: "ON", Time: oTm(2026, 6, 1, 6, 0)},
					}, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mSetFn: func(_ context.Context, _ map[ontimerepo.OntimeCacheKey]float64) error {
					return errors.New("redis set error")
				},
			},
			logger: log,
		}

		svc.fillMisses(t.Context(), resultMap, keys, time.Now())

		if !log.WarnCalled {
			t.Error("expected Warn to be called on MSet error")
		}
		// resultMap should still be filled even if cache set fails
		if len(resultMap) != 1 {
			t.Errorf("resultMap len = %d, want 1", len(resultMap))
		}
	})
}

// ---------- BatchGetOntime ----------

func TestOntimeService_BatchGetOntime(t *testing.T) {
	d1 := oDay(2026, 6, 1)
	d2 := oDay(2026, 6, 2)
	d3 := oDay(2026, 6, 3)

	req := []dto.BatchGetOntimeItem{
		{ServerID: 1, Date: d1},
		{ServerID: 1, Date: d2},
		{ServerID: 2, Date: d3},
	}

	t.Run("all cached", func(t *testing.T) {
		cacheResult := map[ontimerepo.OntimeCacheKey]float64{
			{ServerID: 1, Day: d1}: 99.0,
			{ServerID: 1, Day: d2}: 100.0,
			{ServerID: 2, Day: d3}: 50.0,
		}

		svc := &OntimeService{
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					return cacheResult, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := svc.BatchGetOntime(t.Context(), req)
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
			byServer[r.ServerID] = mp
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
		cacheResult := map[ontimerepo.OntimeCacheKey]float64{
			{ServerID: 1, Day: d1}: 99.0,
		}

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				batchGetOntimeFn: func(_ context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error) {
					// Only return events for the missing keys
					events := []serverrepo.RawEvent{}
					for _, r := range req {
						if r.ServerID == 1 && r.Date.Equal(d2) {
							events = append(events, serverrepo.RawEvent{
								ServerID: 1, Day: d2, Status: "ON",
								Time: d2.Add(6 * time.Hour),
							})
						}
						if r.ServerID == 2 && r.Date.Equal(d3) {
							events = append(events, serverrepo.RawEvent{
								ServerID: 2, Day: d3, Status: "OFF",
								Time: d3.Add(12 * time.Hour),
							})
						}
					}
					return events, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					return cacheResult, nil
				},
				mSetFn: func(_ context.Context, _ map[ontimerepo.OntimeCacheKey]float64) error {
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := svc.BatchGetOntime(t.Context(), req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}

		// Verify server 1 has d1=99 and d2>0
		for _, r := range got {
			if r.ServerID == 1 {
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

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				batchGetOntimeFn: func(_ context.Context, req []serverrepo.BatchGetOntimeRequest) ([]serverrepo.RawEvent, error) {
					dbCalled = true
					return []serverrepo.RawEvent{
						{ServerID: 1, Day: d1, Status: "ON", Time: d1.Add(6 * time.Hour)},
						{ServerID: 1, Day: d2, Status: "ON", Time: d2.Add(8 * time.Hour)},
						{ServerID: 1, Day: d2, Status: "OFF", Time: d2.Add(12 * time.Hour)},
						{ServerID: 2, Day: d3, Status: "ON", Time: d3.Add(8 * time.Hour)},
					}, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, _ []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					return nil, errors.New("redis down")
				},
				mSetFn: func(_ context.Context, _ map[ontimerepo.OntimeCacheKey]float64) error {
					return nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := svc.BatchGetOntime(t.Context(), req)
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
						r.ServerID, s.Date, s.Stats)
				}
			}
		}
	})

	t.Run("empty request returns empty", func(t *testing.T) {
		svc := &OntimeService{
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                logger.NewMockLogger(),
		}

		got, err := svc.BatchGetOntime(t.Context(), nil)
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
	dates := utils.Last30Days()
	oldTime := dates[0].Add(-48 * time.Hour)

	t.Run("success", func(t *testing.T) {
		server := domain.Server{
			Model:  gormModel(1, oldTime),
			Name:   "server-a",
			Status: domain.StatusActive,
		}

		cacheResult := make(map[ontimerepo.OntimeCacheKey]float64)
		for _, d := range dates {
			cacheResult[ontimerepo.OntimeCacheKey{ServerID: 1, Day: d}] = 100.0
		}

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				getByIDFn: func(_ context.Context, id uint) (*domain.Server, error) {
					if id != 1 {
						t.Errorf("GetByID id = %d, want 1", id)
					}
					return &server, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					result := make(map[ontimerepo.OntimeCacheKey]float64, len(keys))
					for _, k := range keys {
						if v, ok := cacheResult[k]; ok {
							result[k] = v
						}
					}
					return result, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, err := svc.GetServerWithOntime(t.Context(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.Server.Name != "server-a" {
			t.Errorf("Server.Name = %q, want server-a", got.Server.Name)
		}
		if len(got.OntimeStats) != len(dates) {
			t.Errorf("len(OntimeStats) = %d, want %d", len(got.OntimeStats), len(dates))
		}
	})

	t.Run("server not found", func(t *testing.T) {
		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				getByIDFn: func(_ context.Context, _ uint) (*domain.Server, error) {
					return nil, errors.New("record not found")
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                logger.NewMockLogger(),
		}

		_, err := svc.GetServerWithOntime(t.Context(), 99)
		if err == nil {
			t.Fatal("expected error for non-existent server")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				getByIDFn: func(_ context.Context, _ uint) (*domain.Server, error) {
					return nil, errors.New("db error")
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                logger.NewMockLogger(),
		}

		_, err := svc.GetServerWithOntime(t.Context(), 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// ---------- ListServersWithOntime ----------

func TestOntimeService_ListServersWithOntime(t *testing.T) {
	dates := utils.Last30Days()
	oldTime := dates[0].Add(-48 * time.Hour)

	t.Run("success with cached data", func(t *testing.T) {
		servers := []domain.Server{
			{Model: gormModel(1, oldTime), Name: "server-a", Status: domain.StatusActive},
			{Model: gormModel(2, oldTime), Name: "server-b", Status: domain.StatusPaused},
		}

		// Build cache: return 100% for all server-date combinations
		server1Keys := make([]ontimerepo.OntimeCacheKey, len(dates))
		for i, d := range dates {
			server1Keys[i] = ontimerepo.OntimeCacheKey{ServerID: 1, Day: d}
		}
		server2Keys := make([]ontimerepo.OntimeCacheKey, len(dates))
		for i, d := range dates {
			server2Keys[i] = ontimerepo.OntimeCacheKey{ServerID: 2, Day: d}
		}

		cacheResult := make(map[ontimerepo.OntimeCacheKey]float64)
		for _, k := range server1Keys {
			cacheResult[k] = 100.0
		}
		for _, k := range server2Keys {
			cacheResult[k] = 100.0
		}

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				listFn: func(_ context.Context, createdByID uint, limit, offset int) ([]domain.Server, error) {
					if createdByID != 1 {
						t.Errorf("List createdByID = %d, want 1", createdByID)
					}
					if limit != 10 || offset != 0 {
						t.Errorf("List(%d, %d)", limit, offset)
					}
					return servers, nil
				},
				countFn: func(_ context.Context, createdByID uint) (int64, error) {
					if createdByID != 1 {
						t.Errorf("Count createdByID = %d, want 1", createdByID)
					}
					return 2, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					// Return whatever was requested
					result := make(map[ontimerepo.OntimeCacheKey]float64, len(keys))
					for _, k := range keys {
						if v, ok := cacheResult[k]; ok {
							result[k] = v
						}
					}
					return result, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, total, err := svc.ListServersWithOntime(t.Context(), 1, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 2 {
			t.Errorf("total = %d, want 2", total)
		}
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}

		// Verify server names
		if got[0].Server.Name != "server-a" && got[0].Server.Name != "server-b" {
			t.Errorf("unexpected server name: %s", got[0].Server.Name)
		}
		if got[1].Server.Name != "server-a" && got[1].Server.Name != "server-b" {
			t.Errorf("unexpected server name: %s", got[1].Server.Name)
		}

		// Verify both servers have ontime stats for each date
		for _, s := range got {
			if len(s.OntimeStats) != len(dates) {
				t.Errorf("server %s: len(OntimeStats) = %d, want %d",
					s.Server.Name, len(s.OntimeStats), len(dates))
			}
		}
	})

	t.Run("empty server list", func(t *testing.T) {
		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				listFn: func(_ context.Context, _ uint, limit, offset int) ([]domain.Server, error) {
					return nil, nil
				},
				countFn: func(_ context.Context, _ uint) (int64, error) {
					return 0, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                logger.NewMockLogger(),
		}

		got, total, err := svc.ListServersWithOntime(t.Context(), 1, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d, want 0", total)
		}
		if len(got) != 0 {
			t.Errorf("len(got) = %d, want 0", len(got))
		}
	})

	t.Run("server repo list error", func(t *testing.T) {
		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				listFn: func(_ context.Context, _ uint, limit, offset int) ([]domain.Server, error) {
					return nil, errors.New("db error")
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                logger.NewMockLogger(),
		}

		_, _, err := svc.ListServersWithOntime(t.Context(), 1, 1, 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("server repo count error", func(t *testing.T) {
		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				listFn: func(_ context.Context, _ uint, limit, offset int) ([]domain.Server, error) {
					return []domain.Server{
						{Model: gormModel(1, oldTime), Name: "s1", Status: domain.StatusActive},
					}, nil
				},
				countFn: func(_ context.Context, _ uint) (int64, error) {
					return 0, errors.New("count error")
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{},
			logger:                logger.NewMockLogger(),
		}

		_, _, err := svc.ListServersWithOntime(t.Context(), 1, 1, 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("new server with limited date range", func(t *testing.T) {
		// A server created 5 days ago should only have 5 dates
		fiveDaysAgo := dates[len(dates)-1].AddDate(0, 0, -4)
		servers := []domain.Server{
			{Model: gormModel(3, fiveDaysAgo), Name: "new-server", Status: domain.StatusActive},
		}

		svc := &OntimeService{
			serverRepository: &mockServerRepo{
				listFn: func(_ context.Context, _ uint, limit, offset int) ([]domain.Server, error) {
					return servers, nil
				},
				countFn: func(_ context.Context, _ uint) (int64, error) {
					return 1, nil
				},
			},
			ontimeCacheRepository: &mockOntimeCacheRepo{
				mGetFn: func(_ context.Context, keys []ontimerepo.OntimeCacheKey) (map[ontimerepo.OntimeCacheKey]float64, error) {
					result := make(map[ontimerepo.OntimeCacheKey]float64, len(keys))
					for _, k := range keys {
						result[k] = 100.0
					}
					return result, nil
				},
			},
			logger: logger.NewMockLogger(),
		}

		got, total, err := svc.ListServersWithOntime(t.Context(), 1, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 1 {
			t.Errorf("total = %d, want 1", total)
		}
		if len(got) != 1 {
			t.Fatalf("len(got) = %d, want 1", len(got))
		}
		if got[0].Server.Name != "new-server" {
			t.Errorf("Server.Name = %q, want new-server", got[0].Server.Name)
		}
		// Should have fewer stats than all 30 days
		if len(got[0].OntimeStats) >= len(dates) {
			t.Errorf("len(OntimeStats) = %d, should be < %d for new server",
				len(got[0].OntimeStats), len(dates))
		}
	})
}
