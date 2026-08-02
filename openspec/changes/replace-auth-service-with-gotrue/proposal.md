## Why

`auth-service` là một auth server tự viết (đăng ký/đăng nhập bằng email/username, tự phát hành JWT, forward-auth cho Traefik). Bảo trì tốn công và thiếu các tính năng chuẩn (OIDC discovery, refresh token chuẩn, admin API). Thay nó bằng GoTrue — auth server đã có sẵn, chuẩn OIDC — giúp bỏ hẳn một service phải tự viết và giảm code cần duy trì.

## What Changes

- **BREAKING**: Xoá `auth-service/` (code, generated API, Dockerfile, config).
- **BREAKING**: `user_id` chuyển từ số nguyên (`uint`/`bigint`/`uint64`) sang **UUID** (claim `sub` của GoTrue). Proto gRPC giữ `string` (wire format), model Go và DB dùng `uuid.UUID`.
- **BREAKING**: Đăng nhập bằng **email + password** thôi (bỏ username/name). Client gọi trực tiếp GoTrue tại URL riêng (vd `http://localhost:9999`) để signup/login/refresh/logout.
- **BREAKING**: Bỏ forward-auth của Traefik. Mỗi service HTTP tự verify JWT bằng `go-oidc` qua OIDC discovery.
- **BREAKING**: Bỏ header `X-User-ID`. Services lấy `user_id` từ token đã verify.
- Thêm service `gotrue` (container `ghcr.io/supabase/gotrue`) vào `compose.yml` và `compose.infra.yml` với Postgres DB riêng.
- `common/authclient`: thay `XUserIDMiddleware` bằng OIDC middleware dùng chung; `GetUserID()` trả `uuid.UUID`.
- notification-service thay private endpoint `/api/v1/auth/private/users/{id}` bằng GoTrue Admin API (`GET /admin/users/{uuid}`) với service_role JWT.
- **BREAKING**: Xoá các test hurl (`tests/*.hurl`).
- Cập nhật helm chart (bỏ app `auth-service`, thêm gotrue, bỏ forward-auth middleware) và CI/CD matrix.

## Capabilities

### New Capabilities
- `user-auth`: Xác thực người dùng dựa trên GoTrue — signup/login/refresh/logout qua GoTrue, verify JWT qua OIDC discovery ở mỗi service, và user identity là `uuid.UUID` trong context.

### Modified Capabilities
<!-- Không có spec cũ — repo chưa có openspec/specs/ -->

## Impact

- **Services bị ảnh hưởng**: `server-service`, `importer-service`, `ontime-service`, `notification-service` (đổi type `user_id`, middleware xác thực).
- **Proto**: `common/proto/server/v1/server_service.proto`, `common/proto/event/v1/event_service.proto` (`uint64 user_id` → `string user_id`, wire format), regenerate bằng buf. Model Go dùng `uuid.UUID`.
- **DB**: cột `created_by_id` (server-service) → type `uuid`; thêm DB `gotrue` + user `supabase_auth_admin` trong `config/init.sql`.
- **Infra**: `compose.yml`, `compose.infra.yml`, `helm/uptime-monitor` (values, middleware traefik, networkpolicy, cilium), secrets.
- **Dependencies**: thêm `github.com/coreos/go-oidc/v3` vào `common/authclient`; GoTrue cần private ES256 key (sinh bằng tool keygen).
- **Removed**: `auth-service/`, `tests/*.hurl`, route `/api/v1/auth/*` trong `api/spec.yaml`, config `config/auth-service.yaml`, CI matrix entries.
