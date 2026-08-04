## Context

See proposal.md (Why). The ontime pipeline today loses the "no data" distinction because the calculator returns a bare `float64`, empty status from a `LEFT JOIN` is treated as `OFF`, and the cache stores only numbers. Two draft rewrites (`changes.md`, `changes2.md`) fixed the calculator and cache layers but missed two breakages this design folds in:

- `Batcher.fillMisses` computes one global `dayUntil := TruncateDay(until)` and feeds it to `CalculateDayOntime` for every key. The current calculator anchors its window off `events[0].AnchorTime`, which is why it still works; the rewritten thin-wrapper calculator uses the `today` argument as the window base, so past-day keys would be measured against today's window. The batcher must supply each key's own day.
- `ontime.go`'s `buildOntimeLookup`/`getServersOntime` re-map results to `float64`, which would silently drop `HasData` on the public API path (compiles, wrong output).

## Goals / Non-Goals

**Goals:**
- A single calculator result type (`OntimeResult`) carrying `Uptime`, `OnlineSeconds`, `TotalSeconds`, `HasData`, `Partial`, `ObservedFrom`.
- One gate (`asStatus`) deciding what counts as known status: exactly `ON`/`OFF`.
- "No data" survives every layer: calculator → batcher → Redis → DTO/API.
- A day's uptime is always computed over that day's own window.

**Non-Goals:**
- Fixing the ping-worker race that produces same-timestamp conflicting events (only deterministic ordering for that case; see Decisions).
- Schema/migration of existing Redis keys (TTLs are short; see Risks).
- Frontend rendering changes beyond consuming the new `has_data`/`partial` flags.

## Decisions

### D1. `OntimeResult` replaces `float64` everywhere the calculator is called
`CalculateOntime` and `CalculateDayOntime` return `OntimeResult`. Callers that only need the percentage (interval buckets, cache) project down explicitly; nothing re-derives semantics from a number.
- Alternative rejected: a second `(float64, bool)` return — carries only one flag, no room for `Partial`/`ObservedFrom`, and every caller would need its own convention.

### D2. `asStatus` is the single place that classifies raw status
Only `domain.StatusOn`/`domain.StatusOff` map to known; empty and anything else → unknown. This fixes the silent `""` → `OFF` bug in one place rather than at each call site.

### D3. `CalculateDayOntime` is a thin wrapper, and the batcher passes the target day
`CalculateDayOntime(events, today, now)` computes `[today, today+24h)`, clamping the end to `now` only when `today` is the current day, then delegates to `CalculateOntime` (which handles the no-boundary partial case). The critical counterpart: `Batcher.fillMisses` must call it with `today = TruncateDay(key.Date)`, not the global `dayUntil`. This keeps the past-day vs. current-day behavior correct and makes the previously-hidden dependency between batcher and calculator explicit.

### D4. `DayResult` as the cache/batcher transport unit, decoupled from API shape
`dto.DayResult{HasData bool, Uptime float64}` flows calculator → batcher → Redis. Public `dto.OntimeStats` gains `HasData` and is built from `DayResult` at the response edge. This keeps the Redis/cache shape free to evolve without dragging the JSON contract, and vice versa.
- Alternative rejected: reusing `OntimeResult` in cache — carries `Partial`/`ObservedFrom`/seconds that the per-day stats path doesn't expose, and couples cache encoding to API result fields.

### D5. Redis encodes no-data as a sentinel, strictly decoded
`encode`: `"-"` for no-data, else `fmt.Sprintf("%.2f", uptime)`. `decode`: sentinel → `HasData:false`; non-numeric or anything else → cache miss (`ok=false`). A corrupted/foreign value is never guessed into a real percentage.
- Alternative rejected: storing `-1` as a "no data" float — could survive into arithmetic or display paths and be misread as a real value.

### D6. `mergeIntervals` merges only buckets that agree on `HasData` (and, when both have data, on uptime)
A no-data bucket never collapses into a `0%` bucket.

### D7. Deterministic ordering for same-timestamp events (repository layer)
Keep same-timestamp/different-status events in the calculation (zero-width interval preserves the final state) but make their relative order deterministic at the source: add event `id` as a secondary `ORDER BY` key in `rangeEventSQL`/`dayEventSQL`, and add `WHERE status IS NOT NULL` to `rangeEventSQL`'s `lowerbound` to match `dayEventSQL`.

### D8. `HasData` surfaces at both HTTP and gRPC boundaries
The public HTTP API (ogen) adds `has_data` to `OntimeStats`, `UptimeResponse`, and `IntervalResult`, plus `partial` on `UptimeResponse`; the handler maps the `dto` fields into the regenerated `api` responses. The gRPC `OntimeDayStat` appends `bool has_data = 3` (backward-compatible — existing consumers ignore the new field), threaded from `ontime_grpc.go`. On the consumer side, `notification-service` carries `HasData` through its `domain.OntimeStats` and `ontimeclient`, and `digest.buildReport` drops no-data days from the per-day map so the existing Excel `-` fallback renders "no data" instead of `0%`.
- Alternative rejected: leaving the gRPC digest on numeric-only stats — a fresh server would keep rendering `0%` in the digest, the same bug this change removes elsewhere.

## Risks / Trade-offs

- **Old cache keys hold numeric `0.00` from the no-data bug** → after deploy these decode as `HasData:true, 0%`. TTLs are 1h (past days) / 10s (today), so this self-heals quickly; acceptable transient.
- **`mergeIntervals` compares uptime with `==`** → adjacent buckets are produced by the same event slice over identical sub-patterns, so float results are bit-identical in practice. If that ever diverges, the trade is a missed merge, not a wrong number.
- **`Partial` windows report `TotalSeconds` for the observed sub-window while `From`/`To` reflect the requested range** → internal consistency is preserved (`Uptime = OnlineSeconds/TotalSeconds`); the `partial` flag exists precisely so clients don't mistake the shorter window for the full request. The range response intentionally does not expose `ObservedFrom` yet — revisit if a client needs the exact data-start.
- **Signature changes break callers and tests** → the change updates `ontime.go`, `ontimerange.go`, `batcher.go`, cache repo, handler, generated API, proto, notification-service, and the affected test files in the same change so each module compiles and `go test ./...` passes.
- **Proto/API regeneration drift** → both `generated/api` (ogen) and `common/proto/generated` are checked in; the change regenerates them from `ontime-service/api/spec.yaml` and `common/proto/event/v1` respectively so no hand-edited stub diverges.
- **Rolling deploy of the proto field** → `has_data = 3` is appended, so an old `notification-service` binary ignores it; both modules must still ship together to actually surface the flag in the digest.

## Migration Plan

1. Land calculator + DTO + cache + batcher + service + handler + proto + notification-service changes as one commit so every module stays compilable.
2. Regenerate `generated/api` (ogen) and `common/proto/generated` (buf) as part of that commit.
3. Deploy; Redis keys self-heal via TTL.
4. Rollback: revert the single commit; old numeric cache values are still parseable by the old decoder.

## Open Questions

None.
