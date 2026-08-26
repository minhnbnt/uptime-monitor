# Proposal: add-push-ping-events

## Why

Hiện tại trạng thái server chỉ được ghi nhận qua cơ chế pull (ping-service chủ động probe endpoint theo lịch zset). Các agent chạy ngay trên server (push model) chưa có kênh nào gửi trạng thái lên hệ thống, nên không thể giám sát các máy nằm sau NAT/tường lửa mà agent outbound được. Cần thêm kênh push REST để agent chủ động báo trạng thái on/off.

## What Changes

- Thêm REST endpoint `POST /api/v1/ping/events` trên ping-service nhận batch event `[{id, status}]` (id là ID server, status ON/OFF, không có timestamp — server tự gắn thời điểm nhận).
- Xác thực đầy đủ qua JWT theo cơ chế forward-auth hiện có; yêu cầu scope `ping` trên token (đã có sẵn `CreatePingSession` phát token scope này).
- Forward thêm session id (`sid` claim trong JWT) tới downstream services qua header mới `X-Session-ID`: bổ sung vào AccessTokenInfo, ForwardAuth response headers, Traefik `authResponseHeaders`, và middleware parse ở `common/authclient`.
- Thêm gRPC `ResolveServers(user_id, ids)` vào server-service để ánh xạ server ID → `{server_id, endpoint_id}` với kiểm tra owner.
- Event push đi qua cùng đường `RecordStatusWorker.Record()` như pull (dedupe theo last-status, forward sang ontime-service) — hai kiểu ping hoạt động đồng thời, độc lập.
- Rate-limit theo session: response trả về `next_time` tính bằng hàm băm hiện có `NextExecutionTime(session_id, 30s)` (cùng cơ chế cập nhật score của poll); request đến trước `next_time` bị chặn 429.
- Batch có lỗi từng phần trả 207 kèm mảng `errors` (ID không tồn tại hoặc không thuộc owner, status sai định dạng).
- Sửa bug sẵn có trong `XUserIDMiddleware` (thiếu `return` khi `X-User-ID` rỗng khiến handler bị gọi 2 lần) — cần làm trước khi ping-service có route public đầu tiên.
- Phát hiện agent im lặng (staleness): zset freshness trong Redis — trước khi ghi event (từ push lẫn pull), cập nhật score của server = `now + khoảng tươi`; worker quét các entry quá hạn, bump score `now+10s` chống worker khác lấy trùng, record 1 event UNKNOWN rồi xóa entry (record lỗi thì giữ lại, tự retry sau ≥10s). Agent tắt app nhưng server vẫn được pull probe sẽ không bị đánh unknown oan.
- Ngoài scope có ý định: interval push theo từng endpoint, tương tác với lịch zset của pull.

## Capabilities

### New Capabilities

- `push-ping-events`: Kênh push event trạng thái server từ agent vào ping-service qua REST — xác thực JWT scope `ping`, rate-limit theo session dựa trên băm `session_id`, ghi nhận event dùng chung pipeline với pull.

### Modified Capabilities

<!-- Không có spec hiện hữu nào trong openspec/specs/ -->

## Impact

- **ping-service**: handler HTTP mới (`/api/v1/ping/events`) trên mux health server sẵn có; service layer mới cho push; wiring injector; config Traefik labels trong compose.yml; zset freshness sharded `push:freshness:{shard}` tái dụng ScoreUpdater/ZSetTaskClaimer qua tham số `keyFor`, stale worker loop, status `UNKNOWN`.
- **server-service**: proto mới (`ResolveServers`) + repository/service/handler method.
- **common/proto**: file `.proto` mới cho ResolveServers, regenerate code.
- **common/authclient**: parse `X-Session-ID`, getter `GetSessionID`, fix bug thiếu `return`.
- **auth-service**: `AccessTokenInfo` thêm trường `SID`; `ForwardAuth` set header `X-Session-ID`.
- **API docs**: cập nhật `api/spec.yaml` + schema/path mới.
- **Redis**: key state nhỏ `push:next:{sid}` (TTL ~60s).
- Không đổi schema database, không đổi hành vi pipeline pull hiện có.
