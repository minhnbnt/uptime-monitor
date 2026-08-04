## Why

The ontime-service conflates "this range/day has no recorded state yet" with "the server was down (0% uptime)". Because empty status comes back from a `LEFT JOIN` as an empty string, the calculator silently treats it as `OFF`, numeric results are coerced to `float64` at every layer (calculator → cache → response), and a brand-new server shows `0%` instead of "no data". The range endpoint (`CalculateUptime`) is currently the worst offender, but the same broken semantics flow through `OntimeCacheRepository` (Redis), `Batcher`, and the per-day dashboard path.

## What Changes

- Make the calculator return a structured result (`OntimeResult`) with `HasData`, `Partial`, and `ObservedFrom` instead of a bare `float64`.
- Treat only `"ON"`/`"OFF"` as known status; empty/unknown (NULL-joined) status is reported as "no data", never silently coerced to `0%`.
- Drop the `events[0]` fallback hack in `CalculateDayOntime`; when no known event precedes the window, shrink the observed window to the first known event and flag it `Partial`.
- Thread `HasData` through the cache and public DTOs so "no data" survives Redis and reaches API callers as a distinct value.
- Expose `has_data` (and `partial` on the range response) on the public HTTP API via the ogen spec, and carry `has_data` on the gRPC `OntimeDayStat` so the notification-service digest can also distinguish "no data" from `0%`.
- **Fixes land that the initial drafts missed:**
  - `Batcher.fillMisses` computes a single global `dayUntil` and passes it to every per-day key, so the day window is wrong (past days are measured against today's window). Must pass each key's own day.
  - `ontime.go`'s `buildOntimeLookup`/`getServersOntime` map ontime to `float64`, dropping `HasData` before it reaches the response — must thread `DayResult`.
- Update the affected DTOs, cache encoding, and tests that assert on the old `float64` shapes.

## Capabilities

### New Capabilities
- `ontime`: On-time (uptime) calculation over events for a window or day, distinguishing "no data" from "0%", reporting partial observed windows, and propagating that distinction through the cache and public APIs.

### Modified Capabilities

## Impact

- `ontime-service/internal/service/ontimecalc.go` — `OntimeResult`, `asStatus`, `CalculateOntime`, `CalculateDayOntime`, `newTimeline`, `dedupExact`, `onlineSeconds`.
- `ontime-service/internal/service/ontimerange.go` — range response carries `HasData`/`Partial`; `mergeIntervals` no longer merges a no-data bucket into a `0%` bucket.
- `ontime-service/internal/service/batcher.go` — threads `DayResult`; passes each key's own day window.
- `ontime-service/internal/dto/ontime.go` — adds `DayResult`, adds `HasData` to `OntimeStats`/`IntervalResult`/`UptimeResponse`.
- `ontime-service/api/spec.yaml` + regenerated `generated/api` — `OntimeStats`/`UptimeResponse`/`IntervalResult` gain `has_data` (and `UptimeResponse` `partial`); regenerated via ogen.
- `ontime-service/internal/handler/ontimehandler.go` — maps `has_data`/`partial` into the ogen responses.
- `ontime-service/internal/handler/ontime_grpc.go` — maps `HasData` into the gRPC `OntimeDayStat`.
- `common/proto/event/v1/event_service.proto` + regenerated code — `OntimeDayStat` gains `bool has_data = 3` (appended, backward-compatible).
- `notification-service/internal/domain/server.go`, `notification-service/internal/infrastructure/ontimeclient/client.go`, `notification-service/internal/service/digest.go` — thread `HasData`; digest drops no-data days so the Excel `-` fallback renders "no data".
- `ontime-service/internal/infrastructure/repository/ontimecache.go` — Redis value becomes a sentinel `"-"` for no-data vs. a numeric percentage; `MGet`/`MSet` use `DayResult`.
- `ontime-service/internal/repository/ontime.go` — optional: deterministic `ORDER BY` for same-timestamp events; `WHERE status IS NOT NULL` consistency on `rangeEventSQL`'s `lowerbound`.
- Tests: `ontimecalc_test.go`, `ontimerange_test.go`, `ontime_test.go`, `ontime_integration_test.go`, `ontime_integration_extra_test.go` updated to the new signatures/shapes; handler and notification-service client/digest updated where assertions depend on the new fields.