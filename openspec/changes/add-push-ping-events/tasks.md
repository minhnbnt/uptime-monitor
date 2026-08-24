# Tasks: add-push-ping-events

## 1. Auth chain — session id tới downstream + fix middleware

- [x] 1.1 auth-service: thêm `SID string` vào `AccessTokenInfo`, đọc claim `sid` trong `ValidateAccessToken` (token/validator.go); cập nhật unit test
- [x] 1.2 auth-service: `ForwardAuth` set header `X-Session-ID` từ `info.SID`; cập nhật test forwardauth
- [x] 1.3 common/authclient: fix bug thiếu `return` sau `next.ServeHTTP` khi `X-User-ID` rỗng (middleware.go:47-51); thêm test case request không có headers chỉ được handle đúng 1 lần
- [x] 1.4 common/authclient: parse `X-Session-ID` trong `XUserIDMiddleware`, thêm `GetSessionID(ctx)`; unit test

## 2. server-service — RPC ResolveServers

- [ ] 2.1 Thêm message + rpc vào `common/proto/server/v1/server_service.proto` (`ResolveServersRequest{user_id, repeated names}`, `ResolvedServer{name, server_id, endpoint_id}`), regenerate code theo cách build hiện tại (Makefile)
- [ ] 2.2 Repository: query `servers JOIN endpoints WHERE created_by_id = ? AND name IN (?)` trả về name/server_id/endpoint_id; unit/testcontainer test nếu repo đã có pattern tương ứng
- [ ] 2.3 Service + handler gRPC method `ResolveServers` (mapping như các handler hiện có), wiring injector; test mapping

## 3. ping-service — push handler

- [ ] 3.1 Domain/DTO: struct request item `{Name, Status}`, response `{NextTime, Accepted[], Errors[]}`; hằng số `pushInterval = 30 * time.Second`
- [ ] 3.2 Rate limiter Lua: script `EVAL` nguyên tử check-and-set mốc `push:next:{sid}` (SET PX 60s) trả blocked/accepted + helper DEL giải phóng; unit test miniredis/testcontainer theo pattern test redis sẵn có (processor_test.go): gửi sớm bị chặn, hết hạn thì qua, 2 goroutine song song cùng sid chỉ 1 qua cổng
- [ ] 3.3 grpcclient wrapper cho `ResolveServers` (ping-service side)
- [ ] 3.4 Push service: decode → validate status ON/OFF → resolve tên → dựng accepted/errors (ambiguous khi >1 row cùng tên) → cổng Lua check-and-set (blocked → 429 kèm next_time) → record qua `RecordStatusWorker.Record()` → 0 accepted thì DEL mốc; unit test bảng case 200/207/400/429 + "toàn lỗi không giữ mốc"
- [ ] 3.5 HTTP handler + route `/api/v1/ping/events` trên mux hiện có (app/http.go) với chain `XUserIDMiddleware → RequireScope("ping")`; wiring injector; test httptest cho 403 thiếu scope và 429 gửi sớm

## 4. Wiring & API docs

- [ ] 4.1 compose.yml: cập nhật `authResponseHeaders` thêm `X-Session-ID`; thêm router Traefik cho ping-service rule `PathPrefix(/api/v1/ping/events)`, middlewares `cors,forward-auth`
- [ ] 4.2 api/spec.yaml + `api/paths/ping.yaml` + `api/schemas/ping.yaml`: document endpoint với đủ mã 200/207/400/401/403/429 theo cấu trúc file hiện có

## 5. Verification end-to-end

- [ ] 5.1 `go test ./...` toàn repo xanh (kể cả test mới)
- [ ] 5.2 Lint (`golangci-lint run`) sạch
- [ ] 5.3 Smoke test thủ công theo compose: tạo ping-session token → POST batch hợp lệ (200 + next_time) → POST lại ngay (429) → batch lẫn tên lạ (207 + errors) → xác nhận event thay đổi trạng thái xuất hiện ở ontime-service
