# Go URL Shortener

## What This Is

A URL shortener built with Go and Dapr, structured as two microservices communicating via Dapr building blocks. Supports URL shortening with redirect, event-driven click analytics with geo/device/referer enrichment, and link management with pagination and search. Production-ready with Docker Compose, CI/CD, and 80%+ test coverage.

## Core Value

Shorten a long URL and reliably redirect anyone who visits the short link. Everything else (analytics, events) is secondary to this working correctly.

## Requirements

### Validated

- ✓ Shorten a URL via API and receive a short link back — v1.0
- ✓ Redirect to original URL when visiting a short link (302) — v1.0
- ✓ Track click count per short link via Dapr pub/sub — v1.0
- ✓ GeoIP country, device/browser, referer tracking — v1.0
- ✓ List, detail, delete short links with pagination — v1.0
- ✓ URL Service and Analytics Service communicate via Dapr pub/sub — v1.0
- ✓ Services call each other via Dapr service invocation — v1.0
- ✓ Clean architecture layers (handler → service → repository) — v1.0
- ✓ SQLite for persistent storage — v1.0
- ✓ Docker Compose with Dapr sidecars — v1.0
- ✓ Unit tests for all layers with 80%+ coverage — v1.0
- ✓ Integration tests with real Dapr sidecars — v1.0
- ✓ CI pipeline (lint, test, build) in GitHub Actions — v1.0
- ✓ Rate limiting prevents abuse (100 req/min per IP) — v1.0

### Active

(None — next milestone not yet defined)

### Out of Scope

- Web UI / frontend — API-only project
- User accounts / authentication — not needed for learning goals
- Custom aliases — keeping short codes auto-generated for v1
- Link expiration — adds complexity without teaching new Go/Dapr patterns
- Dapr state store — using SQLite directly to learn both raw DB access and Dapr separately

## Context

Shipped v1.0 with 9,401 LOC Go across 56 files.
Tech stack: Go 1.24, Dapr, Chi router, SQLite (modernc.org/sqlite), sqlc, Zap logger.
Two services: **URL Service** (port 8080) and **Analytics Service** (port 8081).
Dapr provides pub/sub for click events and service invocation for cross-service calls.
Clean architecture ensures each layer is testable and swappable.
90.7% handler test coverage, integration tests with real Dapr sidecars in Docker.

## Constraints

- **Language**: Go — the entire point of the project
- **Runtime**: Dapr — learning distributed application patterns
- **Storage**: SQLite — zero-dependency, file-based database
- **Architecture**: Clean architecture with handler/service/repository layers
- **Deployment**: Docker Compose to orchestrate services + Dapr sidecars

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 2 services (URL + Analytics) | Clear separation of concerns, natural pub/sub boundary | ✓ Good — clean service isolation, independent scaling |
| SQLite over Dapr state store | Learn raw DB access alongside Dapr; keep storage simple | ✓ Good — full SQL control, single-writer is fine for learning |
| Dapr for inter-service communication | Learn distributed patterns (pub/sub, service invocation) | ✓ Good — clean abstraction, graceful degradation |
| Clean architecture | Learn production Go patterns (layered, testable, swappable) | ✓ Good — enabled 80%+ coverage with mock injection |
| sqlc for type-safe SQL | Compile-time safety with full SQL control | ✓ Good — caught schema drift at build time |
| Chi router | Idiomatic Go, good middleware support | ✓ Good — clean route groups, middleware ordering |
| RFC 7807 Problem Details | Standardized API error responses | ✓ Good — consistent error format across services |
| NanoID 8-char codes | ~218 trillion combinations, collision retry | ✓ Good — no collisions in practice |
| Fire-and-forget pub/sub | User never blocked by analytics | ✓ Good — redirect latency unaffected |
| DaprClient interface wrapper | Testability without mocking 40-method SDK interface | ✓ Good — enabled publishClickEvent and service invocation tests |
| testcontainers-go for integration | Real Docker containers, not mocks | ✓ Good — catches real Dapr issues |

---
*Last updated: 2026-02-15 after v1.0 milestone*
