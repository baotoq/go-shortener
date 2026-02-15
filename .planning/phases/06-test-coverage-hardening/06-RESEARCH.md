# Phase 06: Test Coverage Hardening - Research

**Researched:** 2026-02-15
**Domain:** Go testing, HTTP handler test coverage, Dapr integration testing
**Confidence:** HIGH

## Summary

Phase 6 closes test coverage gaps identified in the v1.0 milestone audit. The URL Service handler layer currently has 44.6% coverage, well below the 80% target. Analysis reveals uncovered code in three areas: (1) middleware (rate limiter, logger) with 0% coverage, (2) router setup with 0% coverage, (3) readiness probe with 0% coverage, and (4) fire-and-forget click event publishing with 0% coverage. Additionally, no integration tests exist that exercise real Dapr sidecars, leaving pub/sub and service invocation untested against actual Dapr runtime.

The standard approach for handler coverage is table-driven tests using httptest.ResponseRecorder and chi router integration. For Dapr integration tests, testcontainers-go provides programmatic Docker container management, allowing tests to spin up real Dapr sidecars, verify pub/sub message delivery, and test service invocation paths without mocking.

**Primary recommendation:** Use table-driven handler tests for middleware/router/readiness probe coverage, and testcontainers-go with Docker Compose for real Dapr integration tests running in CI.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| testify | 1.10.0+ | Assertions and mocking | De facto Go testing library, already in use |
| httptest | stdlib | HTTP response recording | Built-in Go package, zero overhead |
| testcontainers-go | 0.28.0+ | Docker container orchestration for tests | Industry standard for real integration tests |
| go-test-coverage | v2 (vladopajic) | Coverage threshold enforcement | Already in use (.coverage.yaml), supports file/package/total thresholds |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| chi | 5.0.0+ | Router for handler tests | URL param extraction in tests (already in use) |
| golang.org/x/time/rate | stdlib | Rate limiter testing | Verify rate limiting behavior |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| testcontainers-go | Manual Docker Compose scripts | Containers already in project; testcontainers provides cleanup, wait strategies, better isolation |
| httptest | Real HTTP servers | httptest is faster, deterministic, no port conflicts |

**Installation:**
```bash
go get github.com/testcontainers/testcontainers-go@latest
```

## Architecture Patterns

### Recommended Test Structure
```
internal/{service}/delivery/http/
├── handler.go
├── handler_test.go          # Existing handler tests
├── middleware.go
├── middleware_test.go       # NEW: Middleware unit tests
├── router.go
└── router_test.go           # NEW: Router integration tests

test/integration/
├── dapr_pubsub_test.go      # NEW: Real Dapr pub/sub test
└── dapr_invocation_test.go  # NEW: Real Dapr service invocation test
```

### Pattern 1: Table-Driven Middleware Tests
**What:** Test middleware in isolation using httptest and mock http.Handler
**When to use:** Rate limiter, logger, any middleware with side effects
**Example:**
```go
// Source: Go testing best practices 2026
func TestRateLimiter_Middleware_ExceedsLimit_Returns429(t *testing.T) {
  rl := NewRateLimiter(2) // 2 requests per minute

  handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
  })

  // First request succeeds
  req1 := httptest.NewRequest("GET", "/test", nil)
  req1.RemoteAddr = "192.168.1.1:12345"
  rr1 := httptest.NewRecorder()
  rl.Middleware(handler).ServeHTTP(rr1, req1)
  assert.Equal(t, http.StatusOK, rr1.Code)

  // Second request succeeds
  req2 := httptest.NewRequest("GET", "/test", nil)
  req2.RemoteAddr = "192.168.1.1:12345"
  rr2 := httptest.NewRecorder()
  rl.Middleware(handler).ServeHTTP(rr2, req2)
  assert.Equal(t, http.StatusOK, rr2.Code)

  // Third request fails (rate limit exceeded)
  req3 := httptest.NewRequest("GET", "/test", nil)
  req3.RemoteAddr = "192.168.1.1:12345"
  rr3 := httptest.NewRecorder()
  rl.Middleware(handler).ServeHTTP(rr3, req3)
  assert.Equal(t, http.StatusTooManyRequests, rr3.Code)
  assert.Contains(t, rr3.Header().Get("X-RateLimit-Limit"), "2")
}
```

### Pattern 2: Router Integration Tests
**What:** Test full routing setup including middleware order and route registration
**When to use:** Verify NewRouter creates correct route structure
**Example:**
```go
// Source: httptest documentation
func TestNewRouter_HealthCheckRoute_BypassesRateLimit(t *testing.T) {
  handler := setupTestHandler(t)
  logger := zap.NewNop()
  rl := NewRateLimiter(0) // 0 requests allowed

  router := NewRouter(handler, logger, rl)

  // Health check should bypass rate limiter
  req := httptest.NewRequest("GET", "/healthz", nil)
  rr := httptest.NewRecorder()
  router.ServeHTTP(rr, req)

  assert.Equal(t, http.StatusOK, rr.Code)
}
```

### Pattern 3: Readiness Probe Tests with Mock DB
**What:** Test Readyz handler using in-memory SQLite or mock sql.DB
**When to use:** Verify DB ping timeout, Dapr health check
**Example:**
```go
// Source: Testing HTTP handlers in Go
func TestReadyz_DatabaseUnavailable_Returns503(t *testing.T) {
  mockRepo := mocks.NewMockURLRepository(t)
  service := usecase.NewURLService(mockRepo, nil, zap.NewNop(), "http://localhost")

  // Create failing DB connection
  db, mock, _ := sqlmock.New()
  mock.ExpectPing().WillReturnError(errors.New("connection refused"))

  handler := NewHandler(service, "http://localhost", nil, zap.NewNop(), db)

  req := httptest.NewRequest("GET", "/readyz", nil)
  rr := httptest.NewRecorder()

  handler.Readyz(rr, req)

  assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
  var resp HealthResponse
  json.NewDecoder(rr.Body).Decode(&resp)
  assert.Equal(t, "unavailable", resp.Status)
  assert.Contains(t, resp.Reason, "database unavailable")
}
```

### Pattern 4: Testcontainers-Go for Dapr Integration Tests
**What:** Programmatically start Dapr sidecars and services in Docker containers
**When to use:** Real Dapr pub/sub and service invocation tests
**Example:**
```go
// Source: testcontainers-go documentation
func TestDaprPubSub_ClickEvent_DeliveredToAnalytics(t *testing.T) {
  ctx := context.Background()

  // Start Redis for Dapr pub/sub
  redisC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
    ContainerRequest: testcontainers.ContainerRequest{
      Image: "redis:7",
      ExposedPorts: []string{"6379/tcp"},
      WaitingFor: wait.ForLog("Ready to accept connections"),
    },
    Started: true,
  })
  require.NoError(t, err)
  defer redisC.Terminate(ctx)

  // Start URL Service + Dapr sidecar
  urlServiceC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
    ContainerRequest: testcontainers.ContainerRequest{
      FromDockerfile: testcontainers.FromDockerfile{
        Context: "../..",
        Dockerfile: "Dockerfile.url-service",
      },
      ExposedPorts: []string{"8080/tcp"},
      WaitingFor: wait.ForHTTP("/healthz").WithPort("8080/tcp"),
    },
    Started: true,
  })
  require.NoError(t, err)
  defer urlServiceC.Terminate(ctx)

  // Publish click event via URL Service
  // Verify event received by Analytics Service
  // ...
}
```

### Anti-Patterns to Avoid
- **Testing publishClickEvent with real goroutines:** Fire-and-forget goroutines are hard to test synchronously. Use Dapr client mocking for unit tests, integration tests for real behavior.
- **Over-mocking Dapr client:** For coverage, mock the DaprClient interface. For integration tests, use real Dapr sidecars.
- **Testing router with individual endpoints:** Router tests should verify middleware order and route registration, not re-test handler logic.
- **Skipping context timeout tests:** Readiness probe uses 2-second timeout; test timeout expiration paths.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Docker container lifecycle for tests | Shell scripts in Makefile | testcontainers-go | Automatic cleanup on test failure, wait strategies, port discovery, parallel test isolation |
| Coverage enforcement | Manual go tool cover + grep | go-test-coverage (already in use) | Per-file, per-package, and total thresholds with override rules |
| HTTP response recording | Custom ResponseWriter wrapper | httptest.ResponseRecorder | Battle-tested, handles edge cases, zero dependencies |
| Rate limiter testing with time | time.Sleep() between requests | Inject custom time.Ticker or test getLimiter directly | Tests become flaky and slow with real time |

**Key insight:** Dapr integration testing complexity grows exponentially with hand-rolled Docker scripts. Testcontainers provides wait strategies, port mapping, network isolation, and automatic cleanup that would take hundreds of lines to replicate.

## Common Pitfalls

### Pitfall 1: Goroutine Leaks in Middleware Tests
**What goes wrong:** NewRateLimiter starts a background cleanup goroutine in StartCleanup() that runs forever. Tests accumulate goroutines.
**Why it happens:** No cleanup mechanism; ticker never stops in tests.
**How to avoid:** Add StopCleanup() method or test getLimiter() and Middleware() separately without calling NewRateLimiter.
**Warning signs:** `go test -race` reports goroutine leaks, test memory grows over time.

### Pitfall 2: Chi Router Requires Real Router for URLParam
**What goes wrong:** chi.URLParam(r, "code") returns empty string in tests unless request goes through chi.Router.
**Why it happens:** Chi stores URL params in request context; only chi.Router populates them.
**How to avoid:** Use chi.NewRouter() and router.ServeHTTP() in tests, not handler.Method() directly.
**Warning signs:** Handler tests pass but code behaves differently; 404s in integration tests.

### Pitfall 3: Testcontainers Requires Docker Daemon
**What goes wrong:** CI fails with "Cannot connect to Docker daemon" if Docker not available.
**Why it happens:** testcontainers-go needs Docker to run containers.
**How to avoid:** GitHub Actions has Docker pre-installed; ensure Docker service running in CI. Add skip logic: `if testing.Short() { t.Skip("integration test requires Docker") }`.
**Warning signs:** Tests pass locally but fail in CI.

### Pitfall 4: Rate Limiter Shared State Across Tests
**What goes wrong:** Rate limiter tests interfere with each other; flaky failures.
**Why it happens:** RateLimiter.limiters map shared across tests using same IP.
**How to avoid:** Create new RateLimiter instance per test. Use different IPs per test case.
**Warning signs:** Tests pass individually (`go test -run TestSpecific`) but fail when run together.

### Pitfall 5: Dapr Health Check Hardcoded localhost:3500
**What goes wrong:** Readyz handler checks http://localhost:3500/v1.0/healthz, fails in tests without Dapr.
**Why it happens:** Dapr sidecar port hardcoded in handler.
**How to avoid:** Mock HTTP client in tests OR make Dapr health check URL configurable OR test with real Dapr sidecar.
**Warning signs:** Readyz tests always fail with "connection refused" unless Dapr running.

### Pitfall 6: Coverage Threshold Configuration Confusion
**What goes wrong:** go-test-coverage fails on handler package even after adding tests.
**Why it happens:** .coverage.yaml has package threshold 70% but handler package still below.
**How to avoid:** Check per-package coverage: `go test -coverprofile=coverage.out ./internal/.../http && go tool cover -func=coverage.out | grep total`. Add package-specific override if needed.
**Warning signs:** CI fails with "package coverage below threshold" but total coverage meets target.

## Code Examples

Verified patterns from official sources:

### Middleware Test with Table-Driven Pattern
```go
// Source: Go table-driven tests documentation
func TestRateLimiter_Middleware(t *testing.T) {
  tests := []struct {
    name           string
    requestsPerMin int
    requestCount   int
    expectedStatus int
  }{
    {
      name:           "WithinLimit_Returns200",
      requestsPerMin: 10,
      requestCount:   5,
      expectedStatus: http.StatusOK,
    },
    {
      name:           "ExceedsLimit_Returns429",
      requestsPerMin: 2,
      requestCount:   3,
      expectedStatus: http.StatusTooManyRequests,
    },
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      rl := NewRateLimiter(tt.requestsPerMin)
      handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
      })

      var lastStatus int
      for i := 0; i < tt.requestCount; i++ {
        req := httptest.NewRequest("GET", "/test", nil)
        req.RemoteAddr = "192.168.1.1:12345"
        rr := httptest.NewRecorder()
        rl.Middleware(handler).ServeHTTP(rr, req)
        lastStatus = rr.Code
      }

      assert.Equal(t, tt.expectedStatus, lastStatus)
    })
  }
}
```

### Testcontainers Wait Strategy
```go
// Source: testcontainers-go documentation
func setupDaprSidecar(ctx context.Context, t *testing.T, appPort int) testcontainers.Container {
  req := testcontainers.ContainerRequest{
    Image: "daprio/daprd:1.13.0",
    Cmd: []string{
      "./daprd",
      "--app-id", "test-service",
      "--app-port", strconv.Itoa(appPort),
      "--dapr-http-port", "3500",
    },
    ExposedPorts: []string{"3500/tcp"},
    WaitingFor: wait.ForHTTP("/v1.0/healthz").
      WithPort("3500/tcp").
      WithStartupTimeout(30 * time.Second),
  }

  container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
    ContainerRequest: req,
    Started:          true,
  })
  require.NoError(t, err)

  return container
}
```

### Coverage-Friendly Fire-and-Forget Test
```go
// Source: Advanced testing in Go
func TestRedirect_WithDaprClient_PublishesEvent(t *testing.T) {
  // Use WaitGroup to make goroutine testable
  mockDapr := mocks.NewMockDaprClient(t)
  var wg sync.WaitGroup
  wg.Add(1)

  mockDapr.EXPECT().PublishEvent(mock.Anything, "pubsub", "clicks", mock.Anything).
    Run(func(args mock.Arguments) {
      defer wg.Done()
      // Verify event payload
      data := args.Get(3).([]byte)
      var event events.ClickEvent
      json.Unmarshal(data, &event)
      assert.Equal(t, "abc123", event.ShortCode)
    }).
    Return(nil)

  // Create handler with mock Dapr client
  handler := NewHandler(service, baseURL, mockDapr, logger, db)

  // ... trigger redirect ...

  // Wait for goroutine to complete
  done := make(chan struct{})
  go func() {
    wg.Wait()
    close(done)
  }()

  select {
  case <-done:
    // Success
  case <-time.After(1 * time.Second):
    t.Fatal("publishClickEvent goroutine did not complete")
  }
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| go test -cover manual checks | go-test-coverage with .coverage.yaml | 2024+ | Automated enforcement in CI, file/package/total thresholds |
| Docker Compose scripts for integration tests | testcontainers-go | 2023+ | Parallel test isolation, automatic cleanup, wait strategies |
| Anonymous test functions | t.Run with scenario names | Go 1.7+ | Better test output, can run specific subtests |
| gomock | testify/mock with mockery | 2022+ | Simpler syntax, better IDE support, fewer code generation steps |

**Deprecated/outdated:**
- Manual Docker scripts in test/integration/: testcontainers-go is the standard for Go projects
- Hardcoded sleep in rate limiter tests: Use token bucket inspection or mock time

## Open Questions

1. **Should Readyz Dapr health check be mocked or require real Dapr?**
   - What we know: Current implementation hardcodes localhost:3500 check
   - What's unclear: Whether to make URL configurable, mock HTTP client, or always test with real Dapr
   - Recommendation: Add handler tests with mock HTTP client for coverage, add integration test with real Dapr for verification

2. **Should integration tests run in CI on every commit or only on main?**
   - What we know: testcontainers adds ~10-30s overhead per test
   - What's unclear: Balance between CI speed and integration coverage
   - Recommendation: Run in CI on all commits (already have Docker); use `testing.Short()` flag for fast local testing

3. **What happens to click events during Dapr sidecar downtime?**
   - What we know: publishClickEvent logs errors but doesn't retry
   - What's unclear: If lost events are acceptable or need dead-letter queue
   - Recommendation: Document as known limitation; add integration test to verify graceful degradation

## Sources

### Primary (HIGH confidence)
- [Go Wiki: TableDrivenTests](https://go.dev/wiki/TableDrivenTests) - Official Go documentation for table-driven test patterns
- [httptest package documentation](https://pkg.go.dev/net/http/httptest) - Standard library HTTP testing utilities
- [testcontainers-go GitHub repository](https://github.com/testcontainers/testcontainers-go) - Official testcontainers for Go
- [testcontainers-go documentation](https://golang.testcontainers.org/) - Official usage guides and wait strategies
- [go-test-coverage GitHub](https://github.com/vladopajic/go-test-coverage) - Coverage threshold enforcement tool (already in use)
- [Dapr Go SDK documentation](https://docs.dapr.io/developing-applications/sdks/go/go-client/) - Official Dapr client testing guidance

### Secondary (MEDIUM confidence)
- [Go Testing Excellence: Table-Driven Tests and Mocking](https://dasroot.net/posts/2026/01/go-testing-excellence-table-driven-tests-mocking/) - 2026 Go testing best practices
- [Testing HTTP Handlers in Go](https://www.cloudbees.com/blog/testing-http-handlers-go) - httptest patterns for handlers and middleware
- [Advanced Mocking Techniques with Testify](https://tillitsdone.com/blogs/advanced-mocking-with-testify/) - testify mock.MatchedBy and error path testing
- [testcontainers-dapr-example](https://github.com/etiennetremel/testcontainers-dapr-example) - Dapr integration test reference implementation
- [How-To: Run Dapr in self-hosted mode with Docker](https://docs.dapr.io/operations/hosting/self-hosted/self-hosted-with-docker/) - Docker Compose with Dapr sidecars

### Tertiary (LOW confidence)
- [Better Stack: Testing in Go with Testify](https://betterstack.com/community/guides/scaling-go/golang-testify/) - General testify usage guide
- [Go Test Coverage blog post](https://go.dev/blog/cover) - Background on coverage tooling

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries verified in official docs and already in use (except testcontainers-go)
- Architecture: HIGH - Patterns verified in Go wiki, httptest docs, testcontainers docs
- Pitfalls: HIGH - Common issues documented in official sources and project history

**Research date:** 2026-02-15
**Valid until:** 2026-03-15 (30 days - stable ecosystem)
