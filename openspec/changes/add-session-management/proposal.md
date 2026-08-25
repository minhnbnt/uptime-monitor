# Proposal: add-session-management

## Why

Session giờ được lưu trong Postgres (JTI, scopes, expires_at) nhưng người dùng không có cách nào xem mình đang có những phiên đăng nhập nào, cũng như thu hồi session cũ hoặc session agent do `CreatePingSession` sinh ra. Tính năng push-ping-events vừa hoàn tất khiến số session trên tài khoản tăng lên, nhu cầu quản lý trở nên cấp thiết.

## What Changes

- auth-service: thêm `GET /api/v1/auth/sessions` — liệt kê các session chưa hết hạn của user trong token, phân trang `page`/`per_page` kèm `meta{page, per_page, total}` theo đúng quy ước route list servers; mỗi mục gồm `id` (UUID = JTI), `scopes`, `current` (đúng nếu trùng `sid` của token gọi), `created_at`, `expires_at`.
- auth-service: thêm `DELETE /api/v1/auth/sessions/{sessionId}` — thu hồi đúng một session thuộc sở hữu của user trong token; không tìm thấy hoặc không thuộc sở hữu → 404 (không lộ sự tồn tại). Refresh token của session bị vô hiệu ngay lập tức; access token còn hiệu lực đến khi hết hạn (đã ghi nhận là giới hạn có chủ đích).
- Cả hai endpoint yêu cầu Bearer token scope `app`; token scope `ping` bị từ chối 403.
- Không đổi schema DB, không thêm cột metadata (user-agent/IP) — làm sau nếu cần.

## Capabilities

### New Capabilities
- `session-management`: Người dùng liệt kê và thu hồi các phiên đăng nhập của chính mình qua REST API của auth-service.

### Modified Capabilities

(không có — các capability hiện có không đổi requirement)

## Impact

- **auth-service**: `api/spec.yaml` + `api/paths/auth.yaml` + `api/schemas/auth.yaml` (endpoint mới), regenerate ogen (`go generate ./cmd`).
- **auth-service internal**: `dto` (SessionInfo), `service` (ListSessions, RevokeSession), `infrastructure/repository/session.go` (xóa theo user+JTI), `handler/authhandler.go` (2 handler mới + kiểm tra helper app-scope).
- **Không ảnh hưởng** service khác, compose/Traefik (route `/api/v1/auth/*` đã đi qua forward-auth sẵn).
