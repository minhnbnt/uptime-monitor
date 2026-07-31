## Context

- ping-service consumes Debezium CDC from server-service via a Redis stream. The Debezium Redis sink writes one stream per topic, named `<prefix>.<schema>.<table>`; the existing consumer reads `uptime.public.servers`.
- `ServerHTTPConfig` lives in its own table `server_http_configs` (PK = `server_id`). server-service upserts/deletes it independently of the `servers` row (see `server.go` UpdateServer/CreateServer), so `servers` CDC does not fire on HTTP config changes.
- The scheduler meta cache (`ServerMetaCache`) stores each server as a Redis hash with `http_config=1` marker + `http_port/http_endpoint_path/http_expected_code/http_body_check_expr/http_method` fields. `mapToServer` builds `ServerHTTPConfig` when the marker is present; absent marker → pod-status ping.
- The existing `messageProcessor` handles parse → dispatch → ack for one stream; parse/ack logic is identical across streams, only the payload type and handler differ.

## Goals / Non-Goals

**Goals:**
- React to HTTP config create/update/delete in real time (no 1h TTL wait, no stale HTTP_DNS after config removal).
- Reuse the CDC parse/dispatch/ack machinery for both streams via a single generic processor.
- Keep the single-consumer read loop (one XReadGroup across streams, dispatch by stream name).

**Non-Goals:**
- No changes to server-service (table already produced by GORM).
- No ordering guarantees between the two streams (see Risks).
- No backfill/migration of existing cache entries; periodic gRPC refresh already covers cache misses.

## Decisions

### D1. Add table to Debezium config
`config/debezium.yml` `table.include.list` → `public.servers,public.server_http_configs`. Sink produces stream `uptime.public.server_http_configs`.

### D2. One message processor, `json.RawMessage`, switch by stream name
Keep a single `messageProcessor` that parses a raw CDC envelope and dispatches per table:
- `cdcMessage{Before, After json.RawMessage, Op string}` — one envelope for all tables; `Before`/`After` are unmarshalled lazily inside each table's handler.
- `ProcessMessage(ctx, stream string, msg)` — extract `value`, unmarshal `cdcMessage`, then `switch stream { case ServerStreamKey: …; case HTTPConfigStreamKey: … }`.
- Per-table handler (`handleServer` / `handleHTTPConfig`) switches on `Op`, unmarshals `After`/`Before` into the table's struct, and calls the domain handler. Nil `After` on create/update → ack without calling the handler.
- Delete: shared helper `deletedData(event)` returns the `Before` row (falling back to `After`) as raw JSON; each table unmarshals it to read its own ID column (`id` vs `server_id`) — no `GetID`/`Identifiable` needed.

Rationale: with only two tables the op-dispatch differs per table (HTTP maps both `c`/`r` and `u` to `OnSet`), so a per-table switch is more honest than forcing both tables through one `EventHandler[T]` mapping plus adapters. It drops `dataMessage[T]`, `Identifiable`, and the adapter layer entirely. Alternative considered: generics (`StreamEventProcessor[T]` + `EventHandler[T]`) — closed core for new tables, but ~2× the machinery for the same result today; revisit if a third table appears and the switch grows.

### D3. Multi-stream consumer
`StreamEventConsumer` reads all streams in one `XReadGroup` call; a single processor receives the stream name per message:
- `StreamProcessor interface{ ProcessMessage(ctx, stream string, msg redis.XMessage) bool }`
- `Run(ctx, streams []string, p StreamProcessor)` — create a consumer group per stream, read `[]string{s1,">",s2,">"}`, call `p.ProcessMessage(ctx, stream.Stream, msg)`, `XAck` on the matching stream.
- Exported consts `ServerStreamKey = "uptime.public.servers"`, `HTTPConfigStreamKey = "uptime.public.server_http_configs"`.
- The single `messageProcessor` is built by the service layer with both handlers and passed in; the consumer no longer constructs processors.

Alternative considered: a `map[string]StreamProcessor` so each stream has its own processor instance (no stream name in the interface). Rejected — the switch lives in the processor anyway, and one processor keeps handler wiring in one place.

### D4. Cache write paths
Extract `httpConfigHashValues(cfg *domain.ServerHTTPConfig) []any` (marker + 5 fields) reused by `SetMulti` and the new:
- `SetHTTPConfig(ctx, id, cfg)` → `HSet` the shared values + refresh TTL.
- `ClearHTTPConfig(ctx, id)` → `HDel` the 6 hash fields (marker + fields).
Clear leaves identity fields intact; `mapToServer` then yields `HTTPConfig == nil`.

### D5. Service wiring
`EndpointEventService` gains an `HTTPConfigEventHandler` (wraps `ServerMetaCache` → `SetHTTPConfig`/`ClearHTTPConfig`) and a logger. `Run(ctx)` builds one processor and passes the stream list:
```go
p := redis.NewMessageProcessor(s.eventHandler, s.httpHandler, s.logger)
s.consumer.Run(ctx, []string{redis.ServerStreamKey, redis.HTTPConfigStreamKey}, p)
```
No change to `app.go` — `RegisterEventService` already injects cache, scheduler, and consumer.

## Risks / Trade-offs

- [No cross-stream ordering] Debezium writes the two streams independently; a `server_http_configs` insert could be processed before/after its `servers` event. `HSet` merges fields and `mapToServer` skips keys missing required identity fields, so the cache converges; a transient partial key is self-healing (or refreshed via gRPC on miss).
- [In-flight vs CDC race] A CDC http-config event could be overwritten by a concurrent `SetMulti` from a gRPC batch refresh writing stale config. Bounded by the refresh path only running on cache miss; acceptable for a 1h-TTL cache.
- [Unknown op / malformed message] Not acked → retried → can block the stream if permanent. Same behavior as the existing `servers` stream; kept for consistency.

## Migration Plan

1. Deploy config change (add table to Debezium) + ping-service code together. Debezium picks up the new table without restarting consumers; if the new stream is empty, behavior is unchanged.
2. Rollback: revert config `table.include.list` (stops new stream) and revert ping-service changes; cache still eventually correct via gRPC refresh/TTL.

## Open Questions

None — deferred unknowns (ordering, in-flight races above) are accepted trade-offs, not blockers.
