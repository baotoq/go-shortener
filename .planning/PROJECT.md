# Go URL Shortener

## What This Is

A URL shortener built with Go and go-zero, structured as three microservices (url-api, analytics-rpc, analytics-consumer) communicating via zRPC (gRPC) and Kafka. Supports URL shortening with redirect, event-driven click analytics with geo/device/referer enrichment, and link management with pagination and search. Built on go-zero's code generation, PostgreSQL, and Kafka.

## Core Value

Shorten a long URL and reliably redirect anyone who visits the short link. Everything else (analytics, events) is secondary to this working correctly.

## Current State

Shipped v2.0 — full stack rewrite from Chi/Dapr/SQLite to go-zero/zRPC/Kafka/PostgreSQL complete with feature parity.

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
- ✓ Rebuild URL Service with go-zero .api spec and code generation — v2.0
- ✓ Rebuild Analytics Service with go-zero .proto spec and zRPC generation — v2.0
- ✓ Replace Dapr service invocation with zRPC (gRPC/protobuf) — v2.0
- ✓ Replace Dapr pub/sub with Kafka via go-zero queue — v2.0
- ✓ Migrate from SQLite to PostgreSQL with go-zero sqlx — v2.0
- ✓ Docker Compose with PostgreSQL, Kafka, and all three services — v2.0
- ✓ Maintain feature parity with v1.0 (shorten, redirect, analytics, link management) — v2.0

### Active

(None — planning next milestone)

### Out of Scope

- Web UI / frontend — API-only project
- User accounts / authentication — not needed for learning goals
- Custom aliases — keeping short codes auto-generated
- Link expiration — adds complexity without teaching new patterns
- Service mesh / Kubernetes — Docker Compose for now
- ~~KRaft Kafka (no Zookeeper)~~ Implemented in v2.0 (KRaft mode, no Zookeeper)

## Context

Shipped v2.0 with 3,425 LOC Go (full rewrite: 11,139 added / 11,181 removed).
Tech stack: go-zero, PostgreSQL 16, Kafka (KRaft), zRPC (gRPC/protobuf), Docker Compose.
Three services: **url-api** (HTTP REST), **analytics-rpc** (zRPC server), **analytics-consumer** (Kafka consumer).
CI: GitHub Actions with golangci-lint v2, unit tests, testcontainers-go integration tests.
Known tech debt: dead IncrementClickCount method, consumer coverage at 76.4%, no automated E2E test.

## Constraints

- **Language**: Go — the entire point of the project
- **Framework**: go-zero — learning its conventions, code generation, and patterns
- **Database**: PostgreSQL — production-grade relational database
- **Events**: Kafka (KRaft mode) — reliable async event pipeline via go-zero queue
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
| RFC 7807 Problem Details | Standardized API error responses | ✓ Good — consistent error format, preserved across v2.0 |
| NanoID 8-char codes | ~218 trillion combinations, collision retry | ✓ Good — no collisions in practice |
| Fire-and-forget pub/sub | User never blocked by analytics | ✓ Good — redirect latency unaffected, preserved in Kafka migration |
| DaprClient interface wrapper | Testability without mocking 40-method SDK interface | ✓ Good — replaced by go-zero native patterns in v2.0 |
| testcontainers-go for integration | Real Docker containers, not mocks | ✓ Good — catches real issues, extended with PostgreSQL in v2.0 |
| go-zero full adoption (v2.0) | Learn go-zero framework patterns, code generation, built-in tooling | ✓ Good — .api/.proto code gen, built-in middleware, clean DI |
| Drop Dapr for go-zero native | go-zero provides zRPC + queue natively, Dapr adds unnecessary layer | ✓ Good — simpler architecture, fewer moving parts |
| PostgreSQL over SQLite | Production-grade, removes single-writer limitation | ✓ Good — goctl model generation, proper connection pooling |
| Kafka for events | Reliable, persistent event pipeline via go-zero queue integration | ✓ Good — KRaft mode, fire-and-forget preserved, enrichment pipeline works |
| KRaft mode (no Zookeeper) | Simpler deployment, fewer containers | ✓ Good — single Kafka container, auto-create topics |
| analytics-consumer as separate service | Dedicated Kafka consumer, not embedded in analytics-rpc | ✓ Good — independent scaling, clean separation |
| zRPC graceful degradation | Link detail returns 0 clicks if analytics-rpc down | ✓ Good — user-facing endpoint never fails due to analytics |
| Hand-written mocks over mockgen | Simple function field pattern for small project | ✓ Good — no tooling overhead, easy to maintain |
| Multi-stage Dockerfile | Single Dockerfile with named targets for all services | ✓ Good — shared build cache, consistent builds |
| testcontainers-go with pgx driver | Integration tests with real PostgreSQL | ✓ Good — catches schema/query issues early |

---
*Last updated: 2026-02-22 after v2.0 milestone*
