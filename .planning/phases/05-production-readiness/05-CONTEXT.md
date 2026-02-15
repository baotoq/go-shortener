# Phase 5: Production Readiness - Context

**Gathered:** 2026-02-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Make both services tested, containerized, and deployable to production. This includes unit and integration tests across all layers, Docker multi-stage builds, Docker Compose orchestration with Dapr sidecars, GitHub Actions CI/CD pipeline, health check endpoints, and switching cross-service calls to Dapr service invocation. No new user-facing features.

</domain>

<decisions>
## Implementation Decisions

### Testing strategy
- Unit tests for all layers: handler, usecase, and repository
- Mocks at boundaries using mockery/gomock (generated mocks from interfaces)
- Scenario-based test style: each test function is a meaningful scenario with clear setup/act/assert
- Dapr mocking: interface mocks for unit tests, real Dapr sidecar via testcontainers for integration tests
- Target >80% coverage per service

### Docker & local dev
- Multi-stage Dockerfiles: build in Go image, copy binary to distroless/scratch for minimal production images
- Docker Compose with Dapr sidecars as separate containers (each service paired with its own daprd container)
- GeoIP database mounted as a volume (not baked into image) for easy updates without rebuild
- Health check endpoints: `/healthz` (liveness) and `/readyz` (readiness — checks DB and Dapr connectivity)

### CI/CD pipeline
- GitHub Actions as CI provider
- Pipeline stages: lint (golangci-lint) + test (unit + integration) + build (binary) + Docker (image build)
- Trigger: PRs and pushes to main/master only (not every branch push)
- Integration tests use testcontainers to spin up Dapr sidecars in CI — no Docker Compose dependency in pipeline
- Testcontainers keep integration tests self-contained and runnable anywhere

### Service invocation
- Switch click count enrichment call (URL Service -> Analytics Service) from direct HTTP to Dapr service invocation
- Deletion cascade stays as pub/sub (async event-driven — no need for synchronous confirmation)
- Dapr service invocation uses HTTP protocol (not gRPC — simpler, services already speak HTTP)
- Service discovery via Dapr app-id instead of hostname/port environment variables

### Claude's Discretion
- Specific golangci-lint configuration and enabled linters
- Exact testcontainer setup and teardown patterns
- Dockerfile base image versions (Go, distroless)
- Docker Compose network configuration
- GitHub Actions caching strategy for Go modules and Docker layers
- Health check implementation details (response format, check intervals)

</decisions>

<specifics>
## Specific Ideas

- Testcontainers for Dapr integration tests in CI (user explicitly wanted easy-to-run integration tests)
- Dapr app-id based discovery replaces environment-variable hostname approach
- Readiness probe should verify both database connectivity and Dapr sidecar availability

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 05-production-readiness*
*Context gathered: 2026-02-15*
