## Context

Ping-service hiện cache server metadata trong Redis hash `scheduler:meta:<id>` (key theo server id), trong đó có cả `label_selector` và `domain` đã resolve (endpointcache.go:120-125, mapToServer:194-199). Domain được resolve **eager** trong `ServerProvider.fillK8sFields` khi cache miss (endpointprovider.go:117-128), mỗi server 1 bản domain riêng dù trỏ cùng object.

Vấn đề: N server trỏ cùng 1 k8s object → N lần resolve domain (Pod = 1 k8s `Get` lấy IP), dễ dính rate-limit k8s API. Khi meta cache bị invalidate (CDC `servers` update/delete ở serverHandler.go:111, hoặc stale domain ở pingloop.go:136 xoá cả hash) → lần GetBatch kế phải refetch gRPC + re-resolve domain.

Nguồn dữ liệu: gRPC `GetEndpoints` trả identity + http config (`ep.GetHttpDnsConfig()`, endpoint_client.go:57); domain resolve từ `k8sclient.ResolveDomainName` (Service/StatefulSet = chuỗi compute, Pod = `Get` IP) — phụ thuộc object, không phụ thuộc server row.

## Goals / Non-Goals

**Goals:**
- Tách Pod IP đã resolve ra cache riêng key theo object identity `(namespace, kind, object_id)` → dùng chung giữa các server trỏ cùng Pod, N server → 1 lần k8s resolve/TTL.
- Chỉ cache Pod (kind duy nhất phải trả k8s API `Get` lấy IP); Service/StatefulSet là chuỗi compute nên resolve trực tiếp, không cache.
- Resolve domain **lazy** tại điểm dựng URL (ResolveURL) thay vì eager tại GetBatch.
- Stale domain chỉ invalidate key domain, không đụng meta cache → không refetch gRPC, http config/identity sống sót.
- Bỏ caching `label_selector` (không đáng: API chính `List Pods` vẫn chạy mỗi check).

**Non-Goals:**
- Không xoá `K8sRuntime`/`dto.K8sObjectCheckParams.K8s` (để diff gọn; chúng vô hại vì nil-safe) — làm sau nếu muốn.
- Không đụng proto, server-service, event-service, Debezium config, CDC `server_http_configs` (change `cdc-http-config-topic` để riêng).

## Decisions

### D1. Domain cache riêng, key theo object identity — chỉ Pod
`internal/infrastructure/scheduler/domaincache.go` — Redis string:
- Key: `scheduler:domain:<namespace>:<kind>:<object_id>` (thực tế chỉ Pod ghi; giữ `kind` trong key cho tự mô tả)
- `Get(ctx, namespace, kind, objectID) (string, bool, error)` — `bool` = hit/miss
- `Set(ctx, namespace, kind, objectID, domain)` — skip nếu `domain == ""`
- `Delete(ctx, namespace, kind, objectID)`
- TTL: 1h (cùng `metaCacheTTL`, đủ vì stale-detection đã tự sửa khi IP đổi)
- `RegisterDomainCache` DI, dùng `RedisClientWrapper` như `ServerMetaCache`

**Rationale**: chỉ Pod trả k8s API khi resolve (urlResolver.go:52-63 `resolvePodURL` = `Get` lấy IP); Service (`name.namespace.svc.cluster.local`) và StatefulSet (`name-0.name.namespace.svc.cluster.local`) là `Sprintf` thuần (urlResolver.go:30-41) — cache chúng chỉ thêm Redis round-trip vô ích. Key theo object identity để N server trỏ cùng Pod dùng chung 1 entry, và TTL đóng vai trò rate-limiter tự nhiên: mỗi Pod ≤ 1 lần k8s `Get`/TTL.

### D2. ResolveURL = cached (chỉ Pod), ResolveDomain = raw
- `URLResolverService.ResolveURL`: `resolveCached` — nếu `params.Kind != "Pod"` gọi thẳng `k8sClient.ResolveDomainName` (Service/STS compute, không đụng cache); nếu Pod → `DomainCache.Get` → hit buildURL, miss → `k8sClient.ResolveDomainName` → `Set` (best-effort, lỗi Set bỏ qua) → buildURL.
- `URLResolverService.ResolveDomain`: giữ **raw** (gọi thẳng `k8sClient.ResolveDomainName`) — phục vụ so sánh fresh ở stale path.
- gRPC `PingServer` (ping_server.go:94) dùng chung `URLResolverService` → được cache miễn phí cho Pod.

**Alternative considered**: cache cả Service/StatefulSet, hoặc cache cả 2 phương thức + cờ bypass / trick delete-then-resolve. Rejected — Service/STS cache không mang lại lợi ích (không chạm API); raw `ResolveDomain` không cần đổi interface; delete-then-resolve sẽ xoá cache ngay cả khi IP không đổi (mỗi ping fail đều re-resolve, phí).

### D3. Meta cache thu hẹp scope
- `ServerMetaCache.SetMulti` (endpointcache.go:120-125): bỏ `label_selector` + `domain`; chỉ write identity + `interval_ns`/`timeout_ns` + http_config marker/fields (`httpConfigHashValues`).
- `mapToServer` (194-199): bỏ reconstruct `K8sRuntime` → `sv.K8s` luôn nil.
- `ServerProvider.fillK8sFields` xoá hẳn; `ServerProvider` bỏ dependency `k8sClient`; GetBatch = MGet → miss → gRPC → SetMulti.

### D4. Stale invalidation nhắm vào domain cache
`checkHTTPDNS` (pingloop.go:86-145):
- D1 = `url.Hostname()` (host từ ResolveURL) thay vì `k8sParams.K8s.Domain`
- Bỏ check `K8s == nil`/`Domain == ""` (dòng 115) — chỉ còn `Kind != "Pod"`
- Bỏ hack `freshParams`/`freshK8s` (124-126) — `ResolveDomain(ctx, k8sParams)` trực tiếp (raw = fresh)
- `D2 != D1` → `domainCache.Delete(ns, kind, objectID)` thay vì `metaCache.Delete(id)` → `errStaleDomain`
- `PingLoopService`: bỏ inject `ServerMetaCache`, inject `DomainCache`; interface `domainCache { Delete(ctx, namespace, kind, objectID string) error }`

**Hệ quả**: meta cache sống sót → lần GetBatch kế là cache HIT → không refetch gRPC, http config còn nguyên. Trường hợp "IP không đổi" (pingloop_test:444) không xoá gì — giữ nguyên.

### D5. Label selector không cache
`workloadChecker` đã nil-safe và tự resolve selector live khi `params.K8s == nil` (workloadChecker.go:55-71). Bỏ caching chỉ là xoá write/read trong cache; hành vi check không đổi (vẫn `List Pods` mỗi lần).

## Risks / Trade-offs

- [Pod IP volatile] Domain Pod đổi khi pod restart → cached domain có thể stale ≤ TTL → Mitigation: stale-detection bắt khi ping fail (fresh resolve + so sánh), tự heal trong ~10s (claim lock).
- [gRPC Ping handler dùng domain cached] `PingServer` trước đây resolve fresh mỗi call cho Pod, giờ có thể trả domain stale ≤ TTL → Mitigation: một-shot check chấp nhận sai lệch nhỏ; vòng lặp scheduling vẫn đúng nhờ stale-detection.
- [Share key bị invalidate] Một server phát hiện stale xoá key dùng chung → server khác cùng object cũng re-resolve → Mitigation: đúng và an toàn, không sai lệch.
- [Cache poisoning] k8s resolve trả domain sai và bị cache → Mitigation: `Set` skip domain rỗng; TTL 1h giới hạn; Pod IP chỉ thay đổi theo thực tế.
- [K8sRuntime vestigial] `K8sRuntime` + `params.K8s` luôn nil sau thay đổi → Mitigation: giữ để diff gọn, nil-safe sẵn; ghi nhận cleanup riêng.

## Migration Plan

- Internal refactor, không cần migrate dữ liệu: Redis key domain mới tự sinh; key cũ (domain trong meta hash) hết TTL tự sạch.
- Rollback: revert working tree; cache tự hội tụ lại qua gRPC refresh/TTL.

## Open Questions

- Có cần TTL domain ngắn hơn meta (ví dụ 5-10 phút) để bắt IP đổi nhanh hơn? Hiện dùng chung 1h vì stale-detection đã lo; để mặc định.
- Cleanup `K8sRuntime`/`dto.K8sObjectCheckParams.K8s` (xoá struct + shortcut trong urlResolver.go/workloadChecker.go) — làm trong change riêng.
