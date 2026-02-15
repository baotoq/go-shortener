# Go URL Shortener

A URL shortener built as two Go microservices communicating via [Dapr](https://dapr.io/) pub/sub. Shorten a URL, redirect visitors, and track rich click analytics — all with clean architecture and zero external infrastructure beyond Dapr.

## Key Features

- Shorten URLs and redirect via short codes (302)
- Event-driven click tracking (fire-and-forget, never blocks redirects)
- Analytics enrichment: geo-location (GeoIP), device type (User-Agent), traffic source (Referer)
- Summary and detail analytics endpoints with date filtering and cursor pagination
- Per-IP rate limiting with token bucket algorithm
- RFC 7807 Problem Details for all API errors
- Embedded database migrations (no separate migration tool needed)

## Tech Stack

| Layer | Technology |
|---|---|
| **Language** | Go 1.24 |
| **Runtime** | Dapr 1.13 (sidecar pattern) |
| **HTTP Router** | chi v5 |
| **Database** | SQLite (pure Go driver, no CGO) |
| **SQL Codegen** | sqlc |
| **Migrations** | golang-migrate (embedded) |
| **Short Codes** | NanoID (8-char, base62) |
| **Logging** | Zap (structured) |
| **GeoIP** | MaxMind GeoLite2 |

## Prerequisites

- **Go 1.24+**
- **Dapr CLI** ([install guide](https://docs.dapr.io/getting-started/install-dapr-cli/)) — run `dapr init` after installing
- **sqlc** (optional, only if modifying SQL) — `go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`
- **GeoLite2 Country database** (optional) — place at `data/GeoLite2-Country.mmdb` for geo-location enrichment; falls back to "Unknown" if missing

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/baotoq/go-shortener.git
cd go-shortener
```

### 2. Install Dependencies

```bash
go mod download
```

### 3. Start Both Services

```bash
make run-all
```

This starts both services with Dapr sidecars using `dapr run -f dapr.yaml`:

| Service | App Port | Dapr HTTP | Dapr gRPC |
|---|---|---|---|
| url-service | 8080 | 3500 | 50001 |
| analytics-service | 8081 | 3501 | 50002 |

SQLite databases are created automatically in `data/` on first run. Migrations run on startup.

### 4. Try It Out

```bash
# Shorten a URL
curl -s -X POST http://localhost:8080/api/v1/urls \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://github.com"}' | jq

# Redirect (follow with -L, or open in browser)
curl -I http://localhost:8080/{short_code}

# Check click count
curl -s http://localhost:8081/analytics/{short_code} | jq

# Get full analytics summary
curl -s http://localhost:8081/analytics/{short_code}/summary | jq

# Paginated click details
curl -s http://localhost:8081/analytics/{short_code}/clicks?limit=10 | jq
```

## Available Commands

| Command | Description |
|---|---|
| `make run-all` | Run both services with Dapr sidecars |
| `make run-url` | Run URL Service only (port 8080) |
| `make run-analytics` | Run Analytics Service only (port 8081) |
| `make build` | Build binaries to `bin/` |
| `sqlc generate` | Regenerate type-safe SQL code after schema changes |

## Architecture

### Service Overview

```
┌──────────────┐        Dapr pub/sub         ┌───────────────────┐
│              │  ── click event (async) ──>  │                   │
│  URL Service │       topic: "clicks"       │ Analytics Service  │
│  (port 8080) │                              │   (port 8081)     │
│              │                              │                   │
│  - Shorten   │                              │  - GeoIP lookup   │
│  - Redirect  │                              │  - UA parsing     │
│  - Rate limit│                              │  - Referer class. │
└──────┬───────┘                              └────────┬──────────┘
       │                                               │
   shortener.db                                   analytics.db
```

Each service has its own SQLite database (database-per-service pattern). Click events are published after redirects in a fire-and-forget goroutine — redirect latency is never affected by analytics processing.

### Clean Architecture Layers

Both services follow the same layered structure:

```
internal/{service}/
├── delivery/http/      # Handlers, router, middleware
│                       # Parses requests → calls use case → serializes response
├── domain/             # Entities and domain errors
│                       # Pure Go types, no framework dependencies
├── usecase/            # Business logic + repository interfaces
│                       # Orchestrates domain rules, defines ports
├── repository/sqlite/  # sqlc-generated data access
│                       # Implements repository interfaces
└── database/           # Connection setup + embedded migrations
```

Dependency flow: `delivery` → `usecase` → `repository` (interfaces in usecase, implementations in repository).

### Directory Structure

```
go-shortener/
├── cmd/
│   ├── url-service/main.go         # URL Service entry point
│   └── analytics-service/main.go   # Analytics Service entry point
├── internal/
│   ├── urlservice/                  # URL shortening + redirect
│   ├── analytics/                   # Click tracking + enrichment
│   │   └── enrichment/              # GeoIP, User-Agent, Referer classifiers
│   └── shared/events/               # Cross-service event types (ClickEvent)
├── pkg/problemdetails/              # RFC 7807 error response library
├── db/                              # SQL schemas + queries (sqlc source)
├── dapr/components/                 # Dapr component configs (in-memory pubsub)
├── dapr.yaml                        # Multi-app run configuration
├── sqlc.yaml                        # sqlc codegen configuration
└── Makefile
```

## API Reference

### URL Service (port 8080)

#### `POST /api/v1/urls` — Create Short URL

```bash
curl -X POST http://localhost:8080/api/v1/urls \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com"}'
```

**201 Created**
```json
{
  "short_code": "a1B2c3D4",
  "short_url": "http://localhost:8080/a1B2c3D4",
  "original_url": "https://example.com"
}
```

Duplicate URLs return the existing short code (idempotent).

#### `GET /api/v1/urls/{code}` — Get URL Details

**200 OK**
```json
{
  "short_code": "a1B2c3D4",
  "short_url": "http://localhost:8080/a1B2c3D4",
  "original_url": "https://example.com"
}
```

#### `GET /{code}` — Redirect

**302 Found** with `Location: https://example.com`

Publishes a `ClickEvent` to the `clicks` topic asynchronously after sending the redirect.

#### Rate Limiting

100 requests/minute per IP (configurable via `RATE_LIMIT` env var). Headers included:

| Header | Description |
|---|---|
| `X-RateLimit-Limit` | Max requests allowed |
| `X-RateLimit-Remaining` | Remaining requests |
| `X-RateLimit-Reset` | Unix timestamp when limit resets |

### Analytics Service (port 8081)

#### `GET /analytics/{code}` — Click Count

**200 OK**
```json
{
  "short_code": "a1B2c3D4",
  "total_clicks": 42
}
```

#### `GET /analytics/{code}/summary` — Full Analytics

Query params: `from` and `to` (YYYY-MM-DD, optional).

**200 OK**
```json
{
  "short_code": "a1B2c3D4",
  "total_clicks": 42,
  "countries": [
    { "value": "US", "count": 25, "percentage": "59.5%" }
  ],
  "device_types": [
    { "value": "Desktop", "count": 30, "percentage": "71.4%" }
  ],
  "traffic_sources": [
    { "value": "Direct", "count": 18, "percentage": "42.9%" }
  ]
}
```

#### `GET /analytics/{code}/clicks` — Paginated Click Details

Query params: `cursor` (base64, optional), `limit` (1-100, default 20).

**200 OK**
```json
{
  "clicks": [
    {
      "short_code": "a1B2c3D4",
      "clicked_at": 1707984000,
      "country_code": "US",
      "device_type": "Desktop",
      "traffic_source": "Social"
    }
  ],
  "next_cursor": "MTcwNzk4NDAwMA==",
  "has_more": true
}
```

### Error Responses

All errors use [RFC 7807 Problem Details](https://www.rfc-editor.org/rfc/rfc7807) with `Content-Type: application/problem+json`:

```json
{
  "type": "https://api.example.com/problems/invalid-url",
  "title": "Invalid URL",
  "status": 400,
  "detail": "original_url is required"
}
```

## Database Schema

### URL Service (`data/shortener.db`)

```sql
CREATE TABLE urls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

### Analytics Service (`data/analytics.db`)

```sql
CREATE TABLE clicks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT NOT NULL,
    clicked_at INTEGER NOT NULL,       -- Unix timestamp
    country_code TEXT NOT NULL,         -- e.g. "US", "Unknown"
    device_type TEXT NOT NULL,          -- e.g. "Desktop", "Mobile"
    traffic_source TEXT NOT NULL        -- e.g. "Direct", "Social"
);
```

## Environment Variables

All have sensible defaults — no `.env` file needed for local development.

### URL Service

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_PATH` | `data/shortener.db` | SQLite database path |
| `BASE_URL` | `http://localhost:8080` | Base URL for generated short links |
| `RATE_LIMIT` | `100` | Requests per minute per IP |

### Analytics Service

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8081` | HTTP server port |
| `DATABASE_PATH` | `data/analytics.db` | SQLite database path |
| `GEOIP_DB_PATH` | `data/GeoLite2-Country.mmdb` | MaxMind GeoLite2 database path |

## Event Flow

```
User clicks short link
        │
        ▼
  URL Service: GET /{code}
        │
        ├─► 302 Redirect (immediate)
        │
        └─► goroutine (fire-and-forget):
              │
              ▼
         Dapr Sidecar
         POST /v1.0/publish/pubsub/clicks
              │
              ▼
         Analytics Service: POST /events/click
              │
              ├─► GeoIP lookup (IP → country)
              ├─► User-Agent parse (→ device type)
              ├─► Referer classify (→ traffic source)
              │
              └─► INSERT enriched click into analytics.db
```

**ClickEvent payload:**
```json
{
  "short_code": "a1B2c3D4",
  "timestamp": "2026-02-15T10:30:00Z",
  "client_ip": "203.0.113.1",
  "user_agent": "Mozilla/5.0 ...",
  "referer": "https://twitter.com/..."
}
```

## Project Status

This is a learning project focused on Go patterns and Dapr. See [`.planning/ROADMAP.md`](.planning/ROADMAP.md) for the full delivery plan.

| Phase | Description | Status |
|---|---|---|
| 1 | Foundation & URL Service Core | Done |
| 2 | Event-Driven Analytics (Dapr pub/sub) | Done |
| 3 | Enhanced Analytics (GeoIP, UA, Referer) | Done |
| 4 | Link Management (list, filter, delete) | In Progress |
| 5 | Production Readiness (tests, CI/CD, Docker) | Planned |

## Troubleshooting

### Dapr not initialized

```
ERROR: Dapr is not initialized in your local environment.
```

Run `dapr init` to initialize the Dapr runtime locally.

### Port already in use

```
listen tcp :8080: bind: address already in use
```

Another process is using the port. Find it with `lsof -i :8080` and stop it, or change the `PORT` env var.

### GeoIP "Unknown" for all requests

The GeoLite2 database is optional. To enable geo-location:
1. Download `GeoLite2-Country.mmdb` from [MaxMind](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)
2. Place it at `data/GeoLite2-Country.mmdb`
3. Restart the analytics service

### Click events not arriving

1. Verify both services are running: check Dapr dashboard or logs
2. Ensure Dapr sidecars started (look for "dapr initialized" in logs)
3. The in-memory pub/sub loses messages on restart — this is expected for local dev

### SQLite locked errors

SQLite is configured with `SetMaxOpenConns(1)`. If you see locking errors, ensure only one instance of each service is running against the same database file.
