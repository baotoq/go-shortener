# Project Research Summary

**Project:** Go URL Shortener with Dapr Microservices
**Domain:** URL Shortener with distributed microservices architecture
**Researched:** 2026-02-14
**Confidence:** HIGH

## Executive Summary

This project implements a URL shortener service using Go 1.26 and Dapr as the distributed application runtime. Research shows URL shorteners are deceptively simple on the surface but require careful architecture to handle the extreme read/write imbalance (100:1 ratio) and scale effectively. The recommended approach uses a two-service microservices architecture: URL Service for shortening and redirection, Analytics Service for click tracking via asynchronous pub/sub events.

The critical insight from research is that this domain demands read-path optimization from day one. Most URL shortener implementations fail when they treat it as simple CRUD without caching strategy, use inappropriate databases for concurrent access (SQLite pitfall), or add synchronous analytics that slow redirects. The Dapr sidecar pattern provides excellent infrastructure abstraction (state management, pub/sub, service invocation) but introduces operational complexity and specific pitfalls around resource limits, health probes, and retry configurations.

Key risks include SQLite concurrent write corruption (must migrate before multi-instance deployment), pub/sub retry explosion from double retry behavior, and context propagation failures that break timeout cascades. These are all preventable with proper patterns established in Phase 1 (Foundation) and validated through integration testing with real Dapr sidecars, not just mocks. The recommended stack (Go 1.26, Dapr SDK v1.13.0, Chi router, modernc.org/sqlite for dev, slog for logging) provides a production-quality foundation while remaining simple enough for a learning project.

## Key Findings

### Recommended Stack

Go 1.26 with Dapr provides an excellent foundation for microservices with strong concurrency primitives and minimal dependencies. The stack prioritizes stdlib components and pure Go libraries to avoid CGO complexity and ensure single-binary deployments.

**Core technologies:**
- **Go 1.26**: Latest stable release with excellent concurrency model, strong stdlib reduces external dependencies
- **Dapr Go SDK v1.13.0**: Official SDK for pub/sub and service invocation, abstracts infrastructure concerns (state, messaging, observability)
- **modernc.org/sqlite**: Pure Go SQLite driver (no CGO), perfect for development and single-instance deployments, must migrate to PostgreSQL for production scale
- **Chi v5**: Lightweight HTTP router with excellent middleware ecosystem, stdlib-compatible, preferred over Gin for idiomatic Go
- **slog (stdlib)**: Structured logging built into Go 1.21+, zero dependencies, sufficient performance for this project
- **validator/v10**: Industry standard struct validation, reduces boilerplate
- **golangci-lint v2.9.0**: Latest linting aggregator, essential for code quality
- **distroless containers**: Minimal attack surface (2MB base image vs 800MB golang:1.26), production best practice

**Critical version notes:**
- Dapr SDK v1.13.0 supports Go 1.21+, fully tested with 1.26
- modernc.org/sqlite requires Go 1.22+ for stdlib features
- Must build with `CGO_ENABLED=0` for distroless containers

### Expected Features

URL shorteners have well-established feature expectations. Users differentiate between table stakes (expected in all shorteners) and competitive differentiators (advanced features).

**Must have (table stakes):**
- Basic URL shortening with auto-generated codes
- Short link redirection (302 temporary redirects, not 301)
- Click tracking (total click counts minimum)
- Custom aliases (human-readable slugs like /promo2026)
- Link management API (CRUD operations)
- Error handling (404 for missing links, validation)
- Rate limiting (prevent abuse from day one)

**Should have (competitive advantage):**
- Real-time analytics (Dapr pub/sub enables this, competitive advantage vs Bitly's delayed analytics)
- Geo-location tracking (country-level from IP address)
- Device/browser detection (User-Agent parsing)
- Link expiration (time-based TTL)
- QR code generation (popular request, low implementation cost)
- Traffic source tracking (referrer header parsing)
- Bulk operations (batch API for creating multiple links)

**Defer (v2+, high complexity/low value):**
- Password protection (niche use case, adds friction)
- Link editing with version history (storage overhead, complex UI)
- Click fraud detection (only needed at scale)
- A/B testing support (complex routing logic)
- Webhook notifications (Dapr makes this easier, but defer until requested)

**Anti-features to avoid:**
- 301 permanent redirects (browsers cache forever, breaks analytics and destination updates)
- Unlimited link retention without expiration (accumulates spam, resource waste)
- Real-time dashboard updates via WebSocket (adds complexity for minimal value in learning project)

### Architecture Approach

The recommended architecture uses the Dapr sidecar pattern with clean architecture layering (Handler → Service → Repository). Two services run independently with Dapr sidecars handling cross-cutting concerns: URL Service owns shortening and redirection, Analytics Service consumes click events asynchronously via pub/sub.

**Major components:**
1. **URL Service** — Owns URL shortening and redirection logic. Publishes `url.clicked` events on redirect. Uses Dapr state store (SQLite dev, PostgreSQL prod) for URL mappings. Critical path is read-optimized (redirect must be <10ms).
2. **Analytics Service** — Subscribes to `url.clicked` events, aggregates click statistics, provides query API. Decoupled from URL Service (doesn't know it exists). Eventual consistency is acceptable (analytics can lag real-time clicks).
3. **Dapr Sidecars** — One per service. Handles service invocation, state management, pub/sub, observability (distributed tracing). Services communicate with sidecars via localhost HTTP/gRPC, never directly with each other.
4. **State Stores** — Each service owns its database (no shared databases). SQLite for development (single instance only), PostgreSQL for production (supports concurrent writes).
5. **Pub/Sub Broker** — Decouples services via async messaging. Redis for production, in-memory for local development. Must configure dead letter topics and idempotent handlers.

**Key patterns:**
- **Sidecar architecture**: Infrastructure concerns abstracted from business logic, portable across environments
- **Clean architecture**: Handler (HTTP) → Service (business logic) → Repository (data access), enables isolated testing
- **Event-driven communication**: Pub/sub for async workflows (analytics), service invocation for sync RPC (if needed)
- **State store abstraction**: Key-value access via Dapr API, not direct SQL (portability across backends)

**Data flow (redirect + analytics):**
```
Client GET /abc123
  → URL Service Handler
  → URL Service (get long URL from state)
  → Publish url.clicked event (fire-and-forget)
  → Return 302 redirect
  → (async) Analytics Service receives event
  → Increment click counter in analytics state store
```

### Critical Pitfalls

Research identified 8 critical pitfalls that can derail production deployments. These are patterns that work in development but fail at scale or cause production incidents.

1. **SQLite Concurrent Write Corruption** — Multiple Dapr sidecars writing to same SQLite file causes database corruption and data loss. Use SQLite only for single-instance dev/testing. Migrate to PostgreSQL before any multi-instance deployment. Warning signs: "database is locked" errors, intermittent failures.

2. **Missing Context Propagation** — Creating new contexts with `context.Background()` instead of propagating request context breaks timeout cascades, causes resource leaks. Always pass `r.Context()` through entire call chain. Verify with integration tests that timeouts propagate to downstream services.

3. **Pub/Sub Retry Explosion** — Dapr resiliency policies augment (not replace) message broker's built-in retries, creating multiplicative retry storms. Configure dead letter topics, understand broker retry behavior, make handlers idempotent for at-least-once delivery.

4. **Sidecar Resource Limits Not Set** — Dapr sidecars have no default resource limits, leading to OOM kills in production. Set limits from day one: CPU limit 300m/request 100m, Memory limit 1000Mi/request 250Mi. Configure `GOMEMLIMIT` to 90% of memory limit.

5. **Testing Against Mocks Instead of Dapr Contract** — Mocking Dapr client interfaces without understanding actual Dapr behavior creates false confidence. Write integration tests against real Dapr sidecars running in CI. Verify serialization, retries, failure modes with real components.

6. **Ignoring Read/Write Imbalance** — URL shorteners have 100:1 read/write ratio. Optimizing for write path (URL creation) instead of read path (redirect) creates performance bottleneck. Cache aggressively, measure redirect latency separately, design schema for fast single-key lookups.

7. **Service Invocation Without Resiliency Policies** — Assuming Dapr handles resiliency automatically. Must configure explicit policies for timeouts, retries, circuit breakers. Only retry idempotent operations. Use exponential backoff to prevent thundering herd.

8. **Sidecar Health Check Misconfiguration** — Default Kubernetes liveness probes timeout before Dapr sidecar initializes, causing CrashLoopBackoff. Increase initial delay to 30-60s, use startup probes (K8s 1.16+), probe Dapr's `/v1.0/healthz` endpoint.

## Implications for Roadmap

Based on research, the project should follow a dependency-driven phase structure that establishes core patterns early and defers complexity until needed. The architecture research shows clear build order: foundation (patterns and structure) → URL Service core (value delivery) → event-driven analytics (async patterns) → production readiness (scale and reliability).

### Phase 1: Foundation & URL Service Core

**Rationale:** Establishes critical patterns (clean architecture, context propagation, Dapr integration) from the start. Delivers core value (shorten + redirect) quickly. URL Service can function independently without analytics, allowing early testing.

**Delivers:**
- Project structure (cmd/, internal/, pkg/ layout per architecture research)
- URL Service with shortening and redirection endpoints
- Dapr state store integration (SQLite for dev)
- Basic input validation and error handling
- Rate limiting to prevent abuse

**Addresses features:**
- Basic URL shortening (table stakes)
- Short link redirection with 302 redirects (table stakes)
- Custom aliases (table stakes)
- API access (table stakes)
- Error handling (table stakes)
- Rate limiting (table stakes)

**Avoids pitfalls:**
- SQLite limitations documented clearly (development only)
- Context propagation pattern established from first handler
- Read path optimization (redirect <10ms target)
- Clean architecture prevents business logic in handlers

**Research flag:** Standard patterns. No additional research needed.

### Phase 2: Event-Driven Analytics

**Rationale:** Introduces async communication patterns once URL Service is stable. Analytics Service can be developed/tested independently. Pub/sub decouples services completely (URL Service doesn't block on analytics).

**Delivers:**
- Analytics Service with pub/sub subscription to `url.clicked` events
- Click tracking and aggregation in separate state store
- Analytics query API (GET /analytics/{code})
- Dead letter topic configuration
- Idempotent event handlers (at-least-once delivery safety)

**Uses stack elements:**
- Dapr pub/sub (Redis for production, in-memory for dev)
- Dapr state store (separate database from URL Service per architecture)
- Dapr SDK subscription handlers

**Implements architecture components:**
- Analytics Service (second microservice)
- Pub/Sub Broker integration
- Event-driven communication pattern

**Addresses features:**
- Click tracking (table stakes)
- Link analytics dashboard (table stakes)

**Avoids pitfalls:**
- Pub/sub retry explosion (configure DLT, understand broker retries)
- Testing against mocks (integration tests with real Dapr pub/sub)
- Synchronous analytics blocking redirects (fire-and-forget events)
- Shared database anti-pattern (each service owns its state)

**Research flag:** Standard patterns. Dapr pub/sub is well-documented.

### Phase 3: Enhanced Analytics

**Rationale:** Adds competitive differentiators once core analytics pipeline is proven. These features augment existing analytics without new architectural patterns. Low risk, high user value.

**Delivers:**
- Geo-location tracking (IP → country mapping)
- Device/browser detection (User-Agent parsing)
- Traffic source tracking (Referer header parsing)
- Analytics enrichment in event handler

**Addresses features:**
- Geo-location tracking (differentiator)
- Device/browser detection (differentiator)
- Traffic source tracking (differentiator)

**Avoids pitfalls:**
- Keeping redirect path fast (enrichment happens in async analytics service)
- Privacy concerns (hash/sanitize sensitive data in logs)

**Research flag:** Needs research. IP geolocation libraries and User-Agent parsing strategies need evaluation during planning.

### Phase 4: Link Management Features

**Rationale:** Extends URL Service with management capabilities after core workflows are stable. Features build on existing infrastructure without new services.

**Delivers:**
- Link listing/search endpoints
- Link expiration (time-based TTL)
- Bulk URL creation API
- QR code generation
- UTM parameter injection

**Addresses features:**
- Link management (table stakes)
- Link expiration (differentiator)
- Bulk operations (differentiator)
- QR code generation (differentiator)
- UTM parameter injection (differentiator)

**Avoids pitfalls:**
- Background job for expiration cleanup (proper async pattern)
- Transaction handling for bulk operations

**Research flag:** Needs research. Dapr's background job capabilities and QR code libraries need evaluation.

### Phase 5: Production Readiness

**Rationale:** Production concerns should be addressed before any deployment, but can be developed incrementally alongside feature work. Critical for reliability at scale.

**Delivers:**
- Comprehensive testing (unit, integration with Dapr, end-to-end)
- Docker Compose for local development (both services + Dapr sidecars)
- Kubernetes manifests with proper resource limits
- Health probes configured for Dapr initialization
- Resiliency policies (timeouts, retries, circuit breakers)
- Observability (structured logging, distributed tracing, metrics)
- CI/CD pipeline (lint, test, build, container)
- PostgreSQL migration from SQLite

**Avoids pitfalls:**
- Sidecar resource limits not set (configured in K8s manifests)
- Health check misconfiguration (adjusted initial delays)
- Service invocation without resiliency (explicit policies)
- SQLite corruption (migrated to PostgreSQL)
- Testing against mocks only (CI runs integration tests)

**Research flag:** Standard patterns for Docker, Kubernetes, CI/CD.

### Phase Ordering Rationale

- **Phase 1 before Phase 2**: URL Service must exist to publish events that Analytics Service consumes. Shortening/redirection provides value independently.
- **Phase 2 before Phase 3**: Analytics pipeline must work before enriching events. Core pub/sub patterns must be proven before adding complexity.
- **Phase 3 before Phase 4**: Analytics enhancements are simpler than link management features. Proves event enrichment pattern.
- **Phase 4 and Phase 5 can overlap**: Production readiness concerns (testing, deployment) should start early and continue throughout. Link management features can be developed while production infrastructure is prepared.

**Critical path dependencies:**
- Phase 1 (URL Service) blocks Phase 2 (Analytics needs events)
- Phase 2 (Analytics Service) blocks Phase 3 (Enhanced Analytics)
- Phase 1+2 must complete before Phase 5 (can't deploy without core services)

**Pitfall prevention sequence:**
- Phase 1: Establishes context propagation, clean architecture, read optimization
- Phase 2: Addresses pub/sub retry explosion, shared database anti-pattern
- Phase 3-4: Maintains patterns established in Phase 1-2
- Phase 5: Addresses all production-critical pitfalls (resource limits, health probes, resiliency, database migration)

### Research Flags

**Phases needing deeper research during planning:**
- **Phase 3 (Enhanced Analytics)**: IP geolocation libraries (accuracy, pricing, rate limits), User-Agent parsing strategies (library selection, accuracy), privacy/GDPR considerations for location tracking
- **Phase 4 (Link Management)**: QR code generation libraries (format support, customization), Dapr background jobs or external cron for expiration cleanup, bulk operation transaction strategies

**Phases with standard patterns (skip research-phase):**
- **Phase 1 (Foundation)**: Well-documented Go project structure, Chi router usage, Dapr state store basics all covered in STACK.md and ARCHITECTURE.md
- **Phase 2 (Analytics)**: Dapr pub/sub patterns extensively documented in official docs, reference implementations available
- **Phase 5 (Production)**: Docker, Kubernetes, GitHub Actions all have established patterns for Go services

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Verified with official Go 1.26 release notes, Dapr SDK v1.13.0 GitHub releases, active library maintenance confirmed. All major choices have official documentation and production usage. |
| Features | MEDIUM | Verified with multiple URL shortener comparisons (Bitly, Rebrandly, Short.io market analysis), system design resources, but no official "URL shortener standard" exists. Feature priorities inferred from competitive analysis. |
| Architecture | HIGH | Official Dapr documentation for sidecar pattern, state management, pub/sub. Reference implementations (Dapr Store) validate project structure. Clean architecture patterns well-established in Go community. |
| Pitfalls | MEDIUM-HIGH | Critical pitfalls sourced from official Dapr troubleshooting docs (SQLite, health probes, retries). Context propagation and testing issues from engineering blogs (Grab, Rotational Labs). Some pitfalls inferred from general microservices experience. |

**Overall confidence:** HIGH

The stack and architecture recommendations are highly confident, sourced from official documentation and verified with version-specific release notes. Feature prioritization has medium confidence due to lack of canonical standards, but competitive analysis across multiple market leaders (2026 data) provides strong guidance. Pitfall identification benefits from official Dapr troubleshooting documentation and production experience reports from engineering teams.

### Gaps to Address

**During planning/execution:**

1. **IP Geolocation Strategy** (Phase 3): Research didn't identify specific library recommendation. Need to evaluate during Phase 3 planning: free vs paid services, accuracy requirements, rate limits, privacy compliance. Options: MaxMind GeoLite2, ipapi.co, IP2Location.

2. **Background Job Implementation** (Phase 4): Dapr doesn't have native scheduled job support. Need to decide between: external cron triggering endpoints, sidecar actor reminders, or third-party job schedulers. Research during Phase 4 planning.

3. **PostgreSQL Migration Strategy** (Phase 5): Research focused on SQLite limitations but didn't detail migration process. Need to plan: data migration scripts, zero-downtime deployment strategy, state store configuration changes. Address during Phase 5 planning.

4. **Unique Click Tracking** (Future): Research noted Rebrandly tracks unique clicks (deduplication) vs total clicks. If needed, requires session tracking or fingerprinting strategy not covered in current research. Defer until requested by users.

5. **Custom Domain Support** (Future): Rebrandly's key differentiator is custom domain support. If added later, requires DNS configuration and multi-tenancy architecture not in current scope. Note as potential v2+ feature.

## Sources

### Primary Sources (HIGH confidence)

**Official Documentation:**
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26) — Latest Go version verification
- [Dapr Go SDK v1.13.0](https://github.com/dapr/go-sdk/releases) — SDK version and compatibility
- [Dapr Documentation](https://docs.dapr.io/) — Sidecar architecture, state management, pub/sub, service invocation patterns
- [Dapr SQLite Component](https://docs.dapr.io/reference/components-reference/supported-state-stores/setup-sqlite/) — State store configuration and limitations
- [Dapr Kubernetes Production Guidelines](https://docs.dapr.io/operations/hosting/kubernetes/kubernetes-production/) — Resource limits, health probes
- [Dapr Resiliency Policies](https://docs.dapr.io/operations/resiliency/policies/) — Retry, timeout, circuit breaker configuration
- [Go slog Package](https://pkg.go.dev/log/slog) — Stdlib structured logging

**Framework Documentation:**
- [Chi Router](https://go-chi.io/) — HTTP routing and middleware
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure Go SQLite driver
- [validator/v10](https://pkg.go.dev/github.com/go-playground/validator/v10) — Struct validation
- [golangci-lint](https://golangci-lint.run/docs/product/changelog/) — v2.9.0 release verification

### Secondary Sources (MEDIUM-HIGH confidence)

**Engineering Blogs:**
- [Grab Engineering: Context Deadlines](https://engineering.grab.com/context-deadlines-and-how-to-set-them) — Context propagation patterns
- [Rotational Labs: Contexts in Go Microservice Chains](https://rotational.io/blog/contexts-in-go-microservice-chains/) — Microservice context best practices
- [Diagrid: Fault Tolerant Microservices with Dapr](https://www.diagrid.io/blog/fault-tolerant-microservices-made-easy-with-dapr-resiliency) — Resiliency patterns
- [Diagrid: Dapr Testing](https://www.diagrid.io/blog/mastering-the-dapr-inner-development-loop-part-1-code-unit-test) — Integration testing strategies

**System Design Resources:**
- [System Design School: URL Shortener](https://systemdesignschool.io/problems/url-shortener/solution) — Architecture patterns
- [Hello Interview: Design Bit.ly](https://www.hellointerview.com/learn/system-design/problem-breakdowns/bitly) — Read/write imbalance analysis
- [AlgoMaster: Design URL Shortener](https://algomaster.io/learn/system-design-interviews/design-url-shortener) — System design interview patterns

**Best Practices:**
- [Three Dots Labs: Clean Architecture in Go](https://threedots.tech/post/introducing-clean-architecture/) — Handler/Service/Repository pattern
- [Go Standard Project Layout](https://github.com/golang-standards/project-layout) — Project structure conventions
- [Go Testing Excellence 2026](https://dasroot.net/posts/2026/01/go-testing-excellence-table-driven-tests-mocking/) — Testing strategies

### Tertiary Sources (MEDIUM confidence)

**Competitive Analysis:**
- [10 Best URL Shorteners of 2026](https://minily.org/blog/best-url-shorteners-2026) — Feature comparison
- [Bitly vs Rebrandly Comparison 2026](https://theadcompare.com/comparisons/bitly-vs-rebrandly/) — Market leader analysis
- [Dub Links vs Bitly Comparison](https://efficient.app/compare/dub-links-vs-bitly) — Developer-focused shortener features
- [Best URL Shortener APIs 2026](https://www.rebrandly.com/blog/url-shortener-apis) — API-first shortener patterns

**Community Resources:**
- [URL Shortener Common Mistakes](https://urlshortly.com/blog/common-url-shortener-mistakes-and-how-to-avoid-them) — Anti-patterns
- [10 Short URL Best Practices](https://bitly.com/blog/short-url-best-practices/) — Industry best practices
- [301 vs 302 Redirects for URL Shorteners](https://url-shortening.com/blog/301-vs-302-redirects-in-shorteners-speed-seo-and-caching) — Redirect strategy

---
*Research completed: 2026-02-14*
*Ready for roadmap: yes*
