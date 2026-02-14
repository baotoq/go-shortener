# Phase 2: Event-Driven Analytics - Context

**Gathered:** 2026-02-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Track clicks asynchronously via Dapr pub/sub without blocking redirects. Introduce the Analytics Service as a second microservice that receives click events and exposes a simple click count API. Geo-location, device tracking, and traffic source enrichment are Phase 3.

</domain>

<decisions>
## Implementation Decisions

### Click event payload
- Minimal payload: short code + timestamp only
- No IP, User-Agent, or Referer in Phase 2 — Phase 3 will extend the event schema when needed
- Individual click records stored (one row per click), not just counters
- Inline fire-and-forget publish during redirect handler — no internal queue
- On pub/sub failure: log the error and continue with redirect (click is lost, redirect succeeds)

### Analytics API shape
- Endpoint: `GET /analytics/{code}` on the Analytics Service directly (separate port)
- Response: `{"short_code": "abc", "total_clicks": 42}` — total count only for Phase 2
- Zero clicks returns `{"short_code": "abc", "total_clicks": 0}` — not 404
- Analytics Service has its own HTTP server, not proxied through URL Service

### Project structure
- Single Go module, two binaries: `cmd/url-service/` and `cmd/analytics-service/`
- Restructure repo: move existing URL Service code under `services/` directory layout
- Separate SQLite database file for Analytics Service (analytics.db) — true service data isolation
- Shared internal package for common types (e.g., click event struct in `internal/shared/` or `pkg/events/`)

### Dapr pub/sub setup
- In-memory pub/sub component for local development (zero dependencies, messages lost on restart is acceptable)
- Dapr component files in root `dapr/components/` directory — shared across services
- Programmatic subscription via `/dapr/subscribe` endpoint in Analytics Service code
- Makefile targets for running services: `make run-url`, `make run-analytics` wrapping `dapr run`

### Claude's Discretion
- Topic naming convention
- Dapr app-id naming for each service
- Exact port assignments for each service
- Analytics Service clean architecture layer structure (mirror URL Service patterns)
- SQLite schema for click records table
- Error response format for Analytics API (follow RFC 7807 pattern from Phase 1)

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. The project is a learning exercise for Go + Dapr patterns, so following idiomatic conventions is preferred.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 02-event-driven-analytics*
*Context gathered: 2026-02-15*
