# Design: add-push-ping-events

## Context

Ping-service hiện chỉ chạy pull: endpoint vào lịch zset sharded (score = `NextExecutionTime(endpointID, interval)`, FNV-64a), worker claim → probe → `RecordStatusWorker.Record()` (dedupe last-status trong Redis, forward thay đổi qua gRPC sang ontime-service). Ping-service chưa có HTTP route public nào (chỉ `/health` nội bộ + gRPC :50053).

Auth đi qua Traefik forward-auth → auth-service `/auth/verify` → header `X-User-ID`, `X-Scopes` → `XUserIDMiddleware` ở `common/authclient`. JWT access token đã chứa claim `sid` nhưng validator (`AccessTokenInfo`) bỏ qua. Scope `ping` và API phát token (`CreatePingSession`) đã tồn tại sẵn.

Server-service sở hữu bảng `servers`/`endpoints` (1 server có đúng 1 endpoint, uniqueIndex). Proto hiện chỉ có lookup theo id / search theo query — chưa có lookup theo tên.

## Goals / Non-Goals

**Goals:**

- Kênh push REST cho agent, xác thực đầy đủ qua JWT scope `ping`
- Rate-limit theo session dùng lại đúng hàm băm của pull (`utils.NextExecutionTime`)
- Event push tái sử dụng 100% pipeline ghi nhận của pull
- Fix bug middleware chặn đường route public

**Non-Goals:**

- Phát hiện agent im lặng / staleness detection (làm change riêng khi cần)
- Interval push theo từng endpoint hoặc cấu hình được (fix cứng 30s)
- Tương tác hai chiều giữa push và lịch zset pull
- Unique constraint cho `Server.Name`

## Decisions

### D1 — Auth chain: mở rộng forward-auth thay vì tự validate JWT trong ping-service

- `token.AccessTokenInfo` thêm `SID string`; `ValidateAccessToken` đọc thêm claim `sid`.
- `ForwardAuth` set thêm header `X-Session-ID`; compose.yml cập nhật `authResponseHeaders: X-User-ID,X-Scopes,X-Session-ID` (định nghĩa middleware Traefik là global nên một dòng phủ tất cả service).
- `common/authclient`: middleware parse `X-Session-ID` + getter `GetSessionID(ctx)`; sửa bug thiếu `return` khi `X-User-ID` rỗng (middleware.go:47-51).
- *Alternative bị loại*: ping-service nhúng JWT validator — trùng lặp key/config, lệch với pattern forward-auth đang dùng bởi mọi service khác.
- Lưu ý bảo mật: `GetUserID/GetScopes/GetSessionID` tin header từ gateway — giữ nguyên mô hình tin cậy hiện có (ping-service không public trực tiếp ngoài Traefik).

### D2 — Endpoint push nằm trên mux health server sẵn có của ping-service

Mở rộng `RunHealthCheckServer` (app/http.go): route `/api/v1/ping/events` với chain `XUserIDMiddleware → RequireScope("ping") → handler`. Compose.yml thêm router Traefik rule `PathPrefix(/api/v1/ping/events)` + middlewares `cors,forward-auth`. Prefix riêng nên không đụng priority của các rule `/api/v1/servers*`.
- *Alternative bị loại*: HTTP server mới riêng — thêm port, thêm wiring, không lợi gì.

### D3 — Lookup qua gRPC RPC mới `ResolveServers(user_id, ids[])` ở server-service

Event tham chiếu server theo **ID**; `Endpoint` chia PK với `Server` (`endpoint.id == server.id`, quan hệ 1-1 chặt) nên hệ chỉ còn MỘT không gian định danh. RPC trả về các ID thuộc owner: `SELECT id WHERE created_by_id = ? AND id IN (?)`. Ping-service tự dựng mảng `errors` từ phần thiếu — báo chung một lý do "not found" cho cả ID lạ lẫn ID của người khác (không lộ sự tồn tại).
- Không cần JOIN lấy endpoint_id: ID dùng chung, event push ghi trực tiếp theo ID đã resolve.
- Điều kiện kèm theo của shared-PK: mọi đường tạo endpoint đặt `ID = serverID` tường minh (SetCheckMethod, batch import); `Endpoint` bỏ `gorm.Model` và cột `server_id`; dữ liệu dev cũ lệch ID thì re-seed.
- *Alternative bị loại*: lookup theo tên — trùng tên gây ambiguous; hai không gian ID song song (event neo theo ServerID riêng) — phải sửa schema ontime + redis namespace xuyên 3 service; ping-service truy vấn thẳng Postgres — phá ranh giới service.

### D4 — Rate limit: state nhỏ Redis per-session, grid băm deterministic

- Chốt A: interval fix cứng `pushInterval = 30s` (hằng số trong ping-service).
- Chốt B: phương án state nhỏ, atomic bằng Lua ngay từ đầu. Script `EVAL` duy nhất tại cổng request:
  ```lua
  -- KEYS[1] = push:next:{sid}
  -- ARGV[1] = now_ms, ARGV[2] = next_allowed_ms (Go tính sẵn bằng
  --          utils.NextExecutionTime(sid, 30s)), ARGV[3] = ttl_ms (60000)
  local cur = redis.call('GET', KEYS[1])
  if cur and tonumber(ARGV[1]) < tonumber(cur) then
    return {0, cur}                                  -- blocked
  end
  redis.call('SET', KEYS[1], ARGV[2], 'PX', ARGV[3]) -- accepted, đặt mốc mới
  return {1, ARGV[2]}
  ```
  Trả blocked → **429** body `{next_time}`. Hash/lưới thời gian vẫn tính ở Go (tái dùng hàm hiện có, dễ test); Lua chỉ đảm bảo check+set là một operation không thể xen kẽ.
- Batch toàn lỗi/rỗng → KHÔNG giữ mốc: nhánh accepted của script tự suy ra `fresh` — tới được lệnh `SET` nghĩa là trước đó chắc chắn không có cửa sổ active (`cur` nil hoặc đã hết hạn), tức mốc vừa đặt do chính request này tạo. 0 event accepted → `DEL` key trả lại trạng thái không-block (agent sửa gửi lại ngay). Mốc còn hạn từ request trước hợp lệ thì request đã bị chặn ở cổng, không bao giờ chạm tới `DEL`. TTL 60s = 2×interval đủ cho agent online; session chết tự dọn key.
- *Alternative bị loại*: stateless grace-window (không storage nhưng block oan agent muộn, và 2 request trong grace đều pass); GET-then-SET hai bước (khe hở race giữa 2 request song song cùng sid).

### D5 — Xử lý batch: validate-all-then-process, partial 207

Decode JSON → validate status + resolve ID (1 RPC) → **cổng Lua check-and-set mốc** (blocked → 429 `{next_time}`, dừng) → record từng item hợp lệ qua `RecordStatusWorker.Record()` (tái dùng nguyên trạng, tự gắn `Time=now`, dedupe, forward ontime) → 0 accepted thì DEL mốc vừa đặt → response:

| Trường hợp | Mã | Body |
|---|---|---|
| Tất cả accepted | 200 | `{next_time, accepted[]}` |
| Một phần | 207 | `{next_time, accepted[], errors[{name,error}]}` |
| Toàn lỗi / body sai | 400 | `{errors[]}` |
| Gửi sớm | 429 | `{next_time}` |
| Thiếu scope | 403 | — |

Status string chấp nhận `ON`/`OFF` (khớp `domain.StatusOn/Off`, so sánh case-sensitive cho nghiêm).

### D6 — API docs

Thêm path + schema vào `api/spec.yaml` theo cấu trúc paths/schemas hiện có (file mới `api/paths/ping.yaml`, `api/schemas/ping.yaml`), document đủ 200/207/400/401/403/429.

## Risks / Trade-offs

- [Hai nguồn chân lý đẩy nhau] Agent push OFF trong lúc pull probe thấy sống → trạng thái lật theo nguồn sau. → Chấp nhận có chủ đích theo yêu cầu "2 kiểu chạy đồng thời"; khi cần ưu tiên nguồn thì thêm policy sau.
- [`Server.Name` không unique] Không còn là vấn đề của kênh push vì event tham chiếu theo ID; trùng tên chỉ ảnh hưởng hiển thị ở tầng khác. → Không cần xử lý tại đây.
- [Header `sid` bị giả mạo nếu ai đó gọi thẳng ping-service] Mô hình tin cậy giống hệt `X-User-ID` hiện nay. → Giữ ping-service không lộ port HTTP ra ngoài mạng nội bộ compose.
- [Clock skew giữa node tính `next_time`] Lưới băm mốc tuyệt đối theo epoch; skew nhỏ hơn 30s chỉ dịch nhẹ cửa sổ. → Chấp nhận, tương tự zset pull vốn đã phụ thuộc clock.
- [Fix middleware ảnh hưởng service khác] Bug hiện tại khiến handler chạy 2 lần với header rỗng; sau fix chỉ chạy 1 lần. → Hành vi đúng, không service nào phụ thuộc hành vi sai này (test hiện có không cover nhánh đó).

## Migration Plan

Deploy thuần additive: auth-service (thêm SID/header) → common (middleware) → server-service (RPC mới) → ping-service (route mới) → compose labels. Rollback = tắt router Traefik của ping-service; không có data migration, key Redis tự hết hạn sau 60s.

## Open Questions

Không có — các điểm mở đã chốt trong explore: interval 30s cứng, block kiểu stored-next, 207 + errors array, route `/api/v1/ping/events`.
