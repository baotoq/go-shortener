# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-15)

**Core value:** Shorten a long URL and reliably redirect anyone who visits the short link
**Current focus:** Phase 11 - CI Pipeline & Docker Hardening (COMPLETE)

## Current Position

Phase: 11 of 11 (CI Pipeline & Docker Hardening)
Plan: 3 of 3 in current phase
Status: Phase Complete (11-03 complete: CI pipeline & integration tests)
Last activity: 2026-02-16 - Completed Phase 11 (all 3 plans)

Progress: [██████████] 100% (33/33 total plans)

## Performance Metrics

**Velocity:**
- Total plans completed: 33 (v1.0: 18, v2.0: 15)
- Average duration: 218s (v2.0 phases 7-11)
- Total execution time: ~3330s (v2.0 actual)

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 1. Foundation & URL Service Core | 3 | - | - |
| 2. Event-Driven Analytics | 3 | - | - |
| 3. Enhanced Analytics | 3 | - | - |
| 4. Link Management | 2 | - | - |
| 5. Production Readiness | 5 | - | - |
| 6. Test Coverage Hardening | 2 | - | - |
| 7. Framework Foundation | 3 | 653s | 218s |
| 8. Database Migration | 3 | ~550s | ~183s |
| 9. Messaging Migration | 3 | ~600s | ~200s |
| 10. Resilience & Infrastructure | 3 | 927s | 309s |
| 11. CI Pipeline & Docker Hardening | 3 | ~600s | ~200s |

**Recent Trend:**
- v1.0 completed: 6 phases, 18 plans
- v2.0 COMPLETE: Phases 7-11 all complete (15 plans total)

*Updated: 2026-02-16 after Phase 11 execution complete*

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- **go-zero full adoption (v2.0)**: Learn go-zero framework patterns, code generation, built-in tooling
- **Drop Dapr for go-zero native**: go-zero provides zRPC + queue natively, Dapr adds unnecessary layer
- **PostgreSQL over SQLite**: Production-grade, removes single-writer limitation
- **Kafka for events**: Reliable, persistent event pipeline via go-zero queue integration
- **Enhanced ClickEvent (07-01)**: Move IP/UserAgent/Referer from enrichment layer to event for cleaner consumer
- **Validation support in problemdetails (07-01)**: Added FieldError type for go-zero validator integration
- **RFC 7807 global error handler (07-02)**: Use httpx.SetErrorHandlerCtx for consistent Problem Details responses
- **Stub logic pattern (07-02)**: Return mock data to prove full request/response pipeline before DB wiring
- **Port allocation (07-03)**: URL API on 8080, Analytics RPC on 8081 for clear service separation
- **Direct connection mode (07-03)**: No Etcd in Phase 7, service discovery deferred to v2.1
- **PostgreSQL on port 5433 (08-01)**: Port 5432 occupied by existing container, all configs use 5433
- **ProblemDetail.Body() pattern (08-02)**: go-zero writes plaintext for error-implementing bodies, Body() returns non-error struct
- **UUIDv7 + NanoID (08-02)**: google/uuid for time-ordered PKs, go-nanoid for 8-char short codes with collision retry
- **Fire-and-forget click increment (08-02)**: Goroutine with context.Background() for async DB update
- **Zero-value click semantics (08-03)**: Unknown short codes return 0 total_clicks, not error
- **Kafka KRaft mode (09-01)**: No Zookeeper dependency, single-node Kafka with KRaft for local dev
- **analytics-consumer as separate service (09-01)**: Dedicated Kafka consumer service, not embedded in analytics-rpc
- **threading.GoSafe for Kafka publishing (09-02)**: Fire-and-forget pattern preserved, panic recovery via go-zero threading
- **zRPC graceful degradation (09-02)**: GetLinkDetail returns 0 clicks if analytics-rpc is down, does not fail
- **Click enrichment in consumer (09-03)**: GeoIP (MaxMind), UA parsing (mssola/useragent), referer classification
- **Idempotent click inserts (09-03)**: Duplicate key errors silently skipped, no count inflation

### Phase 10 Execution Decisions

**Plan 10-01 (Resilience & Observability):**
- **DevServer port allocation (10-01)**: url-api=6470, analytics-rpc=6471, analytics-consumer=6472 - unique ports avoid conflicts
- **Consumer DevServer initialization (10-01)**: Explicit c.MustSetUp() call after config load for non-HTTP services
- **AnalyticsRpc timeout (10-01)**: 2000ms timeout on zRPC client for bounded request durations

**Plan 10-02 (Docker Compose Orchestration):**
- **Multi-stage Dockerfile (10-02)**: Single Dockerfile with named build targets for all three services, shared build cache
- **Kafka dual listeners (10-02)**: PLAINTEXT (kafka:9092 for Docker), PLAINTEXT_HOST (localhost:29092 for host) enables both containerized and local dev
- **Kafka as root user (10-02)**: Apache Kafka 3.7.0 tmpfs permission issue resolved by running as root, ephemeral data acceptable for dev
- **Auto-migrations via initdb (10-02)**: PostgreSQL migrations auto-applied on first startup via docker-entrypoint-initdb.d volume mount

**Plan 10-03 (Unit Testing Suite):**
- **Accept 76.4% consumer coverage (10-03)**: resolveCountry requires real GeoIP database file impractical for unit tests, all testable logic paths covered
- **Hand-written mocks over mockgen (10-03)**: Simple function field pattern avoids code generation tooling for small project

### Phase 10 Planning Decisions

- **DevServer for Prometheus (10-01)**: go-zero auto-exposes metrics on separate port via DevServer config, no custom middleware
- **Port allocation for DevServer**: url-api=6470, analytics-rpc=6471, analytics-consumer=6472
- **Built-in circuit breakers (10-01)**: zRPC clients have adaptive Google SRE breaker by default, no custom code
- **Multi-target Dockerfile (10-02)**: Single Dockerfile with named stages for all three services
- **Docker-specific YAML configs (10-02)**: Separate config files with container hostnames (postgres, kafka, analytics-rpc)
- **Dual Kafka listeners (10-02)**: PLAINTEXT for Docker internal, PLAINTEXT_HOST for localhost dev
- **Hand-written mocks (10-03)**: Simple mock structs implementing model interfaces, no mockgen tooling
- **Unit tests only (10-03)**: No integration tests with testcontainers in this phase (deferred to CI pipeline)

### Phase 11 Execution Decisions

**Plan 11-01 (Connection Pool & Health Checks):**
- **PoolConfig as int seconds (11-01)**: ConnMaxLifetime stored as int, converted to time.Duration in ServiceContext (go-zero json doesn't parse durations)
- **Dedicated health server for consumer (11-01)**: analytics-consumer runs separate HTTP server on port 8082 in goroutine (no REST framework)
- **Liveness-only health checks (11-01)**: /healthz confirms process is running, no DB/Kafka probing

**Plan 11-02 (Kafka Docker Fix):**
- **Advertised listener fix (11-02)**: PLAINTEXT_HOST changed from localhost:29092 to localhost:9092 to match host port mapping
- **Auto-create topics enabled (11-02)**: KAFKA_AUTO_CREATE_TOPICS_ENABLE for reliable first-run topic creation

**Plan 11-03 (CI Pipeline & Integration Tests):**
- **golangci-lint v2 format (11-03)**: linters.settings replaces linters-settings, linters.exclusions replaces issues.exclude-*
- **Suppress SA5008/ST checks (11-03)**: go-zero uses custom json tags and non-standard naming conventions
- **Removed revive linter (11-03)**: go-zero framework doesn't follow standard Go doc comment conventions
- **testcontainers-go with pgx driver (11-03)**: Integration tests use pgx for testcontainers compatibility
- **Build tag gating (11-03)**: //go:build integration prevents integration tests from running in unit test job

### Pending Todos

None.

### Blockers/Concerns

**Phase 10 (Resilience & Infrastructure):**
- ~~DevServer on analytics-consumer (non-HTTP service) may need explicit devserver.NewServer start if service.ServiceConf embedding doesn't auto-start it~~ RESOLVED (10-01): Added c.MustSetUp() call
- ~~Dual Kafka listener approach needs verification that both Docker and local dev workflows work correctly (10-02)~~ RESOLVED (10-02): Dual listeners verified, both Docker and local dev workflows functional
- GeoIP database not included in Docker images (falls back to "XX") - acceptable for development (10-02)
- ~~Kafka message publishing from url-api not working in Docker~~ RESOLVED (11-02): Fixed PLAINTEXT_HOST advertised listener from localhost:29092 to localhost:9092
- mssola/useragent library mobile detection limitation - does not detect iPhone/Android (10-03)

## Session Continuity

Last session: 2026-02-16
Stopped at: Completed Phase 11 (CI Pipeline & Docker Hardening) - all 3 plans
Resume file: N/A
Next action: v2.0 milestone complete - all 15 plans across phases 7-11 done

---
*Initialized: 2026-02-14*
*Last updated: 2026-02-16 after Phase 11 execution complete (v2.0 COMPLETE)*
