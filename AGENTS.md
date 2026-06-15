# AGENTS.md — Uptime Monitor

## Quick start

```sh
make dev              # go tool air hot-reload (build & run in Docker)
make build            # production build (entrypoint: ./app)
make generate         # ogen from api/spec.yaml via .ogen.yml
make test             # run all unit tests (skip integration tests via -short)
make test-cover       # unit tests with coverage
make test-cover-html  # open coverage report in browser
make test-integration # run all tests including integration (needs Docker)
go build ./...        # compile check
make format           # auto-fix with golangci-lint (gofmt, gci, govet, ...)
golangci-lint run ./...  # lint check only
hurl --test --variable base_url=http://localhost:8080 tests/*.hurl
```

> **Lint before commit**: always run `golangci-lint run ./...` before committing. It enforces import order (gci), `interface{}`→`any` (gofmt rewrite), bodyclose, noctx, unused, errcheck, ineffassign, staticcheck, and revive lints.

## Architecture

- **DI**: `samber/do/v2` — every component has a `Register*` function called in `main.go` (registration order matters: config → repo → service → handler).
- **Handler → Service → Repository → DB**: layers use consumer-package interfaces (defined where used, not where implemented).
- **API**: OpenAPI spec at `api/spec.yaml`. Run `make generate` to regenerate `generated/api/gen.go` via ogen.
- **CompositeHandler** (`internal/server/composite.go`) embeds `ServerHandler` and `EndpointHandler` to satisfy the generated `ServerInterface`.
- **Router**: Gin, using ogen's `RegisterHandlers`.

## Package layout

```
api/spec.yaml               ← source of truth for routes
generated/api/gen.go        ← auto-generated, do not edit
internal/config/            ← GORM, Temporal, Zap setup (env-driven)
internal/monitor/           ← Temporal workflows, ping, record status
  infrastructure/           ← PingWorker, RecordStatusWorker
    repository/             ← ServerEvent (GORM), RedisServerEvent (Redis)
  services/                 ← PingService (Temporal workflow)
internal/server/
  composite.go              ← unifies handler structs into ServerInterface
  domain/                   ← Server, Endpoint, ServerEvent, User (GORM models)
  dto/                      ← request/response DTOs with validate tags
  handler/                  ← Gin handlers, RequestValidator, helper funcs
  infrastructure/           ← Argon2PasswordEncoder, JwtParser
    repository/             ← GORM and Redis repos
  service/                  ← business logic, OntimeCalculator
internal/utils/             ← TruncateDay, Last30Days, PageValidator
```

## Key details

- **OpenAPI required fields** drive generated types. Marking a field `required` in `api/spec.yaml` makes it a value type (not `*T`) in generated code. Always update the spec then regenerate.
- **Validation**: `RequestValidator` (DI-injected) wraps `go-playground/validator/v10`. Validated at the DTO layer in handlers using `validate` tags.
- **Temporal**: `TemporalSchedulerRepository` reads `TEMPORAL_TASK_QUEUE` and `TEMPORAL_WORKFLOW_NAME` from env. Temporal server runs via compose.
- **Environment** (compose.yml): `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, `TEMPORAL_TASK_QUEUE`, `TEMPORAL_WORKFLOW_NAME`.
- **Running integration tests**: `podman compose up -d --build` (or `docker compose up -d --build`) rebuilds and starts postgres + temporal + app. Hurl tests call the running instance on `:8080`. Use whichever compose tool is available — never use the hyphenated `podman-compose` or `docker-compose`.
- **Database**: PostgreSQL with GORM auto-migrate (`Server`, `Endpoint` models). No manual migrations. Endpoint upsert uses `ON CONFLICT (server_id) DO UPDATE`.
- **Redis cleanup**: Deleting a server removes associated Redis keys (status, ZSet, metadata hash) and unregisters from the scheduler (Temporal/ZSet).
- **Import order** (enforced by gci): std → third-party → `github.com/minhnbnt/uptime-monitor/`.
- **`interface{}` → `any`** enforced by gofmt rewrite rule in `.golangci.yml`.
- **Before committing**: run `golangci-lint run ./...` to ensure code quality.

## Error handling conventions

### Sentinel errors

All sentinel errors live in `internal/errors/errors.go` (package `apperrors`). The package is always imported with `apperrors` alias:

```go
import apperrors "github.com/minhnbnt/uptime-monitor/internal/errors"
```

### Repository layer

Check DB-level errors first and wrap with sentinel before returning:

```go
func (sr *ServerRepository) GetByID(ctx context.Context, id uint) (*domain.Server, error) {
    server, err := gorm.G[domain.Server](sr.db).Where("id = ?", id).First(ctx)
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, fmt.Errorf("server %d: %w", id, apperrors.ErrNotFound)
    }
    if err != nil {
        return nil, fmt.Errorf("failed to get server: %w", err)
    }
    return &server, nil
}
```

### Service layer

Log the full error detail, then check sentinels with `errors.Is` — always **before** the generic `if err != nil`:

```go
server, err := ss.serverRepository.GetByID(ctx, id)
if errors.Is(err, apperrors.ErrNotFound) {
    return nil, apperrors.ErrNotFound
}
if err != nil {
	ss.logger.Error("failed to get server", logger.Error(err))
    return nil, apperrors.ErrInternal
}
```

Rules:
- Do **not** nest `if errors.Is(...)` inside `if err != nil` — keep them as sibling blocks at the same indentation.
- Log actual error with `logger.Error("msg", logger.Error(err))` — the message returned to the client is the sentinel's own message, so no `fmt.Errorf` wrapping needed.
- `ListServers` / `CreateServer` / methods that shouldn't return 404: just return `apperrors.ErrInternal`.

### Handler layer

Use the exported `handler.ToAPIError(err)` for automatic status mapping:

```go
func ToAPIError(err error) *api.ErrorResponseStatusCode {
    if errors.Is(err, apperrors.ErrNotFound) {
        return &api.ErrorResponseStatusCode{
            StatusCode: http.StatusNotFound,
            Response:   errResponse("NOT_FOUND", err.Error()),
        }
    }
    return &api.ErrorResponseStatusCode{
        StatusCode: http.StatusInternalServerError,
        Response:   errResponse("INTERNAL_ERROR", err.Error()),
    }
}
```

Composite's `NewError` also delegates to `handler.ToAPIError` after logging.

### Logger

Always use the `logger.Logger` interface from `internal/logger`, injected via DI (`logger.RegisterLogger`). In tests, use `logger.NewMockLogger()`.
