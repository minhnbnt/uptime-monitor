## Why

Khi nhiều server trỏ cùng 1 Pod, Pod IP bị resolve lặp lại N lần (giá trị giống hệt), mỗi lần là 1 k8s API call (`Get` lấy IP) — dễ dính rate-limit của k8s API và làm ping chậm. Hiện tại domain nằm chung trong meta cache `scheduler:meta:<id>` (key theo server id) nên không dùng chung được giữa các server; meta cache bị invalidate (CDC `servers` update/delete, hoặc phát hiện stale domain) thì phải refetch gRPC + re-resolve domain từ đầu. Service/StatefulSet không bị vấn đề này (domain là chuỗi DNS compute, không gọi k8s API).

## What Changes

- **Mới**: `DomainCache` tách riêng, key theo object identity `(namespace, kind, object_id)`, lưu Pod IP đã resolve (Redis string, TTL) — dùng chung cho mọi server trỏ cùng Pod: N server → 1 lần k8s resolve / TTL.
- **Chỉ cache Pod**: Service/StatefulSet resolve trực tiếp bằng chuỗi compute tại lúc dựng URL, không đụng cache.
- `ServerMetaCache` thu hẹp scope: chỉ còn identity + `interval`/`timeout` + http config. Bỏ `label_selector` và `domain` khỏi meta hash (`SetMulti`/`mapToServer`).
- `URLResolverService.ResolveURL` trở thành điểm đọc cache cho Pod (cache-first → miss → `k8sClient.ResolveDomainName` → write-through); Service/StatefulSet gọi thẳng resolve. `ResolveDomain` giữ raw (fresh) phục vụ so sánh ở stale path. gRPC `PingServer` dùng chung `URLResolverService` nên được hưởng cache miễn phí.
- `ServerProvider.GetBatch` bỏ `fillK8sFields` và dependency `k8sClient` (chỉ cache + gRPC).
- Stale-domain invalidation đổi target: xoá key domain theo `(namespace, kind, object_id)` thay vì xoá cả meta hash → không refetch gRPC, http config + identity sống sót.
- Bỏ caching `label_selector` (selector chỉ là tiền đề cho `List Pods` — API call chính vẫn chạy mỗi lần check, cache không đáng; workloadChecker đã tự resolve live khi selector rỗng).

## Capabilities

### New Capabilities

- `endpoint-domain-cache`: Cache Pod IP đã resolve của endpoint, key theo k8s object identity `(namespace, kind, object_id)`, được đọc lazily lúc resolve HTTP URL và invalidate theo object khi phát hiện stale — tách khỏi meta cache để dùng chung giữa các server trỏ cùng Pod và không bị cuốn theo vòng đời meta cache. Service/StatefulSet resolve trực tiếp (chuỗi compute), không cache.

### Modified Capabilities

None — chưa có main spec nào tồn tại (`openspec/specs/` trống).

## Impact

- `ping-service`: `internal/infrastructure/scheduler/domaincache.go` (mới), `internal/infrastructure/scheduler/endpointcache.go`, `internal/infrastructure/scheduler/endpointprovider.go`, `internal/service/url.go`, `internal/service/pingloop.go`, `internal/app/app.go`, `internal/service/events/serverHandler.go` (nếu cần đồng bộ xoá meta).
- Test: `scheduler_integration_test.go` (round-trip meta + round-trip domain cache), `pingloop_test.go` (mock invalidation target), `url_test.go` (nếu thêm unit test resolve cached).
- Không đổi proto, server-service, event-service, Debezium config.
