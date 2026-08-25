# Tasks: add-session-management

## 1. DTO & Repository

- [x] 1.1 `internal/dto`: thêm `SessionInfo{ID string (uuid), Scopes []string, Current bool, CreatedAt, ExpiresAt time.Time}`
- [x] 1.2 `infrastructure/repository/session.go`: thêm `DeleteByJTIAndUser(ctx, userID uint, jti string) (bool, error)` với `WHERE user_id = ? AND jti = ?` (parse uuid sai → trả false, giữ an toàn như `DeleteByJTI`); module không có pattern test repo/testcontainers nên bỏ test tầng này (theo quyết định của change trước)

## 2. Service (TDD)

- [x] 2.1 Mở rộng `service.SessionRepository` interface + `mockSessionRepo` (thêm `deleteByJTIAndUserFn`); viết `ListSessions(ctx, userID uint, page, perPage int) ([]dto.SessionInfo, int, error)` — lọc `ExpiresAt` quá hạn, set `current` so JTI ↔ sid từ handler, phân trang in-memory (total = số mục sau lọc, cắt slice theo page/perPage); test bảng case: lọc hết hạn, đánh dấu đúng 1 current, danh sách rỗng, trang 2/total 45 với per_page 20
- [x] 2.2 Viết `RevokeSession(ctx, userID uint, jti string) error` — found → nil, !found → `apperrors.ErrNotFound`, DB error → log + `apperrors.ErrInternal`; test cả ba nhánh

## 3. API surface & Handler

- [x] 3.1 OpenAPI: `api/schemas/auth.yaml` thêm `SessionInfo`; `api/paths/auth.yaml` thêm `list-sessions` (GET `/api/v1/auth/sessions`, BearerAuth, query `page` default 1 + `per_page` default 20/max 100 giống servers.yaml, 200 `{data}` + `$ref` `PaginationMeta` có sẵn trong common.yaml) và `delete-session` (DELETE `/api/v1/auth/sessions/{sessionId}`, format uuid, 204/404); route entries trong `api/spec.yaml`; chạy `go generate ./cmd`
- [x] 3.2 Handler: trích helper `appScopedInfo(ctx)` (tokenInfoFromContext + check scope `app`, thiếu → ErrForbidden), refactor `CreatePingSession` dùng chung; implement `ListSessions` (đọc `ListSessionsParams` do ogen sinh, map default) /`RevokeSession` map dto ↔ api type
- [x] 3.3 Test handler (pattern httptest như `forwardauth_test.go`): token scope `ping` → 403 cho cả hai endpoint; revoke id người khác → 404; happy path 200 list (kèm kiểm tra meta) + 204 delete

## 4. Verification

- [x] 4.1 `go test ./...` xanh trong auth-service (và toàn repo nếu nhanh)
- [x] 4.2 `golangci-lint run` sạch auth-service
- [x] 4.3 Smoke test theo compose: login → GET sessions thấy `current=true` → POST `/api/v1/auth/sessions/ping` tạo agent session → GET thấy mục scope `["ping"]` → DELETE agent session → 204 → GET không còn mục đó → refresh bằng refresh token đã thu hồi bị từ chối
