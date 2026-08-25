# Design: add-session-management

## Context

Session là row `domain.Session` (`sessions`: `user_id`, `jti` UUID uniqueIndex, `scopes`, `expires_at`, gorm.Model). Refresh đã tra cứu DB theo JTI (`ValidateRefreshToken`) nên xóa row = chết refresh ngay lập tức; access token chỉ được kiểm tra chữ ký/claims (`ValidateAccessToken`, cả nhánh forward-auth lẫn ogen `HandleBearerAuth`). Repo có sẵn `FindByUser` và `DeleteByJTI` (bản này không kèm điều kiện user). Handler lấy thông tin token qua `tokenInfoFromContext`; pattern check scope đang viết inline tại `CreatePingSession` (`slices.Contains(info.Scopes, "app")`). Spec per-service là nguồn sinh ogen qua `//go:generate` có sẵn trong `cmd/main.go`.

## Goals / Non-Goals

**Goals:**

- List + revoke session của chính mình qua 2 endpoint REST, đúng pattern ogen/handler/service/repository hiện có
- Revoke có hiệu lực tức thì với refresh token (tận dụng tra cứu DB sẵn có của refresh)
- Không lộ sự tồn tại của session người khác

**Non-Goals:**

- Thu hồi access token tức thì (phải tra cứu DB mỗi request ở forward-auth hot path) — access token sống đến hạn, TTL ngắn
- Bulk revoke "tất cả thiết bị khác", metadata session (user-agent/IP), last-used-at
- Cleanup job dọn row hết hạn
- UI frontend (làm change/frontend riêng)

## Decisions

### D1 — Định danh public của session = JTI (UUID)

Path param `{sessionId}` là UUID JTI, không phải PK số: `sid` vốn là định danh tự nhiên đã nằm trong JWT và header `X-Session-ID`, dùng chung một không gian giúp client đánh dấu `current` mà không cần map thêm. Khai báo `format: uuid` để ogen tự trả 400 cho id sai định dạng trước khi chạm service.
- *Alternative bị loại*: PK `uint` của gorm.Model — hai không gian định danh song song, client phải giữ mapping.

### D2 — Xóa theo cặp (user_id, jti), tách method riêng

Thêm `SessionRepository.DeleteByJTIAndUser(ctx, userID, jti string) (bool, error)` với `WHERE user_id = ? AND jti = ?`, trả về `found`. Không đụng `DeleteByJTI` (Logout đang dùng). Service map `found == false` → `apperrors.ErrNotFound`.
- *Alternative bị loại*: thêm tham số userID vào `DeleteByJTI` — đổi chữ ký làm ồn mọi call site cũ vì một nhu cầu mới.

### D3 — Check scope gom thành helper dùng chung

Trích helper nhỏ trong package handler (ví dụ `appScopedInfo(ctx) (*token.AccessTokenInfo, error)`: lấy tokenInfoFromContext → thiếu scope `app` trả `ErrForbidden`) và dùng lại ở `CreatePingSession` + 2 handler mới. Một chỗ sửa khi thêm scope logic.
- *Alternative bị loại*: copy `slices.Contains` sang mỗi handler — ba bản sao giống hệt nhau.

### D4 — Lọc hết hạn ở service, phân trang in-memory

`ListSessions(ctx, userID uint, page, perPage int) ([]dto.SessionInfo, int, error)` gọi `FindByUser`, bỏ các mục `ExpiresAt.Before(now)`, `total = len(kết quả sau lọc)`, rồi cắt slice theo `(page-1)*perPage : (page-1)*perPage+perPage` trên mảng đã sort created_at DESC (repo đã order). Đồng thời set `current` bằng cách so JTI với sid lấy từ context handler. Repo giữ nguyên tính dumb.
- *Alternative bị loại*: LIMIT/OFFSET + COUNT hai query trong repo — session mỗi user chỉ chục cái (mỗi login/agent token một row), in-memory rẻ hơn và tránh thêm method đếm.
- *Alternative bị loại*: filter bằng WHERE trong repo — đẩy policy thời gian vào tầng data và chặn khả năng hiển thị trạng thái hết hạn sau này nếu cần.

### D5 — API surface theo pattern auth-service hiện có

- `GET /api/v1/auth/sessions?page=&per_page=` → `200 {"data": [SessionInfo], "meta": PaginationMeta}` (BearerAuth); tham số mặc định 1/20, `per_page` tối đa 100 — trùng khít route list servers (`api/paths/servers.yaml`) kể cả giá trị default
- `DELETE /api/v1/auth/sessions/{sessionId}` → `204` | `404` ErrorResponse (BearerAuth)

`PaginationMeta` đã tồn tại sẵn trong `api/schemas/common.yaml` của auth-service — `$ref` thẳng, không khai báo mới. ogen sinh `ListSessionsParams` với pointer + default nên handler chỉ map qua service.

```yaml
SessionInfo:
  required: [id, scopes, current, created_at, expires_at]
  # id: string format uuid; scopes: []string; current: boolean
  # created_at/expires_at: string format date-time
```

Thêm vào `api/paths/auth.yaml` + route entries trong `api/spec.yaml` + schema trong `api/schemas/auth.yaml`, chạy lại `go generate ./cmd` để sinh handler interface, implement `ListSessions`/`RevokeSession` trên `AuthHandler`.

## Risks / Trade-offs

- [Access token vẫn dùng được sau revoke] → Chấp nhận có chủ đích (TTL ngắn); tài liệu hóa trong spec. Khi cần tức thiệt: thêm tra cứu `GetByJTI` vào forward-auth và ăn chi phí DB mỗi request.
- [Row session hết hạn tồn đọng] → Chỉ là dữ liệu chết, không ảnh hưởng bảo mật (refresh đã chặn theo expires_at); dọn bằng job khi cần.
- [Race 2 tab revoke cùng session] → Tab sau nhận 404; hành vi chấp nhận được, không cần lock.
- [Ping session xuất hiện trong danh sách] → Đúng ý thiết kế: user thấy và thu hồi được token agent từ UI quản lý session.

## Migration Plan

Deploy thuần additive (spec + code auth-service), không đổi DB, không đổi contract cũ. Rollback = revert deploy auth-service.

## Open Questions

Không có.
