# Go URL Shortener

## What This Is

A URL shortener built with Go and go-zero, structured as two microservices communicating via zRPC (gRPC) and Kafka. Supports URL shortening with redirect, event-driven click analytics with geo/device/referer enrichment, and link management with pagination and search. Rebuilt on go-zero's framework with PostgreSQL, Kafka, and code generation.

## Core Value

Shorten a long URL and reliably redirect anyone who visits the short link. Everything else (analytics, events) is secondary to this working correctly.

## Current Milestone: v2.0 Go-Zero Adoption

**Goal:** Full stack rewrite from Chi/Dapr/SQLite to go-zero/zRPC/Kafka/PostgreSQL with feature parity.

**Target features:**
- Rebuild both services using go-zero handler → logic → model pattern with `.api` spec code generation
- Replace Dapr service invocation with zRPC (gRPC/protobuf)
- Replace Dapr pub/sub with Kafka via go-zero queue
- Migrate from SQLite + sqlc to PostgreSQL + go-zero sqlx
- Maintain all v1.0 features (shorten, redirect, analytics, link management)
- Docker Compose orchestration with PostgreSQL, Kafka, and both services

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

- [ ] Rebuild URL Service with go-zero `.api` spec and code generation
- [ ] Rebuild Analytics Service with go-zero `.api` spec and code generation
- [ ] Replace Dapr service invocation with zRPC (gRPC/protobuf)
- [ ] Replace Dapr pub/sub with Kafka via go-zero queue
- [ ] Migrate from SQLite to PostgreSQL with go-zero sqlx
- [ ] Docker Compose with PostgreSQL, Kafka, Zookeeper, and both services
- [ ] Maintain feature parity with v1.0 (shorten, redirect, analytics, link management)

### Out of Scope

- Web UI / frontend — API-only project
- User accounts / authentication — not needed for learning goals
- Custom aliases — keeping short codes auto-generated
- Link expiration — adds complexity without teaching new patterns
- Redis caching — defer to future milestone
- Service mesh / Kubernetes — Docker Compose for now

## Context

Shipped v1.0 with 9,401 LOC Go across 56 files.
v2.0 is a full stack rewrite: go-zero replaces Chi + Dapr entirely.
Go-zero provides: `.api` spec code generation, zRPC (gRPC), built-in middleware (rate limiting, circuit breaker, load shedding), Kafka queue integration, sqlx for database access.
Two services preserved: **URL Service** (HTTP API) and **Analytics Service** (zRPC + Kafka consumer).
PostgreSQL replaces SQLite for production-grade storage.
Kafka replaces Dapr's in-memory pub/sub for reliable event delivery.

## Constraints

- **Language**: Go — the entire point of the project
- **Framework**: go-zero — learning its conventions, code generation, and patterns
- **Database**: PostgreSQL — production-grade relational database
- **Events**: Kafka — reliable async event pipeline via go-zero queue
- **Service comms**: zRPC (gRPC/protobuf) — go-zero's native RPC framework
- **Deployment**: Docker Compose to orchestrate services + infrastructure

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
| go-zero full adoption (v2.0) | Learn go-zero framework patterns, code generation, built-in tooling | — Pending |
| Drop Dapr for go-zero native | go-zero provides zRPC + queue natively, Dapr adds unnecessary layer | — Pending |
| PostgreSQL over SQLite | Production-grade, removes single-writer limitation | — Pending |
| Kafka for events | Reliable, persistent event pipeline via go-zero queue integration | — Pending |

---
*Last updated: 2026-02-15 after v2.0 milestone start*
