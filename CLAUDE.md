# Go Shortener

URL shortener microservices project using Go, Dapr, and SQLite.

## Commands

```bash
make run-all          # Run both services with Dapr sidecars
make run-url          # URL Service only (port 8080)
make run-analytics    # Analytics Service only (port 8081)
make build            # Build binaries to bin/
sqlc generate         # Regenerate type-safe SQL code after schema changes
```

## Architecture

Two microservices communicating via Dapr pub/sub (in-memory for dev):

- **url-service** (port 8080): URL shortening + redirect. Publishes click events.
- **analytics-service** (port 8081): Subscribes to click events, enriches with GeoIP/UA/Referer.

Each service follows Clean Architecture: `delivery/http` → `usecase` → `repository/sqlite`
Repository interfaces defined in usecase package (dependency inversion).

```
internal/{service}/
├── delivery/http/    # Handlers, router, middleware
├── domain/           # Entities, domain errors
├── usecase/          # Business logic + repository interfaces
├── repository/sqlite/# sqlc-generated data access
└── database/         # Connection + embedded migrations
```

## Key Patterns

- **sqlc code generation**: SQL in `db/*.sql` → generated Go in `repository/sqlite/`
- **Embedded migrations**: Schema files embedded in binaries via `//go:embed`
- **RFC 7807 Problem Details**: All API errors use `pkg/problemdetails/`
- **Fire-and-forget events**: Click events published in goroutine after redirect response
- **Dapr nil-safety**: Services gracefully degrade if Dapr sidecar unavailable
- **Shared events**: `internal/shared/events/` defines cross-service event types

## Gotchas

- SQLite uses `SetMaxOpenConns(1)` — single writer, cannot scale horizontally
- Dapr pub/sub subscriber returns 200 even on processing errors (prevents retry storm)
- Short codes: NanoID 8-char with collision retry (max 5 attempts)
- Rate limiter is in-memory per-instance, resets on restart
- GeoIP database (`data/GeoLite2-Country.mmdb`) is optional — falls back to "Unknown"
- CloudEvents: Dapr wraps events, handler must extract `data` field before unmarshaling
- Indentation: project uses 2 spaces for Go (see .editorconfig), not standard tabs

## Environment Variables

All have defaults suitable for local dev (no .env needed):
- `PORT` (8080/8081), `DATABASE_PATH`, `BASE_URL`, `RATE_LIMIT`, `GEOIP_DB_PATH`

## Project Status

See `.planning/ROADMAP.md` for full plan. Phase 4 (Link Management) in progress.
`.planning/STATE.md` tracks real-time progress and decisions.
