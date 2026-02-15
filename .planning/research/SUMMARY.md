# Project Research Summary

**Project:** go-shortener
**Domain:** Go Microservices Framework Migration (Chi/Dapr/SQLite → go-zero/Kafka/PostgreSQL)
**Researched:** 2026-02-15
**Confidence:** HIGH

## Executive Summary

This project migrates an existing URL shortener microservices application from Chi/Dapr/SQLite stack to go-zero framework with Kafka and PostgreSQL. The current v1.0 implementation has proven Clean Architecture patterns (delivery/usecase/repository) that work well. The migration goal is to adopt go-zero's production-ready framework features — built-in rate limiting, circuit breakers, load shedding, observability, and code generation — while preserving the working business logic.

The recommended approach is phased migration starting with framework adoption (API/RPC code generation, handler/logic/model separation) before attempting database or messaging migrations. Go-zero's Handler→Logic→Model pattern maps cleanly to the current delivery→usecase→repository pattern, but the framework enforces this through code generation rather than manual structure. Critical early decisions: commit to goctl-generated structure (don't fight the framework), establish .api/.proto discipline immediately (syntax errors block everything), and implement idempotency before Kafka cutover (Dapr→Kafka changes delivery semantics from at-most-once to at-least-once).

Key risk mitigation: Code generation can overwrite custom logic if developers edit generated files (critical pitfall #1). The architecture separates handler (generated, never edit) from logic (manual, safe to edit). Second major risk is SQLite→PostgreSQL migration losing data semantics due to SQLite's loose typing — audit and clean data before migration. Third risk is Kafka consumer rebalancing storms during deployment causing 30+ second processing stalls — configure session timeouts and use staggered deployments.

## Key Findings

### Recommended Stack

Go-zero v1.10.0 (Feb 2025) provides a complete microservices framework with code generation (goctl), built-in resilience patterns, and integrated observability. The migration replaces Chi router with .api specs, Dapr pub/sub with go-queue (Kafka), and SQLite with PostgreSQL. Core value proposition: production patterns (circuit breaker, load shedding, metrics, tracing) are built-in and configuration-driven, eliminating custom implementations.

**Core technologies:**
- **go-zero v1.10.0**: Microservices framework with code generation — replaces Chi router, custom middleware, manual service governance. Built-in rate limiting (token bucket), adaptive circuit breaker (Google SRE algorithm), load shedding (CPU-based), Prometheus metrics, OpenTelemetry tracing.
- **goctl v1.10.0**: Code generation CLI — generates handlers/routes/types from .api specs (HTTP services) and server/client stubs from .proto files (RPC services). Must match framework version exactly.
- **PostgreSQL 13+**: Production database — replaces SQLite's single-writer limitation. go-zero sqlx requires PostgreSQL 13+ for full feature support. Use jackc/pgx/v5 stdlib wrapper for compatibility with go-zero's sqlx-based model generation.
- **go-queue v1.2.2**: Kafka pub/sub framework — go-zero's official queue abstraction. Provides kq.Pusher (producer) and kq.Queue (consumer) with built-in error handling, retry, and consumer group management. Replaces Dapr pub/sub.
- **Keep existing libraries**: NanoID (short code generation), geoip2-golang v2 (analytics enrichment) — go-zero doesn't provide these, domain logic unchanged.

**Replace entirely:**
- Chi router → .api specs (declarative routing)
- Dapr pub/sub → go-queue (Kafka-native)
- sqlc → goctl model (PostgreSQL code generation)
- Custom middleware → go-zero built-in (rate limit, logging, metrics)

### Expected Features

The migration focuses on framework capabilities, not application features. URL shortening logic (NanoID generation, redirect, analytics enrichment) remains unchanged — this is pure infrastructure migration.

**Must have (framework features):**
- **API Code Generation** — .api spec → handlers/routes/types. Core go-zero value, eliminates manual routing boilerplate.
- **Handler/Logic/Model Separation** — go-zero enforces Clean Architecture automatically. Generated structure maps to current delivery/usecase/repository pattern.
- **Automatic Request Validation** — Built-in via .api tags (range, optional, options). Replaces manual validation in handlers.
- **Service Context (DI)** — Auto-generated dependency injection container. Holds DB, Redis, RPC clients, Kafka producers.
- **Error Response Standardization** — Custom httpx.SetErrorHandler() integration preserves RFC 7807 Problem Details format.

**Should have (differentiators):**
- **Built-in Rate Limiting** — CPU-based adaptive load shedding. Supplements existing per-IP rate limiter (behavioral change: protects service, not per-user quota).
- **Adaptive Circuit Breaker** — Google SRE algorithm with 10-second sliding window. Protects Kafka and PostgreSQL calls, automatic recovery.
- **Prometheus Metrics** — Auto-instrumented request duration histograms, status code counters on /metrics endpoint.
- **Automatic Cache Layer** — goctl model -c generates Redis cache-aside pattern for primary key lookups. Use for short_code → URL lookups.
- **zRPC Service Communication** — gRPC with service discovery, P2C load balancing. Optional (current Dapr pub/sub works for async events). Add only if synchronous service calls needed (e.g., analytics queries from url-service).

**Defer (anti-features):**
- **etcd Service Discovery** — Overkill for 2-service learning project. Use direct connection mode.
- **Full OpenTelemetry Stack** — Requires Jaeger/Tempo backend. Enable TraceHandler middleware but defer production telemetry setup.
- **Auto-generated Swagger** — Adds goctl-swagger plugin complexity. Skip for learning project.

### Architecture Approach

Go-zero enforces 3-layer architecture: Handler (HTTP/gRPC I/O) → Logic (business logic) → Model (data access). ServiceContext provides dependency injection. This maps cleanly to the current delivery → usecase → repository pattern, but structure is code-generated rather than manual. Key difference: goctl-generated models don't enforce repository interfaces by default (methods are directly on structs), requiring wrapper layer for Clean Architecture compliance if desired.

**Major components:**
1. **URL Service (API)** — HTTP REST service defined in .api spec. Handles URL shortening, redirect, link management. Publishes ClickEvent to Kafka (fire-and-forget in goroutine). Generated with `goctl api go -api url.api -dir .`
2. **Analytics RPC Service** — gRPC service defined in .proto spec. Exposes GetClickCount, GetAnalyticsSummary endpoints for synchronous queries from URL service (if needed). Generated with `goctl rpc protoc analytics.proto --go_out=. --go-grpc_out=. --zrpc_out=.`
3. **Analytics Consumer** — Separate Kafka consumer binary. Implements kq.Queue.Consume(key, val) to process ClickEvent messages. Enriches with GeoIP/UA/Referer, stores to PostgreSQL. Runs independently from RPC service for scaling.

**Data flows:**
- **Shorten URL**: HTTP → Handler → Logic (validate, dedupe, insert) → Model → PostgreSQL → Response
- **Redirect**: HTTP → Handler → Logic (find URL) → Model → PostgreSQL → 302 Redirect + goroutine(Kafka publish)
- **Analytics Async**: Kafka → Consumer.Consume() → Enrich (GeoIP/UA/Referer) → Model.Insert() → PostgreSQL
- **Analytics Sync (optional)**: URL Service → zRPC Client → Analytics RPC → PostgreSQL → Response

**Migration mapping:**
- delivery/http/ → handler/ (generated, minimal edits)
- usecase/ → logic/ (manual business logic)
- repository/sqlite/ → model/ (goctl-generated from PostgreSQL)
- domain/ → split: DTOs in types/ (generated), entities in model/
- Dapr pub/sub → kq.Pusher (producer) + kq.Queue (consumer)

### Critical Pitfalls

**Top 5 pitfalls by severity:**

1. **Code Generation Overwriting Custom Logic** — Developers edit generated handler files, re-run goctl, lose all work. **Avoid:** NEVER edit files with "DO NOT EDIT" header. Put ALL business logic in logic/*.go (safe-to-edit). Commit generated files to git immediately to track overwrites. (Phase 1)

2. **.api File Syntax Errors Blocking Generation** — Subtle .api DSL errors prevent goctl from generating code with cryptic errors. Common: multiple service declarations, service blocks in imported files, generics. **Avoid:** Run `goctl api validate --api <file>` after every edit. ONE service declaration per main .api file. NO service blocks in imports (types only). NO generics. (Phase 1)

3. **SQLite→PostgreSQL Data Semantics Loss** — SQLite's loose typing allows string in INTEGER column, date as text. PostgreSQL rejects on migration. Auto-increment behavior differs (SQLite reuses ROWID, PostgreSQL SERIAL never reuses). **Avoid:** Audit SQLite data BEFORE migration with `SELECT typeof(id), id FROM urls WHERE typeof(id) != 'integer'`. Clean corrupted data. Use pgloader for schema conversion (handles boolean 0/1 → true/false). Test on production-like data, not empty schema. (Phase 2)

4. **Kafka Consumer Rebalancing Storms** — All consumer replicas restart simultaneously during deployment, triggering cascading rebalances. Processing stalls 30+ seconds. **Avoid:** Configure session timeout > max message processing time (increase from 10s to 30s+). Deploy with staggered restarts. Set kq Processors = partition count. Use incremental cooperative rebalancing (Kafka 2.4+). (Phase 3)

5. **Dapr→Kafka Delivery Semantics Change** — Dapr in-memory pub/sub is at-most-once (drops on failure). Kafka is at-least-once (retries, redelivers). Analytics receives duplicate click events, inflating counts. **Avoid:** Design for idempotency before Kafka cutover. Use `{short_code}:{timestamp}:{ip}` as deduplication key in Redis (TTL = retention period). Accept duplicates, deduplicate at query time or use Redis INCR with SET NX for unique counts. (Phase 3)

**Additional critical pitfalls:**
- **ServiceContext Bloat** — Every dependency stuffed into ServiceContext (GeoIP, UA parser, NanoID). Becomes 500-line god object. **Avoid:** ServiceContext only for infrastructure (DB, Redis, RPC clients). Stateless utilities as functions, not services. (Phase 2)
- **PostgreSQL Connection Pool Exhaustion** — Default settings don't match production. **Avoid:** Set MaxOpenConns = (CPU cores * 2) + 1, MaxIdleConns = MaxOpenConns / 2, ConnMaxLifetime = 5 minutes. Load test before deploy. (Phase 2)
- **zRPC Protobuf Schema Evolution** — Field number changes/reuse break clients silently. **Avoid:** NEVER change/reuse field numbers. Mark deleted fields as `reserved`. Deploy server schema changes before client. (Phase 3)

## Implications for Roadmap

Based on research, suggested phase structure prioritizes framework foundation before infrastructure migration. Go-zero's code generation is foundational — all other features build on generated structure. Database and messaging migrations are high-risk, high-complexity changes that should come after framework patterns are proven.

### Phase 1: Framework Foundation (go-zero Adoption)
**Rationale:** Establish go-zero code generation, handler/logic/model separation, and ServiceContext patterns before attempting database or messaging migrations. This is the lowest-risk starting point — pure code reorganization with familiar patterns.

**Delivers:**
- .api specs for url-service (shorten, redirect, list links)
- .proto spec for analytics-service (GetClickCount RPC)
- Generated handlers, routes, types
- Logic layer with migrated business logic
- ServiceContext with existing dependencies (SQLite, NanoID, GeoIP)
- RFC 7807 error handling via httpx.SetErrorHandler

**Addresses:**
- API Code Generation (table stakes)
- Handler/Logic/Model Separation (table stakes)
- Service Context DI (table stakes)
- Automatic Request Validation (table stakes)

**Avoids:**
- Code generation overwriting custom logic (establish separation immediately)
- .api syntax errors (validate before generation)

**Needs research:** NO — go-zero patterns are well-documented, official examples exist. Standard migration.

---

### Phase 2: Database Migration (SQLite → PostgreSQL)
**Rationale:** After framework patterns are proven, tackle database migration. This is high-complexity due to SQLite's loose typing requiring data audit. Separating from Phase 1 allows validation of go-zero patterns before introducing PostgreSQL complexity.

**Delivers:**
- PostgreSQL schema (migrated from SQLite with pgloader)
- goctl-generated models (replace sqlc)
- Audited and cleaned data migration
- Connection pool configuration (MaxOpenConns, MaxIdleConns)
- Repository wrapper layer (if Clean Architecture interfaces desired)

**Uses:**
- PostgreSQL 13+ (from STACK.md)
- jackc/pgx/v5 stdlib wrapper (from STACK.md)
- goctl model pg datasource (from STACK.md)

**Implements:**
- Model layer (data access component from ARCHITECTURE.md)

**Avoids:**
- SQLite data semantics loss (audit before migration)
- PostgreSQL connection pool exhaustion (configure pools during setup)
- ServiceContext bloat (establish boundaries: infrastructure only)

**Needs research:** MEDIUM — pgloader configuration, data type mapping, foreign key enforcement. Phase-specific research recommended.

---

### Phase 3: Messaging Migration (Dapr → Kafka)
**Rationale:** After database is stable, migrate messaging. This is high-risk due to delivery semantics change (at-most-once → at-least-once). Kafka requires idempotency design that Dapr didn't need. Separate from database migration to isolate risk.

**Delivers:**
- go-queue Kafka producer (kq.Pusher) in URL service
- go-queue Kafka consumer (kq.Queue) in analytics-service
- Kafka topic configuration (retention, replication, partitions)
- Idempotency logic (Redis deduplication)
- Consumer graceful shutdown (drain in-flight messages)

**Uses:**
- go-queue v1.2.2 (from STACK.md)
- segmentio/kafka-go v0.4.50 (from STACK.md)

**Implements:**
- Kafka Pub/Sub pattern (from ARCHITECTURE.md)
- Event flow: URL Service → Kafka → Analytics Consumer (from ARCHITECTURE.md)

**Avoids:**
- Kafka consumer rebalancing storms (configure session timeouts, staggered deploys)
- Dapr→Kafka delivery semantics change (implement idempotency first)
- Kafka topic misconfiguration (7-day retention, replication factor 2)
- Message loss (acks=all for critical events)

**Needs research:** MEDIUM — Kafka topic sizing, consumer group configuration, idempotency patterns. Phase-specific research recommended.

---

### Phase 4: Resilience & Observability (Production Hardening)
**Rationale:** After core infrastructure migration is complete and stable, enable go-zero's built-in production features. These are low-cost, configuration-driven enhancements. Deferring until foundation is stable allows focus on migration risks first.

**Delivers:**
- Prometheus metrics on /metrics endpoint
- Request timeout control (prevent hanging requests)
- MaxConns limiting (prevent resource exhaustion)
- Adaptive load shedding (CPU-based rate limiting)
- Adaptive circuit breaker (protect Kafka and PostgreSQL)
- OpenTelemetry tracing (dev-only, no backend)

**Addresses:**
- Built-in Rate Limiting (differentiator)
- Adaptive Circuit Breaker (differentiator)
- Prometheus Metrics (differentiator)

**Avoids:**
- Rate limiting per-instance (migrate to Redis-based global limiter)
- No authentication on zRPC (add interceptors if RPC exposed)

**Needs research:** NO — go-zero's built-in features are well-documented. Configuration-driven, no custom code.

---

### Phase 5: Optional Enhancements (Future Consideration)
**Rationale:** Features that provide value but aren't essential for migration completion. Evaluate based on project goals after Phase 4 stability.

**Delivers:**
- Automatic cache layer (goctl model -c with Redis)
- Concurrent request deduplication (singleflight for redirect hot paths)
- zRPC service communication (if synchronous analytics queries needed)
- Service discovery (etcd, if scaling beyond 2 services)

**Deferred because:**
- Current architecture works (Dapr pub/sub meets async event requirements)
- High complexity relative to value (cache layer requires full repository rewrite)
- Not required for migration goal (framework adoption complete in Phase 4)

---

### Phase Ordering Rationale

**Why framework first (Phase 1):**
- Go-zero's code generation is foundational — all features build on generated structure
- Lowest risk — code reorganization with no infrastructure changes
- Validates framework patterns before tackling database/messaging complexity
- Critical pitfalls (code generation discipline, .api syntax) must be addressed immediately

**Why database before messaging (Phase 2 before 3):**
- Database migration is data-preserving operation (audit, clean, migrate)
- Kafka migration changes delivery semantics (requires idempotency design)
- Separating allows isolated risk — if PostgreSQL migration fails, rollback without Kafka complexity
- Connection pooling and model generation patterns inform Kafka consumer setup

**Why resilience deferred to Phase 4:**
- Low-cost, configuration-driven features (no migration risk)
- Deferring allows focus on high-risk migrations (database, messaging) first
- Built-in features work best once infrastructure is stable
- Observability needs stable baseline to measure against

**How this avoids pitfalls:**
- Phase boundaries align with critical pitfall prevention phases (from PITFALLS.md mapping)
- Early phases address irreversible mistakes (code generation, .api syntax, data semantics)
- Later phases address operational issues (rebalancing, connection pools, observability)
- Staggered approach prevents compounding failures (PostgreSQL + Kafka migration simultaneously)

### Research Flags

**Phases needing deeper research during planning:**
- **Phase 2 (Database Migration):** pgloader configuration, SQLite type affinity audit queries, PostgreSQL schema conversion patterns, foreign key migration. Niche domain, needs `/gsd:research-phase` for data migration specifics.
- **Phase 3 (Kafka Integration):** Topic sizing calculations, consumer group configuration, idempotency pattern selection, rebalancing strategy tuning. Complex integration, needs phase research for production configuration.

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Framework Foundation):** Well-documented go-zero patterns, official tutorials exist, clear migration path from Chi/Dapr. Standard HTTP/RPC code generation.
- **Phase 4 (Resilience):** Configuration-driven built-in features, go-zero docs cover all aspects. No custom implementation needed.
- **Phase 5 (Optional):** Standard patterns (caching, service discovery), well-documented. Only research if actually implementing.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Official go-zero v1.10.0 release verified (Feb 15, 2025). GitHub releases, official docs, version compatibility matrix validated. PostgreSQL, Kafka, pgx versions cross-checked with go-zero requirements. |
| Features | MEDIUM | Framework features verified via official go-zero docs. Migration complexity estimates based on current codebase analysis. Application features unchanged (NanoID, GeoIP, redirect logic). Lower confidence on effort estimates for goctl model migration (no direct sqlc→goctl migration guide found). |
| Architecture | HIGH | Go-zero's Handler→Logic→Model pattern well-documented with official examples. Monorepo structure validated via community articles and go.work patterns. Current v1.0 codebase analyzed to confirm mapping (delivery→handler, usecase→logic, repository→model). Kafka consumer separation pattern verified in go-queue docs. |
| Pitfalls | MEDIUM | Critical pitfalls sourced from official go-zero issues (GitHub), Kafka rebalancing guides, PostgreSQL migration articles. Code generation pitfalls verified via framework docs and community reports. Lower confidence on SQLite data corruption frequency (no statistics found, based on SQLite type affinity behavior). |

**Overall confidence:** HIGH

Research is comprehensive for framework migration path (Stack, Architecture). Feature migration is straightforward (business logic unchanged). Moderate uncertainty on data migration complexity (Phase 2) and Kafka tuning (Phase 3) — phase-specific research will address these gaps.

### Gaps to Address

**Data migration validation (Phase 2):**
- No direct sqlc→goctl model migration guide found. Need to research query compatibility, generated code differences, and repository interface preservation strategies during Phase 2 planning.
- SQLite data audit queries are generic. Need to inspect actual schema (urls, clicks tables) to identify type affinity risks specific to this project during execution.
- pgloader configuration for go-shortener schema needs testing on snapshot data before production migration.

**Kafka production configuration (Phase 3):**
- Topic sizing (partitions, retention) depends on production traffic patterns not specified in research. Need metrics from current Dapr deployment to inform Kafka configuration.
- Idempotency key selection (`{short_code}:{timestamp}:{ip}`) is proposed but not validated. Need to confirm this prevents duplicates without false positives during Phase 3 planning.
- Consumer rebalancing impact depends on deployment frequency and replica count. Need production deployment patterns to configure session timeouts appropriately.

**Performance baselines (Phase 4):**
- No current performance metrics (request latency, throughput, resource usage) found in research. Need to establish baseline with Chi/Dapr stack before migration to measure go-zero impact.
- Rate limiting behavioral change (per-IP → CPU-based adaptive) may require hybrid approach (keep per-IP + add adaptive). Need user quota requirements to validate.

**How to handle during planning/execution:**
- Phase 2: Run `/gsd:research-phase` for PostgreSQL migration specifics (pgloader config, data audit queries, goctl model patterns)
- Phase 3: Run `/gsd:research-phase` for Kafka production configuration (topic sizing, consumer tuning, idempotency validation)
- Phase 4: Establish performance testing plan in execution doc, capture Chi/Dapr baseline metrics before migration
- All phases: Use .planning/STATE.md to track discoveries and decisions as gaps are resolved

## Sources

### Primary (HIGH confidence)

**Stack research:**
- [go-zero GitHub Releases](https://github.com/zeromicro/go-zero/releases) — v1.10.0 verified (Feb 15, 2025)
- [goctl Installation](https://go-zero.dev/en/docs/tasks/installation/goctl) — Installation methods
- [go-zero Kafka Integration](https://go-zero.dev/en/docs/tutorials/message-queue/kafka) — go-queue usage
- [jackc/pgx v5 Docs](https://pkg.go.dev/github.com/jackc/pgx/v5) — v5.8.0 verified (Dec 26, 2025)
- [goctl Model Generation](https://go-zero.dev/en/docs/tutorials/cli/model) — Database code generation

**Architecture research:**
- [Project Structure | go-zero Documentation](https://go-zero.dev/en/docs/concepts/layout) — Official project layout
- [Framework Overview | go-zero Documentation](https://go-zero.dev/en/docs/concepts/overview) — Architecture and components
- [API Syntax | go-zero Documentation](https://go-zero.dev/en/docs/tasks/dsl/api) — .api file DSL specification
- [goctl RPC | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/cli/rpc) — RPC service generation
- [GitHub - zeromicro/go-queue](https://github.com/zeromicro/go-queue) — Official Kafka integration

### Secondary (MEDIUM confidence)

**Features research:**
- [Middleware | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/http/server/middleware) — Service governance features
- [Circuit Breaker | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/component/breaker/) — Resilience patterns
- [Cache Management | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/mysql/cache) — Auto-caching with goctl model -c
- [Parameter Rules | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/api/parameter) — .api validation tags

**Pitfalls research:**
- [How to migrate from SQLite to PostgreSQL](https://render.com/articles/how-to-migrate-from-sqlite-to-postgresql) — Data semantics, type affinity
- [Kafka Rebalancing: Triggers, Effects, and Mitigation](https://www.redpanda.com/guides/kafka-performance-kafka-rebalancing) — Consumer rebalancing strategies
- [Protocol Buffers Schema Evolution Guide](https://jstontoable.org/blog/protobuf/protobuf-schema-evolution) — Protobuf compatibility rules
- [How to Implement Connection Pooling in Go for PostgreSQL](https://oneuptime.com/blog/post/2026-01-07-go-postgresql-connection-pooling/view) — Pool configuration

### Tertiary (LOW confidence)

- [I've Tried Many Go Frameworks. Here's Why I Finally Chose This One](https://medium.com/@g.zhufuyi/ive-tried-many-go-frameworks-here-s-why-i-finally-chose-this-one-a73ad2636a50) — Framework comparison (opinion piece, not authoritative)
- Community articles on ServiceContext patterns (not official docs, needs validation)

---
*Research completed: 2026-02-15*
*Ready for roadmap: yes*
