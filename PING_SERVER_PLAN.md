# Server Feature Migration Plan

## Cấu trúc đích

```
internal/features/server/
├── dto/
│   ├── server.go       ← internal/server/dto/server.go
│   └── search.go       ← internal/server/dto/search.go
├── repository/
│   ├── server.go       ← internal/repository/server/server.go
│   ├── endpoint.go     ← internal/repository/server/endpoint.go
│   ├── ontime.go       ← internal/repository/server/ontime.go  (*)
│   ├── endpoint_repository_test.go
│   ├── server_repository_test.go
│   └── test_main_test.go
├── service/
│   ├── server.go       ← internal/server/service/server.go
│   ├── endpoint.go     ← internal/server/service/endpoint.go
│   └── interfaces.go   ← extracted from server/service/interfaces.go (1/2)
└── handler/
    ├── server.go       ← internal/server/handler/serverhandler.go
    ├── endpoint.go     ← internal/server/handler/endpointhandler.go
    ├── mapping.go      ← internal/server/handler/mapping.go
    └── interfaces.go   ← extracted from server/handler/interfaces.go (1/2)
```

(*) ontime.go vẫn move dù là data access cho ontime — đây là repository infrastructure, không phải business feature. Business feature ontime (OntimeService, Calculator, Batcher) giữ nguyên.

## Không move (excluded)

- `internal/server/service/import.go` + `internal/server/handler/importhandler.go`
- `internal/server/service/notification.go` + `internal/server/handler/notificationhandler.go`
- `internal/server/service/ontime/` (OntimeService, Calculator, Batcher)
- `internal/repository/ontime/` (cache)
- `internal/repository/search/` (ParadeDB)
- `internal/server/infrastructure/excel.go`
- `internal/server/dto/{import,notification,ontime}.go`

## Tách interface files

### `internal/server/service/interfaces.go` (hiện tại có 6 interfaces)

→ move 3: `ServerRepository`, `EndpointRepository`, `ServerSearchRepository`
  vào `features/server/service/interfaces.go`

→ để lại 3: `OntimeCacheRepository`, `NotificationConfigRepository`, `DigestStarter`
  trong file mới `internal/server/service/extra_interfaces.go`

### `internal/server/handler/interfaces.go` (hiện tại có 5 interfaces)

→ move 2: `ServerService`, `EndpointService`
  vào `features/server/handler/interfaces.go`

→ để lại 3: `OntimeService`, `ImportService`, `NotificationService`
  trong file mới `internal/server/handler/extra_interfaces.go`

## Files cần sửa import

### Trong moved files (self-update imports)

| File tại đích | Import cũ → mới |
|---------------|-----------------|
| `features/server/repository/endpoint.go` | `monitorrepo "...repository/monitor"` → `"...features/ping/repository"` (đã đúng) |
| `features/server/service/server.go` | `serverrepo "...repository/server"` → `"...features/server/repository"` |
| `features/server/service/endpoint.go` | `serverrepo "...repository/server"` → `"...features/server/repository"` |
| `features/server/service/interfaces.go` | `serverrepo "...repository/server"` → `"...features/server/repository"` |
| `features/server/handler/server.go` | `service "..server/service"` → `"...features/server/service"` |
| `features/server/handler/endpoint.go` | `service "..server/service"` → `"...features/server/service"` |
| `features/server/handler/server.go` | `ontime "..server/service/ontime"` → giữ nguyên (external dep) |
| `features/server/handler/server.go` | `infrastructure "..server/infrastructure"` → giữ nguyên (external dep) |

### Ngoài moved files

| File | Thay đổi |
|------|----------|
| `cmd/main.go` | Thay serverrepo import → `features/server/repository`, thay service.ServerService/EndpointService → `features/server/service`, thay handler.ServerHandler/EndpointHandler → `features/server/handler` |
| `internal/server/composite.go` | Thay `serverhandler.ServerHandler` → `featureserverhandler.ServerHandler`, thay `serverhandler.EndpointHandler` → `featureserverhandler.EndpointHandler` |
| `internal/server/service/extra_interfaces.go` (mới) | Import `features/server/repository` thay vì `repository/server` |
| `internal/server/service/ontime/batcher.go` | `serverrepo "...repository/server"` → `"...features/server/repository"` |
| `internal/server/service/ontime/ontime.go` | `serverrepo "...repository/server"` → `"...features/server/repository"` |
| `internal/server/service/notification.go` | Không đổi (dùng interface, không import repo trực tiếp) |

## Số bước

| Step | Count |
|------|-------|
| `git mv` files | ~14 |
| Tạo file interface mới | 4 (2 extract + 2 move) |
| Sửa package name | ~6 files |
| Sửa import | ~15 files |
| Xoá dir cũ | 1 (`internal/repository/server/`) |
| Build + test + lint | - |

## Lưu ý

- `features/server/handler/` và `internal/server/handler/` đều có package name `handler` → trong composite.go và main.go cần alias khác nhau
- `features/server/repository/` package name là `repository`, khác với `monitor`/`server` cũ
- `features/server/service/` và `internal/server/service/` đều có package name `service` → cần alias
