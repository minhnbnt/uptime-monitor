# Ontime + Search Migration Plan

## 1. Ontime — move service + cache repo

### 1.1 git mv

```
internal/repository/ontime/ontimecache.go
  → internal/features/server/repository/ontime/ontimecache.go
    (package: ontime → giữ nguyên)

internal/server/service/ontime/ontime.go
internal/server/service/ontime/batcher.go
internal/server/service/ontime/ontimecalc.go
internal/server/service/ontime/ontime_mocks_test.go
internal/server/service/ontime/ontime_test.go
internal/server/service/ontime/ontimecalc_test.go
internal/server/service/ontime/ontime_integration_test.go
internal/server/service/ontime/ontime_integration_extra_test.go
  → internal/features/server/service/ontime/
    (package: ontime → giữ nguyên)
```

### 1.2 Move OntimeCacheRepository interface

Tạo file mới `features/server/service/ontime/interfaces.go`:

```go
package ontime

import (
    "context"

    "github.com/minhnbnt/uptime-monitor/internal/features/server/dto"
)

type OntimeCacheRepository interface {
    MGet(ctx context.Context, keys []dto.BatchGetOntimeItem) (map[dto.BatchGetOntimeItem]float64, error)
    MSet(ctx context.Context, items map[dto.BatchGetOntimeItem]float64) error
}
```

Xoá `OntimeCacheRepository` khỏi `internal/server/service/interfaces.go`.

### 1.3 Fix imports trong moved files

| File | Thay đổi |
|------|----------|
| `features/server/service/ontime/batcher.go` | `service "…/internal/server/service"` → xoá (interface cùng package) |
| `features/server/service/ontime/batcher.go` | `ontimerepo "…/internal/repository/ontime"` → `"…/features/server/repository/ontime"` |
| `features/server/service/ontime/ontime_mocks_test.go` | `service "…/internal/server/service"` → xoá |

### 1.4 Fix imports ngoài

| File | Thay đổi |
|------|----------|
| `cmd/main.go` | `ontime "…/internal/server/service/ontime"` → `"…/features/server/service/ontime"` |
| `cmd/main.go` | `ontimerepo "…/internal/repository/ontime"` → `"…/features/server/repository/ontime"` |
| `features/server/handler/server.go` | `ontime "…/internal/server/service/ontime"` → `"…/features/server/service/ontime"` |

### 1.5 Xoá dir cũ

- `internal/repository/ontime/`
- `internal/server/service/ontime/`

---

## 2. Search — move ParadeDB searcher vào server repository

### 2.1 Rename + mv

```
internal/repository/search/paradedb.go
  → internal/features/server/repository/search.go
    (package: search → repository)

internal/repository/search/search_integration_test.go
  → internal/features/server/repository/search_test.go
    (package: search → repository)
```

### 2.2 Fix imports

| File | Thay đổi |
|------|----------|
| `features/server/service/server.go` | `"…/internal/repository/search"` → xoá; dùng `serverrepo.ParadeDBSearcher` |
| `cmd/main.go` | `searchrepo "…/internal/repository/search"` → xoá; dùng `serverrepo.RegisterParadeDBSearcher` |

### 2.3 Xoá dir cũ

- `internal/repository/search/`

---

## Tổng kết số bước

| Step | Count |
|------|-------|
| `git mv` files | 10 ontime + 2 search = 12 |
| Tạo file mới | 1 (interfaces.go) |
| Sửa imports moved files | 3 |
| Sửa imports ngoài | 4 (main.go ×3, handler/server.go ×1, server.go ×1) |
| Xoá dir cũ | 4 (repository/ontime, service/ontime, repository/search, repository rỗng) |
| Build + test + lint | - |
