# Phase 10: Resilience & Infrastructure - Research

**Researched:** 2026-02-16
**Domain:** go-zero production features (Prometheus, circuit breakers, load shedding), Docker Compose orchestration, testcontainers-go integration testing
**Confidence:** HIGH

## Summary

Phase 10 hardens the existing microservices stack for production. go-zero provides most resilience features out-of-the-box: Prometheus metrics are automatically exposed on port 6470 via DevServer, circuit breakers use Google SRE's adaptive algorithm on both zRPC and HTTP clients, adaptive load shedding is built into REST servers, and MaxConns limits concurrent connections. Docker Compose orchestrates all services with multi-stage Dockerfile builds, health checks, and startup ordering via `depends_on`. Test coverage uses testcontainers-go for real PostgreSQL/Kafka in integration tests alongside standard unit tests with mocked ServiceContext dependencies.

**Key findings:**
- go-zero auto-enables Prometheus metrics on DevServer port 6470 `/metrics` with `http_server_requests_duration_ms` histogram and `http_server_requests_code_total` counter
- Circuit breakers are enabled by default on zRPC client interceptors (adaptive Google SRE algorithm, K=1.5)
- REST server has built-in adaptive load shedding and MaxConns (default 10000) - no extra code needed
- OpenTelemetry tracing middleware is enabled by default (Trace=true) - just needs a collector endpoint for production
- Docker multi-stage builds keep images small (~20MB for Go binaries)
- testcontainers-go v0.34+ provides PostgreSQL and Kafka modules for integration tests

**Primary recommendation:** Enable DevServer in YAML configs for Prometheus metrics, rely on go-zero's built-in circuit breakers and load shedding (already active), create Dockerfiles and full-stack Docker Compose, and write unit + integration tests targeting 80% coverage.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/zeromicro/go-zero` | v1.10.0 | Framework (already in go.mod) | Built-in Prometheus, circuit breaker, load shedding, tracing |
| `github.com/testify` | v1.9+ | Test assertions | Most popular Go test assertion library |
| `github.com/testcontainers/testcontainers-go` | v0.34+ | Integration test containers | Official testcontainers for Go, PostgreSQL and Kafka modules |
| `go.uber.org/mock` | v0.6+ | Interface mocking (already indirect dep) | Standard mockgen for ServiceContext mocking |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `testcontainers-go/modules/postgres` | v0.34+ | PostgreSQL container for tests | Integration tests needing real database |
| `testcontainers-go/modules/kafka` | v0.34+ | Kafka container for tests | Integration tests needing real message queue |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| testcontainers-go | dockertest | testcontainers is more actively maintained, has module ecosystem |
| go.uber.org/mock | mockery | mockery generates from interfaces too, but uber/mock is already in go.mod |
| Multi-stage Dockerfile | Distroless images | Distroless is slightly smaller but harder to debug |

## Architecture Patterns

### Pattern 1: Prometheus Metrics (Auto-Enabled)

**What:** go-zero auto-exposes Prometheus metrics on DevServer port
**When to use:** Always enabled for production monitoring
**Configuration:**
```yaml
# services/url-api/etc/url.yaml
DevServer:
  Enabled: true
  Port: 6470
  MetricsPath: /metrics
```

**Built-in metrics (automatic, no code needed):**
- `http_server_requests_duration_ms` (Histogram) - request latency by path
- `http_server_requests_code_total` (Counter) - status codes by path and code
- Standard Go runtime metrics (goroutines, GC, memory)

**For zRPC services:**
```yaml
# services/analytics-rpc/etc/analytics.yaml
DevServer:
  Enabled: true
  Port: 6471
  MetricsPath: /metrics
```

**Key characteristics:**
- DevServer runs on a separate port from the main service
- Each service needs a unique DevServer port
- Prometheus middleware is enabled by default (Middlewares.Prometheus: true)
- No code changes needed - purely configuration

### Pattern 2: Circuit Breaker (Built-In)

**What:** Adaptive circuit breaker on zRPC client calls using Google SRE algorithm
**When to use:** Already active by default on all zRPC clients
**How it works:**
```
dropRatio = max(0, (total - protection - K * accepts) / (total + 1))
```
- `K=1.5` (default) - ratio of accepted requests to total before tripping
- `protection` = minimum requests before circuit breaker can activate
- Breaker name = target + method (e.g., `dns:///localhost:8081/analytics.Analytics/GetClickCount`)

**No code changes needed.** The circuit breaker is enabled by default via:
```go
// Already in go-zero's zrpc.MustNewClient internals
// ServerMiddlewaresConf defaults:
//   Breaker: true (enabled by default)
```

**For downstream calls (PostgreSQL, Kafka):**
- go-zero's `sqlx` has built-in breaker on database calls (connection pool handles failures)
- Kafka's `kq.Pusher` has internal retry with backoff
- No additional circuit breaker wrapper needed for these

### Pattern 3: Docker Multi-Stage Build

**What:** Small production Docker images with compiled Go binaries
**When to use:** All three services need Dockerfiles
**Example:**
```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/service ./services/url-api/url.go

# Run stage
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/service /bin/service
COPY services/url-api/etc /etc/service
CMD ["/bin/service", "-f", "/etc/service/url.yaml"]
```

**Key characteristics:**
- Alpine base for small images (~20MB)
- CGO_ENABLED=0 for static binaries
- Separate etc/ directory for configuration
- ca-certificates for HTTPS outbound calls

### Pattern 4: Docker Compose Full Stack

**What:** Single command to start all infrastructure and services
**When to use:** Local development and integration testing
**Key additions to existing docker-compose.yml:**
- Three application services (url-api, analytics-rpc, analytics-consumer)
- Health checks and depends_on with condition: service_healthy
- Shared network for inter-service communication
- Environment variable overrides for Docker networking (localhost -> container names)

### Pattern 5: Unit Tests with Mocked Dependencies

**What:** Test logic layer by mocking model interfaces and RPC clients
**When to use:** All logic files (shorten, redirect, links, getclickcount)
**Example:**
```go
func TestShortenLogic_Shorten(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockUrlModel := mock.NewMockUrlsModel(ctrl)
    mockUrlModel.EXPECT().Insert(gomock.Any(), gomock.Any()).Return(nil, nil)

    svcCtx := &svc.ServiceContext{
        Config:   config.Config{BaseUrl: "http://localhost:8080"},
        UrlModel: mockUrlModel,
    }

    logic := shorten.NewShortenLogic(context.Background(), svcCtx)
    resp, err := logic.Shorten(&types.ShortenRequest{OriginalUrl: "https://example.com"})
    assert.NoError(t, err)
    assert.NotEmpty(t, resp.ShortCode)
}
```

### Pattern 6: Integration Tests with testcontainers-go

**What:** Spin up real PostgreSQL in Docker for integration tests
**When to use:** Testing full handler-to-database flow
**Example:**
```go
func TestShortenHandler_Integration(t *testing.T) {
    ctx := context.Background()
    pgContainer, err := postgres.Run(ctx, "postgres:16-alpine",
        postgres.WithDatabase("shortener"),
        postgres.WithUsername("postgres"),
        postgres.WithPassword("postgres"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).WithStartupTimeout(5*time.Second)),
    )
    require.NoError(t, err)
    defer pgContainer.Terminate(ctx)

    connStr, _ := pgContainer.ConnectionString(ctx, "sslmode=disable")
    // Run migrations, create ServiceContext, test handler
}
```

### Anti-Patterns to Avoid

- **Custom Prometheus middleware**: go-zero already provides it. Don't add your own histogram/counter.
- **Manual circuit breaker wrappers**: go-zero's zRPC and sqlx have built-in breakers. Don't wrap calls in custom breaker logic.
- **Fat Docker images**: Don't copy the entire source tree into the final image. Use multi-stage builds.
- **Hardcoded connection strings in Docker**: Use environment variables or Docker Compose service names.
- **Testing with mocks only**: Integration tests with real databases catch SQL errors that unit tests miss.

## Common Pitfalls

### Pitfall 1: DevServer Port Conflicts

**What goes wrong:** Multiple services try to use the same DevServer port
**Why it happens:** Default DevServer port is 6470 for all services
**How to avoid:** Assign unique DevServer ports per service (6470 for url-api, 6471 for analytics-rpc, 6472 for analytics-consumer)

### Pitfall 2: Docker Compose Service Ordering

**What goes wrong:** Application services start before PostgreSQL/Kafka are ready
**Why it happens:** `depends_on` without `condition: service_healthy` only waits for container start, not readiness
**How to avoid:** Use health checks on infrastructure services and `depends_on` with `condition: service_healthy`

### Pitfall 3: Docker Networking vs localhost

**What goes wrong:** Services inside Docker can't connect to `localhost:5433`
**Why it happens:** Inside Docker, localhost refers to the container itself, not the host
**How to avoid:** Use Docker Compose service names as hostnames (e.g., `postgres:5432` not `localhost:5433`). Create Docker-specific YAML configs or use environment variable overrides.

### Pitfall 4: Test Isolation with Shared Database

**What goes wrong:** Tests interfere with each other through shared state
**Why it happens:** Multiple tests writing to the same database tables
**How to avoid:** Each integration test gets its own testcontainers PostgreSQL instance. Use `t.Cleanup` for teardown.

### Pitfall 5: testcontainers-go Docker Socket Permission

**What goes wrong:** testcontainers can't connect to Docker daemon
**Why it happens:** CI environment may not have Docker socket mounted
**How to avoid:** Ensure CI pipeline has Docker-in-Docker or Docker socket mounted. Use `testcontainers.WithReuse` for faster local development.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Prometheus metrics | Custom histogram/counter setup | go-zero DevServer auto-metrics | Already collects request duration and status codes |
| Circuit breaker | Custom failure counting logic | go-zero built-in breaker | Adaptive Google SRE algorithm, battle-tested |
| Load shedding | Manual request rejection | go-zero adaptive shedding | CPU-aware, automatic, built into REST server |
| MaxConns rate limiting | Custom semaphore/counter | go-zero RestConf.MaxConns | Built-in, configurable via YAML |
| Test containers | Manual Docker commands in tests | testcontainers-go modules | Clean API, automatic cleanup, module ecosystem |
| Mock generation | Hand-written mock structs | go.uber.org/mock (mockgen) | Generated from interfaces, type-safe |

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual Prometheus setup | go-zero DevServer auto-metrics | go-zero v1.4+ | Zero-code metrics collection |
| Static circuit breaker thresholds | Adaptive Google SRE algorithm | go-zero v1.0+ | Self-tuning based on actual traffic |
| Docker Compose v2 file format | Docker Compose v2 with healthcheck conditions | Compose Spec 2021+ | Reliable service startup ordering |
| dockertest library | testcontainers-go | 2023+ | Better module ecosystem, more actively maintained |

## Sources

### Primary (HIGH confidence)

- [go-zero Prometheus monitoring docs](https://go-zero.dev/docs/tutorials/monitor/index) - DevServer auto-metrics, port 6470
- [go-zero HTTP middleware docs](https://go-zero.dev/en/docs/tutorials/http/server/middleware) - PrometheusHandler, built-in middleware
- [go-zero gRPC server configuration](https://go-zero.dev/docs/tutorials/grpc/server/configuration) - ServerMiddlewaresConf (Prometheus, Breaker defaults)
- [go-zero service governance](https://go-zero.dev/docs/tutorials/service/governance/limiter) - MaxConns, load shedding
- [go-zero breaker implementation](https://github.com/zeromicro/go-zero/tree/master/core/breaker) - Google SRE adaptive algorithm
- [testcontainers-go documentation](https://golang.testcontainers.org/) - PostgreSQL and Kafka modules

### Secondary (MEDIUM confidence)

- [Docker multi-stage builds](https://docs.docker.com/build/building/multi-stage/) - Official Docker documentation
- [Docker Compose healthcheck](https://docs.docker.com/reference/compose-file/services/#healthcheck) - Service dependency conditions
- [go.uber.org/mock](https://github.com/uber-go/mock) - MockGen interface mocking

## Metadata

**Confidence breakdown:**
- Prometheus/metrics: HIGH - Verified via go-zero official docs (Context7) and source code
- Circuit breaker: HIGH - Documented in go-zero zRPC docs, Google SRE algorithm well-known
- Docker: HIGH - Standard Docker patterns, well-documented
- Testing: HIGH - testcontainers-go is mature, well-documented

**Research date:** 2026-02-16
**Valid until:** 2026-03-16 (30 days - stable ecosystem)
