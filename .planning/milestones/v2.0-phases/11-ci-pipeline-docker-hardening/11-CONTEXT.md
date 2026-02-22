# Phase 11: CI Pipeline & Docker Hardening - Context

**Gathered:** 2026-02-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Fix CI pipeline for v2.0 go-zero structure, resolve the known Docker Kafka publishing issue, configure PostgreSQL connection pooling, and add health check endpoints. This is a gap closure phase — no new features, only hardening what was built in phases 7-10.

</domain>

<decisions>
## Implementation Decisions

### CI Pipeline Scope
- GitHub Actions CI workflow (`.github/workflows/ci.yml`) for v2.0 structure
- Run on every push to any branch AND on pull requests to master
- Single Go version (latest stable matching go.mod), no matrix builds
- Linting via golangci-lint
- Unit tests with mocks (existing test suite from Phase 10)
- Integration tests using testcontainers-go to spin up real PostgreSQL and Kafka in CI
- Build all three services to verify compilation

### Kafka Docker Fix
- Root cause is likely Kafka listener/advertised-listener mismatch in Docker Compose networking
- Fix the networking config (advertised listeners, Docker DNS resolution for kq.Pusher)
- Keep the fire-and-forget publishing pattern — do not change to synchronous
- Keep dual Kafka listener setup (port 9092 internal Docker, 29092 for host)
- No Kafka connectivity check at startup — keep lazy connection behavior
- Log errors in background goroutine at most, but never block the user

### Connection Pool Tuning
- Config-driven via YAML: add MaxOpenConns, MaxIdleConns, ConnMaxLifetime fields to each service config
- Conservative defaults: MaxOpen=10, MaxIdle=5, MaxLifetime=1h
- Each service has its own independent pool config fields (same defaults, tunable per-service)
- Log active pool configuration at startup for visibility/debugging

### Consumer Health Check
- Lightweight HTTP server on dedicated port (e.g., 8082) for analytics-consumer healthz
- Add healthz endpoints to all three services for consistency
- Health = liveness check (process is running, endpoint responds). Kafka reconnects automatically.
- Standard HTTP status codes: 200 OK when healthy, 503 Service Unavailable when not
- url-api and analytics-rpc: healthz route in existing HTTP/gRPC server
- analytics-consumer: separate small HTTP server since it's a Kafka consumer with no existing HTTP

### Claude's Discretion
- golangci-lint configuration (which linters to enable/disable)
- Exact testcontainers-go setup and test structure
- Health check port numbers for each service
- Exact Kafka advertised listener fix (depends on investigation)
- Pool config struct naming and YAML key names

</decisions>

<specifics>
## Specific Ideas

- Testcontainers-go for integration tests — spin up real PostgreSQL and Kafka containers during test execution, no need for Docker Compose services in CI
- Health endpoints should follow Kubernetes liveness probe conventions (simple GET, fast response)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 11-ci-pipeline-docker-hardening*
*Context gathered: 2026-02-16*
