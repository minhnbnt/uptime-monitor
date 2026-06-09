# AGENTS.md — Uptime Monitor

## Quick start

```sh
make dev              # air hot-reload
make build            # production build
make generate         # oapi-codegen from api/spec.yaml
make test             # run all unit tests
make test-cover       # unit tests with coverage
make test-cover-html  # open coverage report in browser
go build ./...        # compile check
golangci-lint run ./...  # lint + format (gofmt, gci, govet, revive, misspell)
hurl --test --variable base_url=http://localhost:8080 tests/*.hurl
```

> **Lint before commit**: always run `golangci-lint run ./...` before committing. It enforces import order (gci), `interface{}`→`any` (gofmt rewrite), and revives lints.

## Architecture

- **DI**: `samber/do/v2` — every component has a `Register*` function called in `main.go` (registration order matters: config → repo → service → handler).
- **Handler → Service → Repository → DB**: layers use consumer-package interfaces (defined where used, not where implemented).
- **API**: OpenAPI spec at `api/spec.yaml`. Run `make generate` to regenerate `generated/api/gen.go` via oapi-codegen (gin-server mode).
- **CompositeHandler** (`internal/server/composite.go`) embeds `ServerHandler` and `EndpointHandler` to satisfy the generated `ServerInterface`.
- **Router**: Gin, using oapi-codegen's `RegisterHandlers`.

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
- **Database**: PostgreSQL with GORM auto-migrate (`Server`, `Endpoint` models). No manual migrations.
- **Import order** (enforced by gci): std → third-party → `github.com/minhnbnt/uptime-monitor/`.
- **`interface{}` → `any`** enforced by gofmt rewrite rule in `.golangci.yml`.
- **Before committing**: run `golangci-lint run ./...` to ensure code quality.
