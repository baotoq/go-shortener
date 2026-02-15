# Requirements: Go URL Shortener

**Defined:** 2026-02-15
**Core Value:** Shorten a long URL and reliably redirect anyone who visits the short link

## v2.0 Requirements

Requirements for go-zero adoption. Full stack rewrite with feature parity.

### Framework Adoption

- [ ] **FWRK-01**: URL Service defined in `.api` spec with goctl code generation (handler/logic/types)
- [ ] **FWRK-02**: Analytics Service defined in `.proto` spec with goctl zRPC generation (server/client)
- [ ] **FWRK-03**: Handler/Logic/Model separation enforced via go-zero generated structure
- [ ] **FWRK-04**: ServiceContext provides dependency injection for DB, RPC clients, Kafka producer
- [ ] **FWRK-05**: Request validation via `.api` tag rules (replaces manual handler validation)
- [ ] **FWRK-06**: RFC 7807 Problem Details preserved via custom `httpx.SetErrorHandler`
- [ ] **FWRK-07**: Configuration loaded from YAML files via go-zero Config struct
- [ ] **FWRK-08**: Service groups organize routes by domain (shorten, redirect, links)
- [ ] **FWRK-09**: go-zero `logx` replaces Zap logger for structured logging

### Database Migration

- [ ] **DBMG-01**: PostgreSQL schema created matching current SQLite tables (urls, clicks)
- [ ] **DBMG-02**: goctl model generates type-safe CRUD from PostgreSQL schema
- [ ] **DBMG-03**: Connection pool configured (MaxOpenConns, MaxIdleConns, ConnMaxLifetime)
- [ ] **DBMG-04**: URL Service reads/writes URLs to PostgreSQL via goctl-generated models
- [ ] **DBMG-05**: Analytics Service reads/writes clicks to PostgreSQL via goctl-generated models
- [ ] **DBMG-06**: Cursor pagination works with PostgreSQL (list links with pagination)
- [ ] **DBMG-07**: Search and filtering work with PostgreSQL (search by URL, filter by status)

### Messaging Migration

- [ ] **MSNG-01**: URL Service publishes ClickEvent to Kafka via go-queue `kq.Pusher` after redirect
- [ ] **MSNG-02**: Analytics Consumer processes ClickEvent from Kafka via go-queue `kq.Queue`
- [ ] **MSNG-03**: Click events enriched with GeoIP country, device/browser, referer (same as v1.0)
- [ ] **MSNG-04**: Fire-and-forget pattern preserved (redirect never blocked by analytics)
- [ ] **MSNG-05**: Kafka topic configured with appropriate retention and replication
- [ ] **MSNG-06**: Consumer handles duplicate events gracefully (idempotency)

### Service Communication

- [ ] **SRVC-01**: Analytics Service exposes GetClickCount via zRPC (gRPC/protobuf)
- [ ] **SRVC-02**: URL Service calls Analytics zRPC to enrich link details with click counts
- [ ] **SRVC-03**: Protobuf schema defines request/response types for analytics queries
- [ ] **SRVC-04**: zRPC client injected via ServiceContext with built-in circuit breaker

### Resilience & Observability

- [ ] **RESL-01**: Built-in rate limiting enabled (adaptive load shedding based on CPU)
- [ ] **RESL-02**: Circuit breaker protects downstream calls (PostgreSQL, Kafka, zRPC)
- [ ] **RESL-03**: Request timeout control configured per-service
- [ ] **RESL-04**: MaxConns limiting prevents resource exhaustion
- [ ] **RESL-05**: Prometheus metrics auto-exposed on `/metrics` endpoint
- [ ] **RESL-06**: OpenTelemetry tracing enabled for distributed request flow

### Infrastructure

- [ ] **INFR-01**: Docker Compose runs PostgreSQL, Zookeeper, Kafka, url-service, analytics-rpc, analytics-consumer
- [ ] **INFR-02**: Services start and connect to infrastructure containers correctly
- [ ] **INFR-03**: Makefile provides commands for build, run, and code generation
- [ ] **INFR-04**: CI pipeline updated for go-zero build, lint, and test

### Testing

- [ ] **TEST-01**: Unit tests for logic layer with mocked ServiceContext dependencies
- [ ] **TEST-02**: Integration tests with real PostgreSQL and Kafka (testcontainers-go)
- [ ] **TEST-03**: End-to-end flow tested: shorten → redirect → click event → analytics
- [ ] **TEST-04**: 80%+ test coverage maintained across services

## v2.1 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Caching

- **CACH-01**: Redis caching for short code → URL lookups (goctl model -c)
- **CACH-02**: Concurrent request deduplication via singleflight for redirect hot paths

### Service Discovery

- **DISC-01**: etcd service discovery for zRPC load balancing
- **DISC-02**: Health check endpoints for service monitoring

## Out of Scope

| Feature | Reason |
|---------|--------|
| Web UI / frontend | API-only project |
| User accounts / authentication | Not needed for learning goals |
| Custom aliases | Keeping short codes auto-generated |
| Link expiration | Adds complexity without teaching new patterns |
| etcd service discovery | Overkill for 2-service project, use direct connection |
| Auto-generated Swagger | Adds goctl-swagger plugin complexity |
| Full OpenTelemetry backend (Jaeger/Tempo) | Enable middleware only, defer production telemetry setup |
| KRaft Kafka (no Zookeeper) | Standard Zookeeper setup for learning |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| FWRK-01 | Phase 7 | Pending |
| FWRK-02 | Phase 7 | Pending |
| FWRK-03 | Phase 7 | Pending |
| FWRK-04 | Phase 7 | Pending |
| FWRK-05 | Phase 7 | Pending |
| FWRK-06 | Phase 7 | Pending |
| FWRK-07 | Phase 7 | Pending |
| FWRK-08 | Phase 7 | Pending |
| FWRK-09 | Phase 7 | Pending |
| DBMG-01 | Phase 8 | Pending |
| DBMG-02 | Phase 8 | Pending |
| DBMG-03 | Phase 8 | Pending |
| DBMG-04 | Phase 8 | Pending |
| DBMG-05 | Phase 8 | Pending |
| DBMG-06 | Phase 8 | Pending |
| DBMG-07 | Phase 8 | Pending |
| MSNG-01 | Phase 9 | Pending |
| MSNG-02 | Phase 9 | Pending |
| MSNG-03 | Phase 9 | Pending |
| MSNG-04 | Phase 9 | Pending |
| MSNG-05 | Phase 9 | Pending |
| MSNG-06 | Phase 9 | Pending |
| SRVC-01 | Phase 9 | Pending |
| SRVC-02 | Phase 9 | Pending |
| SRVC-03 | Phase 9 | Pending |
| SRVC-04 | Phase 9 | Pending |
| RESL-01 | Phase 10 | Pending |
| RESL-02 | Phase 10 | Pending |
| RESL-03 | Phase 10 | Pending |
| RESL-04 | Phase 10 | Pending |
| RESL-05 | Phase 10 | Pending |
| RESL-06 | Phase 10 | Pending |
| INFR-01 | Phase 10 | Pending |
| INFR-02 | Phase 10 | Pending |
| INFR-03 | Phase 10 | Pending |
| INFR-04 | Phase 10 | Pending |
| TEST-01 | Phase 10 | Pending |
| TEST-02 | Phase 10 | Pending |
| TEST-03 | Phase 10 | Pending |
| TEST-04 | Phase 10 | Pending |

**Coverage:**
- v2.0 requirements: 39 total
- Mapped to phases: 39
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-15*
*Last updated: 2026-02-15 after v2.0 roadmap creation*
