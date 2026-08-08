## 1. DomainCache mới

- [x] 1.1 `internal/infrastructure/scheduler/domaincache.go` (mới) — key `scheduler:domain:<namespace>:<kind>:<object_id>`, Redis string, TTL 1h; `Get(ctx, ns, kind, objectID) (string, bool, error)`, `Set(...)` (skip `domain == ""`), `Delete(...)`; `RegisterDomainCache` dùng `RedisClientWrapper` như `ServerMetaCache`. Chỉ Pod-kind ghi vào thực tế

## 2. Meta cache thu hẹp scope

- [x] 2.1 `endpointcache.go` `SetMulti` — bỏ write `label_selector` + `domain`; chỉ giữ identity + `interval_ns`/`timeout_ns` + http_config marker/fields (`httpConfigHashValues`)
- [x] 2.2 `endpointcache.go` `mapToServer` — bỏ reconstruct `K8sRuntime`; `sv.K8s` luôn nil
- [x] 2.3 `endpointprovider.go` — xoá `fillK8sFields` + dependency `k8sClient`; GetBatch = MGet → miss → gRPC → SetMulti (dọn import `dto` không còn dùng)

## 3. URLResolverService dùng domain cache

- [x] 3.1 `url.go` — inject `*scheduler.DomainCache` (qua interface cục bộ `domainCache` cho test được); `ResolveURL` = `resolveCached`: nếu `params.Kind != "Pod"` gọi thẳng `k8sClient.ResolveDomainName` (Service/STS compute, không đụng cache); nếu Pod → Get → hit buildURL, miss → resolve → `Set` (best-effort) → buildURL
- [x] 3.2 `url.go` — `ResolveDomain` giữ raw (gọi thẳng `k8sClient.ResolveDomainName`, không cache) cho stale compare

## 4. Stale path nhắm domain cache

- [x] 4.1 `pingloop.go` — D1 = `url.Hostname()` từ ResolveURL; bỏ check `k8sParams.K8s` (dòng 115 còn `Kind != "Pod"`); bỏ hack `freshParams`/`freshK8s`; `ResolveDomain(ctx, k8sParams)` trực tiếp
- [x] 4.2 `pingloop.go` — khi `D2 != D1`: `domainCache.Delete(ctx, ns, kind, objectID)` thay `metaCache.Delete(id)`; interface `domainCache { Delete(ctx, namespace, kind, objectID string) error }`; PingLoopService inject `*scheduler.DomainCache`, bỏ `ServerMetaCache`

## 5. Wiring

- [x] 5.1 `internal/app/app.go` — thêm `pingsched.RegisterDomainCache` trước `RegisterServerProvider`

## 6. Tests

- [x] 6.1 `scheduler_integration_test.go` — `TestServerMetaCacheRoundTrip` bỏ field/assert `LabelSelector` + `Domain`; thêm `TestDomainCacheRoundTrip` (Get miss/hit, Set skip empty, Delete, TTL)
- [x] 6.2 `pingloop_test.go` — `mockDomainCache.Delete` đổi signature `(ctx, namespace, kind, objectID string)`; 3 stale tests (:297/375/444) bỏ `K8s: &domain.K8sRuntime{Domain:...}` khỏi server, assert xoá key theo `(ns, kind, objectID)` (hoặc chỉ count); :297 đổi assert `metaCache.Delete(7)` → `domainCache.Delete("default","Pod","web-app")`
- [x] 6.3 `url_test.go` — thêm test `ResolveURL`: Pod cache-first (hit trả domain cached không gọi k8s; miss gọi k8s + Set); Service/StatefulSet bypass cache (resolve trực tiếp, không gọi Get/Set cache); `ResolveDomain` raw

## 7. Verify

- [x] 7.1 `go build ./...`, `go vet ./...`, `golangci-lint run ./...`, `go test ./internal/infrastructure/scheduler/ ./internal/service/ ./internal/handler/` xanh
