# Tasks: add-push-ping-events

## 1. Auth chain — session id tới downstream + fix middleware

- [x] 1.1 auth-service: thêm `SID string` vào `AccessTokenInfo`, đọc claim `sid` trong `ValidateAccessToken` (token/validator.go); cập nhật unit test
- [x] 1.2 auth-service: `ForwardAuth` set header `X-Session-ID` từ `info.SID`; cập nhật test forwardauth
- [x] 1.3 common/authclient: fix bug thiếu `return` sau `next.ServeHTTP` khi `X-User-ID` rỗng (middleware.go:47-51); thêm test case request không có headers chỉ được handle đúng 1 lần
- [x] 1.4 common/authclient: parse `X-Session-ID` trong `XUserIDMiddleware`, thêm `GetSessionID(ctx)`; unit test

## 2. server-service — RPC ResolveServers & Endpoint chia PK với Server

- [x] 2.0 Refactor: `Endpoint` bỏ `gorm.Model`, PK trùng `servers.id` (`foreignKey:ID;references:ID`), bỏ cột `server_id`; mọi nơi tạo endpoint đặt `ID = serverID` (SetCheckMethod, batch import); ping-service domain/CDC parser/client theo mô hình mới
- [x] 2.1 Thêm message + rpc vào `common/proto/server/v1/server_service.proto` (`ResolveServersRequest{user_id, repeated ids}`, `ResolveServersResponse{repeated ids}`), regenerate qua `buf generate`
- [x] 2.2 Repository: `ResolveByIDs` — `SELECT id WHERE created_by_id = ? AND id IN (?)` (Pluck); module không có pattern test repo/testcontainers nên bỏ test ở tầng này
- [x] 2.3 Service `ServerReader.ResolveServers` (TDD: passthrough + ErrInternal) + handler gRPC `ResolveServers` (map uint64↔uint); không cần wiring mới vì ServerServer/ServerRepository đã đăng ký sẵn

## 3. ping-service — push handler

- [x] 3.1 Domain/DTO: struct request item `{ID, Status}`, response `{NextTime, Accepted[], Errors[]}`; hằng số `pushInterval = 30 * time.Second`
- [x] 3.2 Rate limiter Lua: script `EVAL` nguyên tử check-and-set mốc `push:next:{sid}` (SET PX 60s) trả blocked/accepted + helper DEL giải phóng; unit test miniredis/testcontainer theo pattern test redis sẵn có (processor_test.go): gửi sớm bị chặn, hết hạn thì qua, 2 goroutine song song cùng sid chỉ 1 qua cổng
- [x] 3.3 grpcclient wrapper cho `ResolveServers` (ping-service side)
- [x] 3.4 Push service: decode → validate status ON/OFF → resolve ID → dựng accepted/errors (ID lạ/không thuộc owner → "not found") → cổng Lua check-and-set (blocked → 429 kèm next_time) → record qua `RecordStatusWorker.Record()` → 0 accepted thì DEL mốc; unit test bảng case 200/207/400/429 + "toàn lỗi không giữ mốc"
- [x] 3.5 ogen setup: `ping-service/api/{spec.yaml,paths/ping.yaml,schemas/ping.yaml}` + `.ogen.yml` + tool `ogen` trong go.mod + `//go:generate`; handler implement interface generated wrap push service; wiring `RunHealthCheckServer` giữ `/health` + mount server ogen, chain `XUserIDMiddleware → RequireScope("ping")`; test httptest cho 403 thiếu scope và 429 gửi sớm

## 4. Wiring & API docs

- [x] 4.1 compose.yml: cập nhật `authResponseHeaders` thêm `X-Session-ID`; thêm router Traefik cho ping-service rule `PathPrefix(/api/v1/ping/events)`, middlewares `cors,forward-auth`
- [x] 4.2 (đã gộp vào 3.5) — OpenAPI spec per-service chính là tài liệu API; root `api/` docs stale, bỏ qua theo quyết định của owner

## 5. Verification end-to-end

- [x] 5.1 `go test ./...` toàn repo xanh (kể cả test mới)
- [x] 5.2 Lint (`golangci-lint run`) sạch
- [x] 5.3 Smoke test thủ công theo compose: tạo ping-session token → POST batch hợp lệ (200 + next_time) → POST lại ngay (429) → batch lẫn tên lạ (207 + errors) → xác nhận event thay đổi trạng thái xuất hiện ở ontime-service

## 6. Stale detection — agent im lặng

- [x] 6.1 domain: `StatusUnknown ServerStatus = "UNKNOWN"`; hằng số `PushStaleInterval = 90 * time.Second`
- [x] 6.2 scheduler: `ScoreUpdater` + `ZSetTaskClaimer` nhận `keyFor func(shardID uint) string`; logic Unregister chuyển thành `ScoreUpdater.Remove`; integration test cập nhật ctor + case Remove
- [x] 6.3 repository `FreshnessStore` bọc updater+claimer trên cùng `keyFor` prefix `push:freshness`; API Touch/Remove/ClaimOverdue; test testcontainer theo pattern ratelimiter_test
- [x] 6.4 `RecordStatusWorker.Record(ctx, event, freshness)` touch-before-record; callers: pushevent truyền `PushStaleInterval`, pingloop truyền `ep.Interval + PushStaleInterval`; mock/test cập nhật, thêm case dedupe-skip vẫn touch
- [x] 6.5 service `StaleLoopService.Run` mirror `ZsetLoopService`: đủ batch (10) không sleep, else sleep min(until next, 30s); record UNKNOWN → thành công Remove entry, lỗi giữ entry; unit test bảng case
- [x] 6.6 wiring: runner handler per-shard + đăng ký injector + goroutine `app.RunStaleWorker` trong cmd/main.go; `OnDelete` xóa luôn entry freshness
- [x] 6.7 Verification: `go test ./...` + `golangci-lint run` xanh
