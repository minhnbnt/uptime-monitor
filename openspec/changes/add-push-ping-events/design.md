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
- Phát hiện agent im lặng qua zset freshness dùng chung cho event push và pull
- Fix bug middleware chặn đường route public

**Non-Goals:**

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
| Tất cả accepted | 200 | `{next_time, stale_at, accepted[]}` |
| Một phần | 207 | `{next_time, stale_at, accepted[], errors[{id,error}]}` |
| Toàn lỗi / body sai | 400 | `{errors[]}` |
| Gửi sớm | 429 | `{next_time}` |
| Thiếu scope | 403 | — |

Status string chấp nhận `ON`/`OFF` (khớp `domain.StatusOn/Off`, so sánh case-sensitive cho nghiêm).

### D6 — HTTP layer: ogen generate theo pattern server-service

Ping-service giữ pattern chung của repo: OpenAPI spec per-service (`ping-service/api/spec.yaml` + `paths/ping.yaml` + `schemas/ping.yaml`) là nguồn generate qua `go tool ogen` (`.ogen.yml`, `//go:generate` trong cmd/main.go) → package `generated/api`. Handler implement interface do ogen sinh ra, wrap push service. `RunHealthCheckServer` giữ `/health` trên mux và mount ogen server tại path endpoint; auth chain `XUserIDMiddleware → RequireScope("ping")` bọc quanh route push. Root `api/` docs đang stale — bỏ qua, mỗi service tự sở hữu spec.
- *Alternative bị loại*: hand-rolled net/http handler trên mux health — lệch pattern của 5 service REST còn lại, phải tự viết decode/validate mà ogen sinh sẵn từ schema.

### D7 — Freshness zset sharded tái dụng ScoreUpdater, touch trong Record()

Zset `push:freshness:{shard}` (member = server ID, score = hạn stale UnixMilli) tái dụng sharding + hàm băm của scheduler: `ScoreUpdater` và `ZSetTaskClaimer` được tham số hóa bằng `keyFor func(shardID uint) string` (instance scheduler giữ `shardKey` cũ, instance freshness trả `"push:freshness:{n}"`). Điểm cập nhật nằm trong `RecordStatusWorker.Record(ctx, event, freshness)` TRƯỚC bước dedupe — một chỗ cho cả hai nguồn nên không thể quên: event push truyền lease `PushStaleInterval = 90s` (3 cửa sổ push 30s), ping-loop truyền `ep.Interval + PushStaleInterval`. Lease riêng theo caller là bắt buộc: nếu ép const chung thì endpoint pull có interval > 90s sẽ bị flip UNKNOWN oan giữa hai lần poll. Touch lỗi chỉ log warn, không làm rơi event (Redis chết thì stale worker cũng không claim được gì). State nằm trong Redis thay vì in-memory map — restart ping-service không mất tracking, đây chính là yêu cầu "tắt app".
- *Alternative bị loại*: store Redis viết riêng cho freshness (trùng lặp shard key/hash/pipeline đã có); const lease chung mọi nguồn (flip oan endpoint pull interval dài).

### D8 — Claimer tái dụng nguyên bản, FreshnessStore bọc cả hai

`ZSetTaskClaimer` nhận thêm `keyFor`, Lua `claimScript` + parser giữ nguyên một chữ: claim nguyên tử `ZRANGEBYSCORE score ≤ now LIMIT n WITHSCORES` + ZADD bump `now+10s` (claim lock — worker khác không lấy trùng, đúng cơ chế poll) + peek entry kế tiếp. `FreshnessStore` (repository pkg) bọc cặp updater+claimer dựng trên cùng một `keyFor`, expose `Touch/Remove/ClaimOverdue`. Stale loop mirror `ZsetLoopService`: đủ 10 entry → không sleep; else sleep min(until next, 30s). Từng entry due: record UNKNOWN (`domain.StatusUnknown`) qua đúng `Record()` (dedupe last-status tự chặn nhiễu UNKNOWN→UNKNOWN lặp) → thành công thì Remove entry (agent push lại tự tái tạo qua Touch); lỗi thì giữ entry — score đã bump nên không ai lấy trùng và tự retry ≥10s sau. OnDelete xóa luôn entry freshness để server bị xóa không phát UNKNOWN oan. Proto event dùng string tự do nên không phải regenerate; ontime lưu varchar thô không validate lúc ghi, read path `ToServerStatus` vốn coi giá trị lạ là unknown → zero change downstream.
- *Alternative bị loại*: copy claim script vào ScoreUpdater (hai bản script phải sửa song song); single key không shard (lệch mô hình sẵn có).

## Risks / Trade-offs

- [Hai nguồn chân lý đẩy nhau] Agent push OFF trong lúc pull probe thấy sống → trạng thái lật theo nguồn sau. → Chấp nhận có chủ đích theo yêu cầu "2 kiểu chạy đồng thời"; khi cần ưu tiên nguồn thì thêm policy sau.
- [`Server.Name` không unique] Không còn là vấn đề của kênh push vì event tham chiếu theo ID; trùng tên chỉ ảnh hưởng hiển thị ở tầng khác. → Không cần xử lý tại đây.
- [Header `sid` bị giả mạo nếu ai đó gọi thẳng ping-service] Mô hình tin cậy giống hệt `X-User-ID` hiện nay. → Giữ ping-service không lộ port HTTP ra ngoài mạng nội bộ compose.
- [Clock skew giữa node tính `next_time`] Lưới băm mốc tuyệt đối theo epoch; skew nhỏ hơn 30s chỉ dịch nhẹ cửa sổ. → Chấp nhận, tương tự zset pull vốn đã phụ thuộc clock.
- [Fix middleware ảnh hưởng service khác] Bug hiện tại khiến handler chạy 2 lần với header rỗng; sau fix chỉ chạy 1 lần. → Hành vi đúng, không service nào phụ thuộc hành vi sai này (test hiện có không cover nhánh đó).
- [Server bị xóa khi còn trong freshness set] OnDelete xóa entry ngay; race hi hữu giữa claim và xóa chỉ dư 1 UNKNOWN cho ID đã xóa. → Chấp nhận, ontime vốn không kiểm tra tồn tại endpoint khi lưu event.
- [UNKNOWN vào số liệu đếm] CountByStatus chỉ đếm ON/OFF nên UNKNOWN không làm lệch uptime; nếu sau này muốn hiển thị MonitorStatus "UNKNOWN" phía server-service/API (enum docs hiện ["ON","OFF"]) thì xử lý ở change riêng.

## Migration Plan

Deploy thuần additive: auth-service (thêm SID/header) → common (middleware) → server-service (RPC mới) → ping-service (route mới) → compose labels. Rollback = tắt router Traefik của ping-service; không có data migration, key Redis tự hết hạn sau 60s.

## Open Questions

Không có — các điểm mở đã chốt trong explore: interval 30s cứng, block kiểu stored-next, 207 + errors array, route `/api/v1/ping/events`.
