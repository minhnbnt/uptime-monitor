## Context

Repo hiện có `auth-service` tự viết (đăng ký bằng email/username/name, tự phát hành JWT HS256, forward-auth `/auth/verify` cho Traefik gắn header `X-User-ID` dạng số nguyên). Mọi service HTTP đọc `X-User-ID` qua `common/authclient` (`GetUserID() uint`), lưu `user_id` là `bigint`, truyền qua gRPC là `uint64`. Xem proposal.md - Why.

Tham chiếu: oidc-example (`/home/minhnbnt/code/go/oidc-example`) — mẫu đã chạy được với GoTrue standalone + `go-oidc` verify.

## Goals / Non-Goals

**Goals:**
- Xoá hoàn toàn `auth-service`, thay bằng GoTrue standalone.
- Mọi service HTTP verify JWT qua OIDC discovery; `user_id` là UUID string.
- notification-service lấy email qua GoTrue Admin API.

**Non-Goals:**
- Không viết lại auth server / không giữ username login.
- Không migration dữ liệu cũ (DB dev reset chấp nhận được).
- Không viết lại test hurl (xoá bỏ).
- Không giữ forward-auth / header `X-User-ID`.

## Decisions

**D1 — Deploy GoTrue standalone, 2 URL rõ ràng.**
GoTrue container (`ghcr.io/supabase/gotrue:v2.194.0`) trong `compose.yml` và `compose.infra.yml`, expose cổng riêng `9999`. Client gọi thẳng GoTrue để signup/login/refresh/logout. Traefik chỉ proxy API app (bỏ route `/api/v1/auth`). `GOTRUE_JWT_ISSUER` = URL nội bộ của GoTrue.
- *Alternative:* proxy `/api/v1/auth` qua Traefik — bị bác, muốn tách bạch 2 URL.

**D2 — Signing key ES256 dùng chung, service_role JWT cho admin API.**
Sinh ES256 keypair bằng tool keygen (kiểu `cmd/keygen` của oidc-example). GoTrue ký token bằng private key (`GOTRUE_JWT_KEYS`). Các service verify bằng **OIDC discovery** (JWKS công khai) — không cần share private key. Riêng notification-service cần service_role JWT (ký bằng chính private key đó, claim `role: service_role`) để gọi Admin API.
- *Alternative:* dùng HS256 `GOTRUE_JWT_SECRET` — kém an toàn, không chuẩn OIDC.

**D3 — Middleware OIDC dùng chung trong `common/authclient`.**
Thay `XUserIDMiddleware` bằng `AuthMiddleware` verify Bearer token qua `oidc.NewProvider(issuer)` + `IDTokenVerifier` (aud = GoTrue's aud). `GetUserID(ctx)` trả `string` (claim `sub`). Provider tự cache JWKS. Mỗi service HTTP cần config `auth.issuer` (vd `http://gotrue:9999`).
- *Alternative:* mỗi service tự viết middleware — lặp code, bị bác.

**D4 — `user_id` UUID xuyên suốt (model `uuid.UUID`, wire `string`).**
- Proto: `uint64 user_id` → `string user_id` (server_service.proto, event_service.proto) — wire format, regenerate bằng buf. Model Go dùng `github.com/google/uuid`.
- DB: cột `created_by_id` (server-service) → type `uuid`; model `CreatedByID uuid.UUID`. Bỏ cast `uint64(userID)`/`uint(req.UserId)`.
- GORM `Where("created_by_id = ?", userID)` hoạt động với `uuid.UUID` (encode về string khi query).

**D5 — DB GoTrue riêng, không đụng DB cũ.**
Thêm DB `gotrue` + user `supabase_auth_admin` + schema `auth` trong `config/init.sql` (theo `init_odic_example.sql`). `GOTRUE_MAILER_AUTOCONFIRM=true` (không cần confirm email khi dev).

**D6 — Bỏ username/name.**
Register/login chỉ email + password. user_metadata không bắt buộc. notification-service `domain.User` rút gọn còn `{ID string, Email string}`; bỏ `username`/`name` ở mọi nơi liên quan user.

**D7 — Sinh key tại build-time/dev-time.**
Tool keygen (Go) đặt trong repo (vd `tools/keygen` hoặc copy từ oidc-example), sinh `.env.gotrue-es256`. Helm secret chứa JWK + service_role token.

## Risks / Trade-offs

- **Email confirm / SMTP**: GoTrue mặc định có mailer; dùng `AUTOCONFIRM=true` tránh phụ thuộc SMTP khi dev. → Prod cần cấu hình mailer riêng.
- **UUID string làm id**: ảnh hưởng mọi layer (proto/DB/handler). → Làm theo thứ tự proto → authclient → service để compile bắt lỗi sớm.
- **Service role key nằm trên notification-service**: nếu lộ private key, attacker ký được service_role JWT. → Dùng secret riêng, cấp riêng; có thể đổi key khi cần.
- **OIDC discovery cần network tới GoTrue**: service không reach GoTrue thì mọi request 401. → Issuer là URL nội bộ (docker network / k8s service), networkpolicy cho phép egress.
- **Xoá hurl tests**: mất E2E smoke test qua Traefik. → Bù bằng `go test` từng service + smoke test thủ công signup→verify.

## Migration Plan

1. Proto đổi `user_id` → string, regenerate → các service chưa sửa vẫn compile (vì gọi qua interface).
2. `common/authclient` OIDC middleware + `GetUserID() string`.
3. Thêm GoTrue (init.sql, compose, helm, keygen, service token).
4. Sửa từng service: server → importer → ontime → notification (type + middleware + admin API).
5. Xoá `auth-service/`, forward-auth, `X-User-ID`, route auth trong api spec, CI matrix, hurl tests, `index.md`.
6. Reset DB dev, chạy compose, smoke test.

**Rollback:** giữ commit trước khi xoá `auth-service` — có thể revert; DB cũ cần restore nếu đã reset.
