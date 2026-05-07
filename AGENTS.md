# AGENTS.md — Bluebell

Go + Vue 3 community web app. DDD architecture, Gin + GORM backend, Vue 3 + Vite + Tailwind frontend.

## Commands

```bash
make build        # Cross-compile to Linux amd64 (CGO_ENABLED=0)
make run          # BROKEN — see Gotchas
make gotool       # go fmt + go vet
make all          # gotool + build
make clean        # Remove binary

# Dev with hot-reload (correct entrypoint)
air               # Uses .air.toml — watches cmd/, internal/, config/

# Frontend
cd frontend && npm run dev    # Vite dev server on :5173, proxies /api/v1 → :8080
cd frontend && npm run build  # vue-tsc -b && vite build
```

## Architecture

Single Go module `bluebell`. Entrypoint: `cmd/bluebell/main.go`.

```
internal/
├── config/          # Viper config loading
├── dao/             # Data access (database/, cache/)
├── domain/          # Domain models (cachedomain/, dbdomain/, svcdomain/)
├── dto/             # Request/response DTOs
├── handler/         # Gin HTTP handlers
├── http_server/     # HTTP server setup, graceful shutdown
├── infrastructure/  # ES, JWT, logger, OTEL, snowflake, translate
├── middleware/       # Gin middleware (auth, ratelimit, etc.)
├── model/           # GORM DB models
├── router/          # Route registration
└── service/         # Business logic (communitysvc/, postsvc/, usersvc/, mq/)
pkg/
├── enum/            # Shared enums
└── errorx/          # Shared error types
```

DI flows: `main.go` → init infra → init repos (UoW) → init services → init handlers → init router.

## Configuration

- `config.yaml` — local dev (read by `-conf` flag, default `./config.yaml`)
- `config.docker.toml` — Docker container config
- `.air.toml` — air watches `cmd`, `internal`, `config`; builds `./cmd/bluebell/`

`main.go` hardcodes `gin.ReleaseMode` — `config.App.Mode` only affects logger, not Gin.

## Services (Docker Compose)

| Service | Port | Notes |
|---------|------|-------|
| MySQL | 3307:3306 | db: `bluebell`, root password in docker-compose.yml |
| Redis | 6380:6379 | |
| RabbitMQ | 5672 / 15672 | mgmt UI, user/pass: bluebell/bluebell123 |
| Elasticsearch | 9200 | single-node, no auth |
| Jaeger | 16686 | UI, OTLP ingest via OTEL Collector :4318 |
| Nginx (gateway) | 80 | Proxies to backend + serves frontend static assets |

Start deps: `docker-compose up -d mysql redis rabbitmq elasticsearch`

ES and RabbitMQ are **non-critical** — app logs error and continues if they fail.

## Frontend

Vue 3 + TypeScript + Vite + Tailwind CSS 4. No test framework configured.

- `frontend/vite.config.ts` — proxies `/api/v1` to `http://127.0.0.1:8080`
- `frontend/package.json` — no test scripts, no test deps

## Testing

**No test infrastructure.** No `*_test.go` files, no test deps in `go.mod` or `package.json`. README's `go test ./...` is aspirational.

## Gotchas

- **`make run` is broken** — runs `go run ./main.go ./config.yaml` from root, but `main.go` lives in `cmd/bluebell/`. Use `air` for dev, or fix the Makefile to use `go run ./cmd/bluebell/ -conf ./config.yaml`.
- **Gin mode** is hardcoded to `ReleaseMode` in `main.go` — changing `config.app.mode` to `debug` won't affect Gin.
- **Non-standard ports** — MySQL on 3307, Redis on 6380 (not defaults). Check `config.yaml` / `docker-compose.yml` before connecting.
- **Swagger** — `docs/docs.go` is checked in; regenerate with `swag init` if API annotations change.
- **No CI workflows** — `.github/workflows/` does not exist.
- **No pre-commit hooks** — linting is manual via `make gotool`.
