# Phase 10: Resilience & Infrastructure - Context

**Gathered:** 2026-02-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Enable go-zero production features (Prometheus metrics, circuit breakers) and Docker Compose orchestration for all services. Achieve 80%+ test coverage with unit and integration tests. This phase hardens the existing microservices stack — no new business features.

</domain>

<decisions>
## Implementation Decisions

### Circuit breaker behavior
- Use go-zero's built-in breaker package — no external libraries
- When circuit opens, fail fast with error — no silent degradation or cached fallbacks
- Note: existing graceful degradation for analytics-rpc (returns 0 clicks when down) is a *design choice*, not a circuit breaker fallback — that stays as-is

### Claude's Discretion: Circuit breaker scope
- Claude determines which downstream calls get circuit breaker protection (PostgreSQL, Kafka, zRPC) based on architecture analysis
- Claude determines whether rate limiting needs attention in this phase or if existing in-memory rate limiter is sufficient

### Test coverage strategy
- Unit tests for logic layer + integration tests for handlers
- No E2E tests — unit and integration provide the coverage target
- Use testcontainers for integration tests — spin up real PostgreSQL and Kafka in Docker
- Standard library testing package + testify for assertions — no BDD frameworks
- 80% coverage target enforced in CI — builds fail if coverage drops below threshold

### Metrics & observability scope
- Claude's discretion — success criteria requires Prometheus metrics on /metrics with request duration and status code data
- Implementation approach for metrics is open

### Docker Compose orchestration
- Claude's discretion — success criteria requires all services startable via Docker Compose
- Health checks, startup ordering, and networking approach are open

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. Follow go-zero conventions and patterns established in Phases 7-9.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 10-resilience-infrastructure*
*Context gathered: 2026-02-16*
