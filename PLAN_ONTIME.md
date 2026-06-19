# Ontime Migration Plan

## 1. git mv

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

## 2. Move OntimeCacheRepository interface

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

## 3. Fix imports trong moved files

| File | Thay đổi |
|------|----------|
| `features/server/service/ontime/batcher.go` | `service "…/internal/server/service"` → xoá (interface cùng package) |
| `features/server/service/ontime/batcher.go` | `ontimerepo "…/internal/repository/ontime"` → `"…/features/server/repository/ontime"` |
| `features/server/service/ontime/ontime_mocks_test.go` | `service "…/internal/server/service"` → xoá |

## 4. Fix imports ngoài

| File | Thay đổi |
|------|----------|
| `cmd/main.go` | `ontime "…/internal/server/service/ontime"` → `"…/features/server/service/ontime"` |
| `cmd/main.go` | `ontimerepo "…/internal/repository/ontime"` → `"…/features/server/repository/ontime"` |
| `features/server/handler/server.go` | `ontime "…/internal/server/service/ontime"` → `"…/features/server/service/ontime"` |

## 5. Xoá dir cũ

- `internal/repository/ontime/`
- `internal/server/service/ontime/`

## 6. Build + test + lint
