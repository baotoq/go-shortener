# Go Shortener

URL shortener microservices project using Go and go-zero framework.

## Commands

```bash
# Daily workflow (via Makefile)
make run-url            # Run URL API service on :8080
make run-analytics      # Run Analytics RPC service on :8081
make run-consumer       # Run Analytics Consumer service
make run-all            # Run all services
make db-up              # Start PostgreSQL (Docker Compose)
make db-down            # Stop PostgreSQL
make db-migrate         # Apply SQL migrations
make kafka-up           # Start Kafka (Docker Compose)
make kafka-down         # Stop Kafka

# Code generation
make gen-url            # Regenerate URL API from .api spec
make gen-analytics      # Regenerate Analytics RPC from .proto spec
make gen-url-model      # Regenerate URL model from PostgreSQL (requires db-up)
make gen-clicks-model   # Regenerate clicks model from PostgreSQL (requires db-up)
```

## Architecture

Three microservices using go-zero framework:

- **url-api** (services/url-api): URL shortening REST API. go-zero .api spec code generation. Publishes click events to Kafka and calls analytics-rpc via zRPC.
- **analytics-rpc** (services/analytics-rpc): Analytics zRPC service. Protobuf-generated gRPC server. Serves click counts from PostgreSQL.
- **analytics-consumer** (services/analytics-consumer): Kafka consumer. Consumes click events, enriches with GeoIP/device/referer, writes to clicks table.

Shared packages:
- **common/events**: ClickEvent struct (cross-service event contract)
- **pkg/problemdetails**: RFC 7807 Problem Details error library

Each service follows go-zero convention: Handler → Logic → Model (via ServiceContext DI).

```
services/{service}/
├── {service}.api or .proto  # Spec file (source of truth)
├── {service}.go             # Main entry point
├── etc/
│   └── {service}.yaml       # Configuration
├── model/                   # Database models (goctl-generated + custom)
│   ├── *model.go            # Custom methods (edit these)
│   └── *model_gen.go        # Generated CRUD (never edit)
└── internal/
    ├── config/              # Config struct
    ├── handler/             # HTTP handlers (generated)
    ├── logic/               # Business logic (manual)
    ├── mqs/                 # Kafka consumers (analytics-consumer only)
    ├── svc/                 # ServiceContext (DI)
    └── types/               # Request/response types (generated)

services/migrations/         # PostgreSQL DDL (up/down SQL files)
```

## Messaging (Kafka)

Kafka (KRaft mode, no Zookeeper) via Docker Compose on **port 9092**.

```bash
make kafka-up  # First-time setup
```

Flow: URL redirect → Kafka `click-events` topic → analytics-consumer → enriched click in PostgreSQL.

- **Producer**: url-api publishes ClickEvent JSON via `kq.Pusher` using `threading.GoSafe` (fire-and-forget)
- **Consumer**: analytics-consumer reads from `click-events` topic, enriches with GeoIP/UA/referer, inserts into clicks table
- **Topic**: `click-events` (auto-created by producer)

## Service Communication (zRPC)

- **url-api → analytics-rpc**: GetClickCount via zRPC (gRPC/protobuf) for link detail endpoint
- Direct connection mode (`dns:///localhost:8081`), no service discovery (Etcd deferred to Phase 10)
- Graceful degradation: if analytics-rpc is down, link detail returns 0 clicks instead of failing

## Database

PostgreSQL 16 via Docker Compose on **port 5433** (not default 5432).

```bash
make db-up && make db-migrate  # First-time setup
```

Connection: `postgres://postgres:postgres@localhost:5433/shortener?sslmode=disable`

Tables: `urls` (UUIDv7 PK, short_code, original_url, click_count, created_at) and `clicks` (UUIDv7 PK, short_code, clicked_at, country_code, device_type, traffic_source).

Models generated via `goctl model pg datasource`. Custom query methods go in `*model.go`, generated CRUD in `*model_gen.go` (never edit).

## Key Patterns

- **goctl code generation**: .api/.proto specs generate handlers, types, routes. Never edit `types.go` or `*_gen.go`.
- **Handler/Logic separation**: Handlers parse requests. Logic contains business rules. Never mix.
- **ServiceContext**: Dependency injection container. Holds Config + Models + Pusher + RPC clients. Passed to all logic constructors.
- **RFC 7807 Problem Details**: Custom `httpx.SetErrorHandlerCtx` maps errors to Problem Details format.
- **ProblemDetail.Body()**: go-zero writes plaintext for types implementing `error`. Use `pd.Body()` to return a non-error struct for JSON serialization.
- **Validation via tags**: .api type tags (range, options, optional, default) handle all request validation.
- **UUIDv7 primary keys**: `google/uuid` for time-ordered PKs. Natural cursor ordering without separate timestamp column.
- **NanoID short codes**: `go-nanoid` 8-char codes with collision retry (max 5 attempts).
- **Fire-and-forget Kafka publishing**: Redirect handler publishes ClickEvent via `threading.GoSafe` — user never blocked by analytics.
- **Click enrichment pipeline**: Consumer enriches raw click events with country (GeoIP), device type (UA parsing), traffic source (referer classification).
- **logx structured logging**: go-zero's built-in logger with context propagation.

## Gotchas

- PostgreSQL runs on **port 5433**, not 5432 — all configs and Makefile targets use 5433
- Kafka runs on **port 9092** in KRaft mode (no Zookeeper required)
- `*model_gen.go` files are regenerated by goctl — never edit, changes will be lost
- `ProblemDetail` implements `error` interface — must use `.Body()` method in error handler to avoid go-zero's plaintext serialization
- Short codes: NanoID 8-char with collision retry (max 5 attempts)
- Rate limiter is in-memory per-instance, resets on restart
- GeoIP database (`data/GeoLite2-Country.mmdb`) is optional — falls back to "XX"
- analytics-rpc failure degrades gracefully — link detail returns 0 clicks, does not error
- Indentation: Go files use tabs (see .editorconfig), other files use 2 spaces

## Configuration

Config via YAML files in `services/{service}/etc/`. Key fields:

- **url-api**: `Port` (8080), `BaseUrl`, `DataSource`, `KqPusherConf` (Kafka brokers/topic), `AnalyticsRpc` (zRPC target)
- **analytics-rpc**: `ListenOn` (0.0.0.0:8081), `DataSource`
- **analytics-consumer**: `DataSource`, `KqConsumerConf` (Kafka brokers/group/topic), `GeoIPPath`

## Project Status

See `.planning/ROADMAP.md` for full plan. Phase 9 (Messaging Migration) complete.
`.planning/STATE.md` tracks real-time progress and decisions.
