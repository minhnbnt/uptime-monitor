## 1. Proto & shared identity type

- [x] 1.1 Đổi `uint64 user_id` → `string user_id` trong `common/proto/server/v1/server_service.proto` (các message ListServersRequest, GetServerRequest, SearchServersRequest, ServerWithEndpointInput, CountServersByStatusRequest)
- [x] 1.2 Đổi `uint64 user_id` → `string user_id` trong `common/proto/event/v1/event_service.proto` (CountByStatusRequest, GetServersOntimeRequest)
- [x] 1.3 Chạy `make gen-proto` (buf generate) để regenerate Go code
- [x] 1.4 `go build ./...` trong common/proto để xác nhận generate thành công

## 2. common/authclient — OIDC middleware

- [x] 2.1 Thêm dependency `github.com/coreos/go-oidc/v3` vào `common/authclient/go.mod`
- [x] 2.2 Thay `XUserIDMiddleware` bằng middleware OIDC: nhận issuer, tạo `oidc.Provider` + `IDTokenVerifier` (ClientID = GoTrue aud), verify Bearer token, lưu claim `sub` (parse thành `uuid.UUID`) vào context; trả 401 khi thiếu/sai token
- [x] 2.3 Đổi `GetUserID(ctx) uint` → `GetUserID(ctx) uuid.UUID` (parse sub)
- [x] 2.4 Bỏ parsing header `X-User-ID`; thêm hàm cấu hình issuer (vd `NewAuthMiddleware(issuer string, log *slog.Logger)`)
## 3. GoTrue infra (DB + compose + helm + keygen)

- [x] 3.1 Thêm vào `config/init.sql`: user `supabase_admin`, `supabase_auth_admin`, DB `gotrue`, schema `auth` (theo `init_odic_example.sql` của oidc-example)
- [x] 3.2 Thêm service `gotrue` (image `ghcr.io/supabase/gotrue:v2.194.0`, port 9999, env theo gotrue.env) vào `compose.yml` và `compose.infra.yml`
- [x] 3.3 Tạo `config/gotrue.env` (hoặc tương đương): `GOTRUE_JWT_KEYS`, `GOTRUE_JWT_ISSUER`, `GOTRUE_SITE_URL`, `GOTRUE_MAILER_AUTOCONFIRM=true`, `DATABASE_URL` trỏ DB gotrue
- [x] 3.4 Thêm tool sinh ES256 keypair vào repo (copy/cải tiến `cmd/keygen` từ oidc-example), sinh `.env.gotrue-es256` mẫu
- [x] 3.5 Sinh service_role JWT mẫu (ký bằng private key, claim `role: service_role`) để dùng cho notification-service
- [x] 3.6 Cập nhật helm: thêm deployment/service/configmap cho gotrue (hoặc xác nhận gotrue host ngoài cluster qua infra.yml + ExternalName), bỏ app `auth-service` khỏi `values.yaml`/`values.example.yaml`
- [x] 3.7 Cập nhật helm secrets: bỏ `auth-service`, thêm GoTrue env + service token
- [x] 3.8 Cập nhật helm traefik: xoá middleware `forward-auth` (`middleware.yaml`) và tham chiếu trong `ingressroute.yaml`; thêm rule proxy tới gotrue nếu cần
- [x] 3.9 Cập nhật helm networkpolicy + cilium l7: bỏ reference `auth-service`, thêm egress cho services tới gotrue

## 4. server-service — UUID identity

- [x] 4.1 Đổi `CreatedByID uint` → `CreatedByID uuid.UUID` (`gorm:"type:uuid"`) trong `internal/domain/server.go` (và các struct liên quan); thêm dependency `github.com/google/uuid`
- [x] 4.2 Sửa handler `internal/handler/server.go` + `k8s_object.go`: `userID := authclient.GetUserID(ctx)` (giờ là uuid.UUID), bỏ cast uint; khi gọi gRPC dùng `userID.String()`
- [x] 4.3 Sửa `internal/app/http.go`: dùng middleware OIDC mới thay `XUserIDMiddleware`
- [x] 4.4 Kiểm tra repository `Where("created_by_id = ?", ...)` vẫn đúng với `uuid.UUID`; cập nhật nếu cần
- [x] 4.5 Sửa các unit test liên quan `user_id` type

## 5. importer-service

- [x] 5.1 Thay middleware xác thực (OIDC) trong `internal/app/server.go`
- [x] 5.2 Bỏ cast `uint64(userID)` trong `internal/service/import.go` — dùng `userID.String()` khi truyền vào proto
- [x] 5.3 Sửa test liên quan nếu có

## 6. ontime-service

- [x] 6.1 Thay middleware xác thực (OIDC) trong `internal/app/http.go`
- [x] 6.2 Bỏ cast `uint(req.UserId)` trong `internal/handler/ontime_grpc.go`, `internal/handler/statuses.go`, `internal/service/ontime.go`, `internal/infrastructure/serverclient/client.go` — parse string thành `uuid.UUID`, giữ `uuid.UUID` trong model/service
- [x] 6.3 Sửa test liên quan nếu có

## 7. notification-service — OIDC middleware + Admin API

- [x] 7.{i} Thay middleware xác thực (OIDC) trong `internal/app/server.go`
- [x] 7.{i} Rút gọn `internal/domain/user.go`: `User{ID uuid.UUID, Email string}` (bỏ Username, Name)
- [x] 7.{i} Sửa `internal/infrastructure/userclient/client.go`: `FindByID(id uuid.UUID)` → gọi `GET {gotrueURL}/admin/users/{id}` với `Authorization: Bearer <service_role JWT>`; parse email
- [x] 7.{i} Thêm config mới (gotrue URL + service_role token) vào `internal/config` và `config/notification-service.yml`; bỏ `auth_service` cũ
- [x] 7.{i} Sửa `internal/service/digest.go`: `FindByID(ctx, userID)` với userID `uuid.UUID`; bỏ `slog.Uint64`/cast
- [x] 7.{i} Bỏ `internal/infrastructure/serverclient` cast `uint64(createdByID)` → dùng `uuid.UUID`
- [x] 7.7 Sửa test liên quan nếu có

## 8. Xoá auth-service & dọn dẹp

- [x] 8.{i} Xoá thư mục `auth-service/` (code, generated, Dockerfile, api, config)
- [x] 8.{i} Xoá service `auth-service` + labels forward-auth trong `compose.yml`; bỏ `config/auth-service.yaml`
- [x] 8.{i} Xoá `auth-service` khỏi matrix trong `.github/workflows/ci.yml` và `.github/workflows/publish.yml`
- [x] 8.{i} Xoá route `/api/v1/auth/*` và schema auth khỏi `api/spec.yaml`, `api/paths/auth.yaml`, `api/schemas/auth.yaml`
- [x] 8.{i} Xoá `tests/auth.flow.hurl`, `tests/e2e.flow.hurl`, `tests/server.flow.hurl`
- [x] 8.{i} Cập nhật `index.md`: mô tả kiến trúc auth mới (GoTrue, 2 URL, user_id UUID, bỏ forward-auth/X-User-ID)
- [x] 8.{i} Xoá `config/auth-service.yaml` và mọi tham chiếu `auth-service` còn lại trong repo (helm, compose, config)

## 9. Verify

- [x] 9.{i} `go build ./...` và `go test ./... -race` cho từng service còn lại (server, importer, ontime, notification, ping)
- [x] 9.{i} `cd common/proto && buf generate` chạy sạch
- [x] 9.{i} Chạy `pnpx @fission-ai/openspec@latest validate` cho change này
- [x] 9.4 Reset DB dev, chạy compose, smoke test: signup → login → gọi endpoint có xác thực → refresh → logout
