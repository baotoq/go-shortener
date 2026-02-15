# Feature Research: go-zero Framework Adoption

**Domain:** Microservices Framework Migration (URL Shortener: Current Stack → go-zero)
**Researched:** 2026-02-15
**Confidence:** MEDIUM

## Feature Landscape

This research categorizes go-zero framework features by adoption priority for rebuilding the existing URL shortener. **Focus is on framework capabilities, not application features** (URL shortening logic remains the same).

### Table Stakes (Framework Features to Adopt)

Framework capabilities essential for feature parity with existing implementation.

| Framework Feature | Why Expected | Complexity | Notes |
|-------------------|--------------|------------|-------|
| **API Code Generation (.api → Go)** | Core go-zero value proposition - eliminates manual routing boilerplate | LOW | `goctl api` generates handlers, routes, types from .api spec. Replaces manual HTTP handler setup |
| **Handler/Logic/Model Separation** | go-zero enforces Clean Architecture automatically | LOW | Auto-generated structure: handlers (HTTP), logic (business), models (data). Maps to current delivery/usecase/repository pattern |
| **Automatic Request Validation** | Built-in validation via .api tags (range, options, optional) | LOW | Replaces manual validation logic. Use `.api` tag rules: `json:"age,range=[0:120]"` |
| **Automatic JSON Response Writing** | `httpx.OkJson(w, resp)` handles serialization automatically | LOW | Simplifies current manual `json.Marshal` + `w.Write` patterns |
| **Error Response Standardization** | `httpx.Error()` + custom `httpx.SetErrorHandler()` for RFC 7807 compatibility | MEDIUM | Existing RFC 7807 Problem Details can be integrated via custom error handler |
| **Service Context (Dependency Injection)** | `servicecontext.go` for DB/Redis/dependencies | LOW | Replaces manual dependency wiring. Auto-generated, pass deps via Config |
| **Configuration Management** | Config struct auto-generated from .yaml, automatic validation | LOW | Replaces manual env var parsing. Supports validation via struct tags |
| **Service Groups (.api routing)** | `@server(group: links)` organizes handlers by domain (links, analytics) | LOW | Maps to current service separation (url-service, analytics-service) |

### Differentiators (go-zero-Specific Enhancements)

Features that go-zero enables without custom implementation. These are **new capabilities** unlocked by framework adoption.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Built-in Rate Limiting** | Token bucket rate limiter via middleware - replaces custom in-memory limiter | LOW | Current project has custom rate limiter. go-zero provides `SheddingHandler` middleware (adaptive load shedding based on CPU) |
| **Adaptive Circuit Breaker** | Google SRE breaker algorithm with 10-second sliding window, auto-recovery | MEDIUM | No custom implementation needed. Protects downstream calls (Kafka, PostgreSQL). Default threshold: automatic based on error rate |
| **Adaptive Load Shedding** | CPU-based request rejection (default: 5-second window, 50 buckets) | MEDIUM | Protects service under high load. Current project lacks this. Auto-rejects requests when CPU threshold exceeded |
| **Prometheus Metrics (Auto-Instrumentation)** | `PrometheusHandler` middleware exposes `/metrics` with request duration histograms, status code counters | LOW-MEDIUM | Requires config (`Prometheus: {Host, Port, Path}`). Current project lacks built-in metrics |
| **OpenTelemetry Tracing** | `TraceHandler` middleware for distributed tracing with auto-context propagation | MEDIUM | Current project lacks tracing. go-zero embeds trace context in all requests |
| **Automatic Cache Layer (goctl model -c)** | Redis caching auto-generated for primary key/unique index queries | MEDIUM | Current project: manual cache logic. go-zero: `sqlc.CachedConn` wraps queries with cache-aside pattern |
| **Concurrent Request Deduplication** | Built-in singleflight prevents cache stampede for identical concurrent queries | LOW | No current implementation. Useful for redirect hot paths (same short code → many concurrent requests) |
| **PostgreSQL Model Generation** | `goctl model pg` generates CRUD from database schema (datasource mode) | MEDIUM | Replaces sqlc. Compare: sqlc uses SQL files, goctl uses DB introspection. Both generate type-safe code |
| **zRPC Service Communication** | gRPC with service discovery (etcd/k8s), P2C load balancing, built-in interceptors | MEDIUM-HIGH | Current: Dapr pub/sub (fire-and-forget). zRPC: synchronous service calls with built-in resilience |
| **JWT Middleware** | `.api` syntax for JWT auth (`@server(jwt: Auth)`), auto-validates tokens on protected routes | LOW | Current project lacks auth. go-zero: declare in .api, middleware auto-generated. Manual token generation still needed |
| **Request Timeout Control** | `TimeoutHandler` middleware with cascade timeout propagation | LOW | Current project: no timeout handling. go-zero: configure per-route or globally |
| **MaxConns Limiting** | Connection-level concurrency control (default: 10,000 concurrent requests) | LOW | Prevents resource exhaustion. Current project: relies on OS limits |
| **Bloom Filter (Redis-backed)** | `github.com/zeromicro/go-zero/core/bloom` for duplicate detection, cache warming | MEDIUM | Use case: detect duplicate short code generation attempts, prevent cache penetration attacks |

### Anti-Features (go-zero Features to Skip)

Framework capabilities that don't align with URL shortener requirements or conflict with existing decisions.

| Feature | Why Tempting | Why Problematic | Alternative |
|---------|--------------|-----------------|-------------|
| **Mixed HTTP + gRPC in Same Service** | "Serve both protocols from one binary" | go-zero doesn't support this. Requires separate services. Violates single responsibility | Keep services pure: HTTP for url-service, zRPC for inter-service calls if needed |
| **Auto-generated Swagger from .api** | "Free API docs from code generation" | Adds tooling complexity (`goctl-swagger` plugin). Learning project doesn't need public docs | Skip. Use .api files as internal documentation |
| **Cache for All Queries** | "goctl model -c caches everything" | Only supports primary key + unique single-record index. Current project has complex filters (cursor pagination, search) | Use cache selectively: short code → URL lookups only. Bypass cache for list/search queries |
| **etcd Service Discovery** | "Production-grade service registry" | Overkill for 2-service learning project. Adds deployment complexity | Use direct connection mode for zRPC (if adopted). Add etcd only if scaling beyond 2 services |
| **Full OpenTelemetry Stack** | "Distributed tracing for everything" | Requires Jaeger/Tempo backend, adds operational overhead | Enable `TraceHandler` middleware but export to local development tools only (not production setup) |
| **goctl model DDL Generation** | "Generate DB schema from .sql files" | Current project has embedded migrations (proven approach). goctl MySQL DDL gen doesn't support PostgreSQL schema files | Keep existing migration approach (embed .sql). Use `goctl model pg datasource` to generate models from existing DB |
| **Kubernetes-Native Service Discovery** | "Use k8s service DNS" | Learning project runs locally with `make run-all`. K8s adds deployment complexity | Defer until deployment phase. Not a framework feature concern for development |
| **Custom Middleware for Everything** | "Wrap every handler with custom logic" | go-zero auto-enables most middleware (Recover, Metrics, Trace). Custom middleware adds indirection | Use built-in middleware. Add custom only for domain-specific needs (e.g., analytics event publishing) |

## Feature Dependencies

**Framework features have dependencies between each other and on migration decisions:**

```
API Code Generation (.api spec)
    └──requires──> Handler/Logic/Model separation (auto-generated structure)
    └──enables──> Service Groups (routing organization)
    └──enables──> JWT Middleware (declared in .api @server blocks)
    └──enables──> Automatic Request Validation (tag rules in type definitions)

Handler/Logic/Model Separation
    └──requires──> Service Context (DI container)
                       └──requires──> Configuration Management (Config struct)

Automatic Cache Layer (goctl model -c)
    └──requires──> PostgreSQL Model Generation (must use goctl model)
    └──requires──> Redis dependency in Service Context
    └──conflicts with──> Complex Queries (cache only supports PK/unique index)

zRPC Service Communication
    └──requires──> Proto File Definitions (.proto specs)
    └──requires──> Service Discovery Strategy (direct/etcd/k8s decision)
    └──optional──> Circuit Breaker (built-in interceptor)
    └──optional──> Load Shedding (built-in interceptor)

Built-in Rate Limiting (Shedding)
    └──independent from──> Application-level rate limiting
    └──based on──> CPU load (not request count)

Prometheus Metrics
    └──independent from──> Other features (cross-cutting concern)
    └──requires──> Prometheus scraper (external infrastructure)

OpenTelemetry Tracing
    └──independent from──> Other features (cross-cutting concern)
    └──requires──> Trace backend (Jaeger/Tempo - external infrastructure)

Bloom Filter
    └──requires──> Redis dependency
    └──independent from──> Automatic Cache Layer (separate use case)
```

### Dependency Notes

- **API Code Generation is the foundation:** Must adopt this first. All other framework features build on generated structure.
- **Service Context + Configuration are coupled:** Config struct auto-generated, injected into ServiceContext constructor.
- **Automatic Cache Layer locks you into goctl model:** If adopted, cannot use sqlc. Requires full migration to goctl-generated models.
- **zRPC is optional:** Current Dapr pub/sub works for async events. Only adopt zRPC if synchronous service calls are needed (e.g., analytics-service queries from url-service).
- **Resilience features (Circuit Breaker, Load Shedding) are independent:** Can enable/disable per service via config.
- **Observability features (Metrics, Tracing) are independent:** Can enable one without the other.

## Framework Feature Mapping to Existing Capabilities

How go-zero features map to current URL shortener implementation:

| Current Implementation | Existing Pattern | go-zero Replacement | Migration Complexity | Notes |
|------------------------|------------------|---------------------|----------------------|-------|
| **HTTP Routing** | Manual `chi.Router` with route registration | `.api` spec + `goctl api` code gen | LOW | From manual to declarative. `.api` replaces `delivery/http/routes.go` |
| **Request Validation** | Manual validation in handlers (check required fields, ranges) | `.api` tag rules (`optional`, `range`, `options`) | LOW | Validation moves from handler code to .api type definitions |
| **JSON Response Writing** | Manual `json.Marshal` + `w.Write` + status codes | `httpx.OkJson(w, resp)` auto-writes | LOW | Reduces boilerplate, errors less likely |
| **Error Handling** | Custom RFC 7807 `problemdetails` package | Custom `httpx.SetErrorHandler()` integration | MEDIUM | Keep RFC 7807 format, integrate with go-zero's error flow |
| **Dependency Injection** | Manual constructor injection (repository → usecase → handler) | `ServiceContext` struct auto-generated | MEDIUM | Pattern similar, but structure auto-generated. Need to populate ServiceContext with current dependencies |
| **Rate Limiting** | Custom in-memory token bucket (resets on restart) | go-zero `SheddingHandler` (CPU-based adaptive) | MEDIUM | **Behavioral change:** Current limits requests/second per IP. go-zero limits based on CPU load (protects service, not per-user quota) |
| **Database Layer** | sqlc-generated models from `.sql` files | `goctl model pg datasource` from DB schema | MEDIUM-HIGH | **Tool swap:** sqlc → goctl. Both type-safe. sqlc: SQL-first. goctl: schema-first + optional cache |
| **Repository Pattern** | Interfaces defined in usecase, implemented in repository/sqlite | goctl-generated models (no interfaces, direct struct methods) | HIGH | **Architecture shift:** go-zero doesn't enforce repository interfaces. Generated models have methods like `FindOne`, `Insert`. Need custom wrapper for Clean Architecture compliance |
| **Pub/Sub Events** | Dapr SDK (`daprClient.PublishEvent`) | go-queue (Kafka) or keep Dapr | MEDIUM | Dapr works, but go-queue is go-zero-native. Kafka requires infrastructure. Recommendation: keep Dapr for compatibility |
| **Configuration** | Manual `os.Getenv` + defaults | Config struct auto-generated, loaded from .yaml | LOW | From imperative to declarative. Config validation via struct tags |
| **Middleware** | Custom middleware (`ratelimiter.Middleware`, `logging.Middleware`) | Built-in middleware (`SheddingHandler`, `LogHandler`, `PrometheusHandler`) | LOW-MEDIUM | Replace custom with built-in where overlap exists. Keep domain-specific (e.g., click event publishing) |
| **GeoIP/UA Enrichment** | Custom logic in analytics-service usecase | Stays as custom logic (domain-specific, not framework concern) | NO CHANGE | go-zero doesn't affect this. Remains in analytics logic layer |
| **NanoID Generation** | Custom `pkg/nanoid` package | Stays as custom package | NO CHANGE | go-zero doesn't provide short code generation. Domain logic remains unchanged |
| **Caching** | None (considered, not implemented) | `goctl model -c` auto-generates Redis cache-aside | MEDIUM | **New capability:** Add caching for `short_code → url` lookups. Requires Redis, goctl model migration |

### Migration Complexity Legend

- **LOW:** Direct replacement, minimal code changes, pattern remains similar
- **MEDIUM:** Requires refactoring, behavioral changes, or new dependencies
- **HIGH:** Significant architecture shift, testing impact, or decision point (keep vs replace)
- **NO CHANGE:** Feature stays in application layer, unaffected by framework

## MVP Definition (Migration Phases)

### Phase 1: Foundation (Launch With)

Minimum go-zero adoption to achieve feature parity.

- [x] **API Code Generation** — Core go-zero value. Migrate .api specs for url-service + analytics-service
- [x] **Handler/Logic/Model Separation** — Accept auto-generated structure, map existing usecase logic to logic layer
- [x] **Service Context (DI)** — Populate with existing dependencies (DB, NanoID, GeoIP DB)
- [x] **Automatic Request Validation** — Migrate validation logic to .api tag rules
- [x] **Automatic JSON Response Writing** — Replace manual marshaling with `httpx.OkJson`
- [x] **Error Response Standardization** — Integrate RFC 7807 via `httpx.SetErrorHandler`
- [x] **Configuration Management** — Migrate env vars to Config struct + .yaml
- [x] **Service Groups** — Organize routes (`group: links`, `group: analytics`)

**Migration Goal:** Rebuild existing HTTP endpoints using go-zero conventions. No new features, pure framework migration.

**Complexity:** MEDIUM — Mostly code reorganization, patterns remain similar.

### Phase 2: Resilience + Observability (Add After Foundation)

Enable built-in go-zero features that enhance existing capabilities.

- [ ] **Prometheus Metrics** — Enable `/metrics` endpoint for request duration, status codes
- [ ] **Request Timeout Control** — Add timeout middleware to prevent hanging requests
- [ ] **MaxConns Limiting** — Set connection limits to prevent resource exhaustion
- [ ] **Built-in Rate Limiting (Load Shedding)** — Enable CPU-based adaptive shedding (supplement existing per-IP limiter)
- [ ] **Adaptive Circuit Breaker** — Protect Kafka publisher, PostgreSQL queries
- [ ] **OpenTelemetry Tracing** — Enable trace context propagation (dev-only, no backend requirement)

**Migration Goal:** Harden services with go-zero's built-in resilience patterns. Observability baseline.

**Trigger:** Foundation stable, ready to add production-readiness features.

**Complexity:** LOW-MEDIUM — Configuration-driven, minimal code changes.

### Phase 3: Data Layer Migration (Future Consideration)

Database/cache layer migration. **High-impact decision point.**

- [ ] **PostgreSQL Model Generation** — Migrate from sqlc to `goctl model pg datasource`
- [ ] **Automatic Cache Layer** — Enable Redis caching for short code lookups (`goctl model -c`)
- [ ] **Repository Interface Abstraction** — Wrap goctl-generated models to preserve Clean Architecture (if desired)
- [ ] **Concurrent Request Deduplication** — Add singleflight for redirect hot paths

**Migration Goal:** Leverage go-zero's data layer features (caching, model generation). Architecture decision: embrace goctl models or keep sqlc + add manual cache.

**Defer Reason:** High complexity, architectural shift. Evaluate if cache-aside pattern justifies migration from sqlc.

**Complexity:** HIGH — Requires full repository layer rewrite, extensive testing.

### Phase 4: Service Communication (Optional)

Only if synchronous inter-service calls are needed.

- [ ] **zRPC Proto Definitions** — Define .proto for analytics queries (if url-service needs to query analytics-service)
- [ ] **zRPC Service Implementation** — Migrate Dapr pub/sub to zRPC for synchronous calls
- [ ] **Service Discovery** — Choose direct connection (simple) vs etcd (scalable)

**Migration Goal:** Enable request/response patterns between services (beyond fire-and-forget events).

**Defer Reason:** Current Dapr pub/sub meets requirements. Add only if synchronous queries are needed (e.g., "show analytics for short code" from url-service).

**Complexity:** MEDIUM-HIGH — New communication pattern, proto definitions, testing.

## Feature Prioritization Matrix

Framework features ranked by migration value vs. implementation cost.

| Framework Feature | Migration Value | Implementation Cost | Priority | Phase |
|-------------------|-----------------|---------------------|----------|-------|
| API Code Generation | HIGH | MEDIUM | P1 | Foundation |
| Handler/Logic/Model Separation | HIGH | MEDIUM | P1 | Foundation |
| Service Context (DI) | HIGH | LOW | P1 | Foundation |
| Automatic Request Validation | HIGH | LOW | P1 | Foundation |
| Automatic JSON Response Writing | HIGH | LOW | P1 | Foundation |
| Error Response Standardization | HIGH | MEDIUM | P1 | Foundation |
| Configuration Management | HIGH | LOW | P1 | Foundation |
| Service Groups | MEDIUM | LOW | P1 | Foundation |
| Prometheus Metrics | MEDIUM | LOW | P2 | Resilience |
| Request Timeout Control | MEDIUM | LOW | P2 | Resilience |
| MaxConns Limiting | MEDIUM | LOW | P2 | Resilience |
| Built-in Rate Limiting | MEDIUM | LOW | P2 | Resilience |
| Adaptive Circuit Breaker | MEDIUM | LOW | P2 | Resilience |
| OpenTelemetry Tracing | LOW | MEDIUM | P2 | Resilience |
| PostgreSQL Model Generation | HIGH | HIGH | P3 | Data Layer |
| Automatic Cache Layer | MEDIUM | MEDIUM | P3 | Data Layer |
| Repository Interface Abstraction | LOW | HIGH | P3 | Data Layer |
| Concurrent Request Deduplication | MEDIUM | LOW | P3 | Data Layer |
| zRPC Proto Definitions | LOW | MEDIUM | P4 | Service Comm |
| zRPC Service Implementation | LOW | HIGH | P4 | Service Comm |
| Service Discovery (etcd) | LOW | MEDIUM | P4 | Service Comm |

**Priority key:**
- **P1 (Foundation):** Must have for migration completion - core framework adoption
- **P2 (Resilience):** Should have - low-cost enhancements to production-readiness
- **P3 (Data Layer):** High-value but high-cost - defer until foundation stable
- **P4 (Service Comm):** Nice to have - only if synchronous inter-service calls needed

## Framework Comparison: Current Stack vs. go-zero

What changes when migrating to go-zero?

| Aspect | Current Stack | go-zero Stack | Impact |
|--------|---------------|---------------|--------|
| **Routing** | Manual (chi router) | Declarative (.api spec) | **Benefit:** Less boilerplate, single source of truth |
| **Code Organization** | Custom (delivery/usecase/repository) | Enforced (handler/logic/model) | **Tradeoff:** Lose flexibility, gain consistency |
| **Validation** | Manual (handler code) | Declarative (.api tags) | **Benefit:** Validation logic closer to type definitions |
| **Database Code** | sqlc (SQL → Go) | goctl (Schema → Go) | **Tradeoff:** sqlc: SQL-first, goctl: schema-first + cache. Similar type safety |
| **Dependency Injection** | Manual constructor injection | ServiceContext struct | **Neutral:** Pattern similar, structure auto-generated |
| **Error Handling** | Custom RFC 7807 package | Custom httpx.SetErrorHandler | **Neutral:** Keep existing format, integrate with go-zero flow |
| **Rate Limiting** | Custom (request count per IP) | Built-in (CPU-based adaptive) | **Tradeoff:** Lose per-user quota, gain service-level protection |
| **Pub/Sub** | Dapr SDK (framework-agnostic) | go-queue (Kafka-native) | **Tradeoff:** Dapr: simple, local dev. Kafka: production-grade, complex setup |
| **Caching** | None | Built-in (goctl model -c) | **Benefit:** Cache-aside pattern auto-generated |
| **Resilience** | None (manual if needed) | Built-in (breaker, shedding, timeout) | **Benefit:** Production patterns out-of-box |
| **Observability** | None (manual logging) | Built-in (Prometheus, OpenTelemetry) | **Benefit:** Metrics + tracing with config, no code |
| **Service Communication** | Dapr pub/sub (async-only) | zRPC (sync) + go-queue (async) | **Benefit:** Enables request/response patterns if needed |

### Key Architectural Shifts

1. **Declarative → Imperative:** Routing, validation, config move from code to specs (.api, .yaml)
2. **SQL-first → Schema-first:** Database models generated from schema introspection (not SQL files)
3. **Custom → Built-in:** Resilience, observability shift from "build when needed" to "configure built-in"
4. **Clean Architecture → go-zero Conventions:** Repository interfaces optional (goctl models use direct methods)

### Alignment with Project Goals

**Project is a learning exercise:** go-zero provides production patterns (breaker, shedding, metrics) without custom implementation. **Good for learning microservice patterns.**

**Existing features work:** URL shortening, redirect, analytics logic unchanged. Framework change is **transparent to business logic.**

**Complexity increase:** More tooling (goctl), new conventions (.api syntax), potential infrastructure (Redis for cache, Kafka for events). **Tradeoff:** Built-in features vs. operational overhead.

## Existing Features Preserved

Application features that **remain unchanged** regardless of framework migration:

- [x] **NanoID Short Code Generation** — Domain logic, stays in custom package
- [x] **302 Redirect with Click Events** — Logic layer, go-zero doesn't affect redirect behavior
- [x] **GeoIP Lookup** — Custom enrichment logic in analytics-service
- [x] **User-Agent Parsing** — Custom enrichment logic in analytics-service
- [x] **Referer Tracking** — Custom enrichment logic in analytics-service
- [x] **Cursor Pagination** — Query logic, remains in repository (sqlc or goctl-generated)
- [x] **Filtering by Status/Domain** — Query logic, framework-agnostic
- [x] **Search by URL** — Query logic, framework-agnostic
- [x] **Custom Alias Validation** — Business rule, stays in logic layer
- [x] **Collision Retry (5 attempts)** — Domain logic, framework-agnostic
- [x] **RFC 7807 Error Format** — Preserved via `httpx.SetErrorHandler` integration

**Key Insight:** Framework migration affects **how features are structured and delivered**, not **what features exist**. Business logic, domain rules, and data enrichment remain in application layer.

## Sources

### go-zero Framework Overview & Features
- [GitHub - zeromicro/go-zero: A cloud-native Go microservices framework](https://github.com/zeromicro/go-zero)
- [Framework Overview | go-zero Documentation](https://go-zero.dev/en/docs/concepts/overview)
- [go-zero | go-zero Documentation](https://go-zero.dev/en/)

### Code Generation (goctl)
- [API specification | go-zero Documentation](https://go-zero.dev/en/docs/tutorials)
- [goctl api | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/cli/api)
- [goctl model | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/cli/model)

### Middleware & Service Governance
- [Middleware | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/http/server/middleware)
- [Circuit Breaker | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/component/breaker/)
- [Breaker | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/service/governance/breaker)
- [Load Shedding for Busy Services | by Kevin Wan | CodeX | Medium](https://medium.com/codex/load-shedding-governance-on-busy-services-38193ed4888b)

### Parameter Validation
- [Parameter Rules | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/api/parameter)
- [Auto Validation | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/go-zero/configuration/auto-validation)

### Error Handling
- [Error processing | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/http/server/error)

### Cache Management
- [Cache Management | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/mysql/cache)
- [Build Model | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/build-tool/model/)

### Service Communication (zRPC)
- [Enterprise RPC framework zRPC | go-zero](https://go-zero.dev/docs/blog/showcase/zrpc/)
- [goctl rpc | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/cli/rpc)
- [Discovery | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/component/discovery/)
- [Implementing Service Discovery for Microservices | by Kevin Wan | Dev Genius](https://medium.com/codex/implementing-service-discovery-for-microservices-df737e012bc2)

### Pub/Sub (go-queue)
- [GitHub - zeromicro/go-queue: Kafka, Beanstalkd Pub/Sub framework](https://github.com/zeromicro/go-queue)

### JWT Authentication
- [JWT Certification | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/http/server/jwt)
- [JWT | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/api/jwt)

### Service Groups
- [Service Group | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/api/route/group)

### Observability (Prometheus, OpenTelemetry)
- [Metric | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/component/metric/)
- [Go-Zero | Grafana Labs](https://grafana.com/grafana/dashboards/19909-go-zero/)
- [Go Observability Stack: Prometheus, Grafana, and OpenTelemetry](https://dasroot.net/posts/2026/02/go-observability-stack-prometheus-grafana-opentelemetry/)

### Bloom Filter
- [bloom package - github.com/zeromicro/go-zero/core/bloom](https://pkg.go.dev/github.com/zeromicro/go-zero/core/bloom)
- [Efficient Cache Design with Bloom Filters in Go | by Leapcell | Medium](https://leapcell.medium.com/efficient-cache-design-with-bloom-filters-in-go-acbbcd053fab)

### Singleflight (Request Deduplication)
- [Singleflight in Go: A Clean Solution to Cache Stampede | by Dilan Dashintha | The PickMe Engineering Blog | Medium](https://medium.com/pickme-engineering-blog/singleflight-in-go-a-clean-solution-to-cache-stampede-02acaf5818e3)
- [Go Singleflight Melts in Your Code, Not in Your DB](https://victoriametrics.com/blog/go-singleflight/)

### Framework Comparison
- [I've Tried Many Go Frameworks. Here's Why I Finally Chose This One | by gvison | Medium](https://medium.com/@g.zhufuyi/ive-tried-many-go-frameworks-here-s-why-i-finally-chose-this-one-a73ad2636a50)
- [Hardcore Multi-Dimensional Comparison: Kratos vs Go-Zero vs GoFrame vs Sponge | by gvison | Medium](https://medium.com/@g.zhufuyi/hardcore-multi-dimensional-comparison-kratos-vs-go-zero-vs-goframe-vs-sponge-which-go-8c168d75fe36)
- [Top 18 Golang Web Frameworks to Use in 2026](https://www.bacancytechnology.com/blog/golang-web-frameworks)
- [go-zero - Reviews, Pros & Cons | Companies using go-zero](https://www.stackshare.io/go-zero)

### PostgreSQL & Database Code Generation
- [goctl model | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/cli/model)
- [How to Use sqlc for Type-Safe Database Access in Go](https://oneuptime.com/blog/post/2026-01-07-go-sqlc-type-safe-database/view)
- [Comparing the best Go ORMs (2026)](https://encore.cloud/resources/go-orms)

### Dependency Injection Patterns
- [Dependency Injection in Go: Patterns & Best Practices | by Rost Glukhov | Medium](https://medium.com/@rosgluk/dependency-injection-in-go-patterns-best-practices-5e5136df5357)
- [How to Implement Dependency Injection in Go - FreeCodeCamp](https://www.freecodecamp.org/news/how-to-use-dependency-injection-in-go/)

---

*Feature research for: go-zero Framework Adoption (URL Shortener Migration)*
*Researched: 2026-02-15*
*Confidence: MEDIUM (WebSearch verified with official go-zero docs, no Context7 availability for framework-specific validation)*
