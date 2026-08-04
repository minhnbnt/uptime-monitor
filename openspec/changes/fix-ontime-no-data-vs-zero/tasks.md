## 1. Calculator

- [x] 1.1 Define `OntimeResult` in `internal/service/ontimecalc.go` with `Uptime`, `OnlineSeconds`, `TotalSeconds`, `HasData`, `Partial`, `ObservedFrom`
- [x] 1.2 Add `asStatus(raw string)` as the single gate mapping exactly `ON`/`OFF` to known status, everything else unknown
- [x] 1.3 Rewrite `CalculateOntime` to return `OntimeResult`: no-known-state returns `HasData:false` with full requested `TotalSeconds`; otherwise compute `OnlineSeconds`/`TotalSeconds`/`Uptime`/`Partial`/`ObservedFrom`
- [x] 1.4 Rewrite `newTimeline`: use `asStatus` on the last prior event; when no known prior event, forward-scan for the first known in-window event, shrink `StartTime` to it, and flag `Partial` (unless it exactly equals the window start)
- [x] 1.5 Replace `dedupEvents` with `dedupExact` that drops only same-time AND same-status duplicates and keeps same-time/different-status events
- [x] 1.6 Make `onlineSeconds` skip unknown-status events without advancing the boundary time
- [x] 1.7 Rewrite `CalculateDayOntime` as a thin wrapper over `CalculateOntime` computing `[today, today+24h)` clamped to `now` only for the current day; reuse `utils.TruncateDay` instead of redeclaring

## 2. DTOs

- [x] 2.1 Add `dto.DayResult{HasData bool, Uptime float64}` in `internal/dto/ontime.go`; add `HasData bool` to `OntimeStats`
- [x] 2.2 In `internal/dto/ontimerange.go` add `HasData`/`Partial` to `UptimeResponse` and `HasData` to `IntervalResult`

## 3. Cache

- [x] 3.1 In `internal/infrastructure/repository/ontimecache.go` switch `MGet`/`MSet` to `map[...]dto.DayResult`
- [x] 3.2 Add `noDataValue = "-"` sentinel with `encode`/`decode`: decode treats non-sentinel/non-numeric values as cache miss

## 4. Batcher

- [x] 4.1 Thread `dto.DayResult` through `OntimeCacheRepository` interface, `resolveCache`, `fillMisses`, `buildResponse`
- [x] 4.2 In `fillMisses`, call `CalculateDayOntime` with the per-key day `TruncateDay(key.Date)` instead of the global `dayUntil`, mapping `HasData`/`Uptime` into `DayResult`
- [x] 4.3 In `buildResponse`, build `OntimeStats` from `DayResult` carrying `HasData` (zero value naturally means no-data)

## 5. Range service

- [x] 5.1 In `internal/service/ontimerange.go` `CalculateUptime`, use the returned `OntimeResult` fields (`Uptime`, `HasData`, `Partial`, `TotalSeconds`, `OnlineSeconds`) instead of re-deriving `onlineSeconds` from a percentage
- [x] 5.2 Make `mergeIntervals` merge only buckets agreeing on `HasData` (and, when both have data, on uptime); never merge a no-data bucket into a `0%` bucket
- [x] 5.3 Update `CalculateIntervals` to project each interval into `IntervalResult{From, To, Uptime, HasData}`

## 6. Ontime service

- [x] 6.1 Change `buildOntimeLookup` to build `map[uint]map[time.Time]dto.DayResult` so `HasData` survives
- [x] 6.2 Update `getServersOntime` to build `OntimeStats` from `DayResult` with `HasData`

## 7. API exposure

- [x] 7.1 Add `has_data` to `OntimeStats`/`UptimeResponse`/`IntervalResult` and `partial` to `UptimeResponse` in `ontime-service/api/spec.yaml`
- [x] 7.2 Regenerate the ogen API (`go tool ogen` via the `go:generate` directive in `cmd/main.go`)
- [x] 7.3 Update `handler/ontimehandler.go` `toOntimeStats`/`toAPIUptimeResponse` to map `HasData`/`Partial` into the regenerated `api.*` responses

## 8. Proto / gRPC

- [x] 8.1 Append `bool has_data = 3` to `OntimeDayStat` in `common/proto/event/v1/event_service.proto`
- [x] 8.2 Regenerate `common/proto/generated` with `buf generate`
- [x] 8.3 Map `HasData` into the proto in `handler/ontime_grpc.go`
- [x] 8.4 Add `HasData bool` to `notification-service/internal/domain` `OntimeStats`; parse it in `ontimeclient/client.go`
- [x] 8.5 In `notification-service` `digest.buildReport`, drop no-data days from the per-day stats map so the existing Excel `-` fallback renders "no data"

## 9. Repository SQL

- [x] 9.1 Add event `id` as a secondary `ORDER BY` key in `rangeEventSQL`/`dayEventSQL` for deterministic same-timestamp ordering
- [x] 9.2 Add `WHERE status IS NOT NULL` to `rangeEventSQL`'s `lowerbound` to match `dayEventSQL`

## 10. Tests

- [x] 10.1 Replace `ontimecalc_test.go` with tests covering: no-data (`HasData:false`, full `TotalSeconds`), known boundary, partial no-boundary (`ObservedFrom`, shrunk window), conflicting same-timestamp kept
- [x] 10.2 Update `ontimerange_test.go`: `TestCalculateRangeOntime` asserts `OntimeResult`; `TestBuildTimeline` matches the new timeline shape; add `mergeIntervals` no-data-vs-0% case
- [x] 10.3 Update `ontime_test.go` cache mocks (`MGet`/`MSet`) to `map[dto.BatchGetOntimeItem]dto.DayResult`
- [x] 10.4 Update `ontime_integration_test.go`/`ontime_integration_extra_test.go` assertions for `DayResult`/`OntimeStats.HasData`
- [x] 10.5 Add a per-day window test proving a past-day key is computed over its own day, not the current window
- [x] 10.6 Update handler tests and `notification-service` ontimeclient/digest tests for the new `has_data` fields where assertions depend on them

## 11. Verification

- [x] 11.1 `go build ./...` in `ontime-service`
- [x] 11.2 `go test ./...` in `ontime-service` passes
- [x] 11.3 `go build ./...` and `go test ./...` pass in `notification-service`
- [x] 11.4 `go test ./...` passes in `common/proto`
- [x] 11.5 `openspec validate` passes for this change
