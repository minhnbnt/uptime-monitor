## Why

`ServerHTTPConfig` lives in its own table (`server_http_configs`) in server-service. Updating or deleting a server's HTTP config only touches that table, so the Debezium stream on `public.servers` never fires — ping-service's scheduler cache keeps stale HTTP config until the 1h TTL expires, or keeps pinging an HTTP_DNS server as HTTP even after its config was removed. ping-service needs to react to HTTP config changes in real time.

## What Changes

- Add `public.server_http_configs` to the Debezium `table.include.list`, producing a new Redis stream `uptime.public.server_http_configs`.
- ping-service consumes the new stream and applies HTTP config changes to the scheduler meta cache immediately:
  - insert/update → write the `http_*` fields + `http_config=1` marker
  - delete → clear the marker and `http_*` fields (server reverts to pod-status ping)
- Generalize the Redis stream consumer to read multiple streams with one read loop, dispatching by stream name.
- Keep a single `messageProcessor` that parses raw CDC envelopes (`json.RawMessage`) and dispatches to per-table handlers by stream name.
- Add `ServerMetaCache.SetHTTPConfig` / `ClearHTTPConfig`; extract a shared hash-values helper reused by `SetMulti`.

## Capabilities

### New Capabilities
- `server-http-config-cdc`: ping-service consumes the `server_http_configs` CDC stream and keeps its scheduler meta cache in sync with HTTP config create/update/delete events.

### Modified Capabilities
<!-- none — no existing specs -->

## Impact

- `config/debezium.yml`: `table.include.list` gains `public.server_http_configs`.
- ping-service:
  - `internal/infrastructure/redis/redisconsumer.go`: multi-stream read + dispatch, `StreamProcessor` interface, exported stream-key consts.
  - `internal/infrastructure/redis/processor.go`: single `messageProcessor` dispatching by stream name (`handleServer`), `cdcMessage` with `json.RawMessage` fields.
  - `internal/infrastructure/redis/httpconfig_processor.go` (new): `debeziumHTTPConfigData`, `HTTPConfigEventHandler`, `handleHTTPConfig`.
  - `internal/infrastructure/scheduler/endpointcache.go`: `SetHTTPConfig`, `ClearHTTPConfig`, shared `httpConfigHashValues`.
  - `internal/service/endpointevent.go`: `HTTPConfigEventHandler` wrapper + multi-stream `Run`.
- Tests: redis processor tests updated to the generic API; new http-config processor tests; scheduler integration test for cache round-trip.
- No server-service code changes (schema already produces the table).
