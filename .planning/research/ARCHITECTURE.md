# Architecture Research

**Domain:** Go + Dapr URL Shortener Microservices
**Researched:** 2026-02-14
**Confidence:** HIGH

## Standard Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Client (HTTP)                               │
└────────────────────────────────┬────────────────────────────────────┘
                                 │
┌────────────────────────────────┴────────────────────────────────────┐
│                       URL Service (Port 8080)                        │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ Dapr Sidecar (3500) - Service Invocation + State + Pub/Sub  │   │
│  └──────────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Handler → Service → Repository                              │   │
│  │  - POST /shorten     (create short URL)                      │   │
│  │  - GET /{code}       (redirect)                              │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  Publishes: "url.clicked" events                                    │
└──────────────────────────┬───────────────────┬───────────────────────┘
                           │                   │
                           │ Dapr State API    │ Dapr Pub/Sub
                           ↓                   ↓
                    ┌─────────────┐     ┌────────────────┐
                    │   SQLite    │     │  Pub/Sub Topic │
                    │  (urls.db)  │     │  "url.clicked" │
                    └─────────────┘     └────────┬───────┘
                                                 │
┌────────────────────────────────────────────────┴─────────────────────┐
│                   Analytics Service (Port 8081)                      │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │ Dapr Sidecar (3501) - Pub/Sub Subscription + State          │   │
│  └──────────────────────────────────────────────────────────────┘   │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │  Handler → Service → Repository                              │   │
│  │  - GET /analytics/{code}  (get click stats)                  │   │
│  │  - Subscriber: "url.clicked" (process clicks)                │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                      │
│  Subscribes: "url.clicked" events                                   │
└──────────────────────────────────┬───────────────────────────────────┘
                                   │
                                   │ Dapr State API
                                   ↓
                            ┌──────────────┐
                            │    SQLite    │
                            │(analytics.db)│
                            └──────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| **URL Service** | Owns URL shortening and redirection logic | Go HTTP server with Dapr SDK for state and pub/sub |
| **Analytics Service** | Owns click tracking and analytics aggregation | Go HTTP server with Dapr pub/sub subscription handler |
| **Dapr Sidecar** | Service communication, state management, pub/sub | Dapr runtime process (one per service) |
| **SQLite State Store** | Persistent key-value storage for URLs and analytics | Dapr state component configured for SQLite |
| **Pub/Sub Broker** | Asynchronous event delivery between services | Dapr pub/sub component (Redis/Kafka in prod, in-memory for dev) |

## Recommended Project Structure

```
go-shortener/
├── cmd/
│   ├── url-service/       # URL Service entry point
│   │   └── main.go        # Server initialization, Dapr setup
│   └── analytics-service/ # Analytics Service entry point
│       └── main.go        # Server initialization, pub/sub subscription
├── internal/
│   ├── url/               # URL Service domain (not importable externally)
│   │   ├── handler/       # HTTP handlers (Presentation layer)
│   │   │   ├── shorten.go # POST /shorten handler
│   │   │   └── redirect.go# GET /{code} handler
│   │   ├── service/       # Business logic (Use cases)
│   │   │   ├── url_service.go
│   │   │   └── url_service_test.go
│   │   ├── repository/    # Data access (Infrastructure)
│   │   │   ├── url_repository.go
│   │   │   └── dapr_state_repository.go
│   │   └── domain/        # Domain models and interfaces
│   │       ├── url.go     # URL entity
│   │       └── errors.go  # Domain errors
│   ├── analytics/         # Analytics Service domain
│   │   ├── handler/       # HTTP + pub/sub handlers
│   │   │   ├── analytics.go    # GET /analytics/{code} handler
│   │   │   └── subscriber.go   # Dapr pub/sub event handler
│   │   ├── service/       # Business logic
│   │   │   ├── analytics_service.go
│   │   │   └── analytics_service_test.go
│   │   ├── repository/    # Data access
│   │   │   ├── analytics_repository.go
│   │   │   └── dapr_state_repository.go
│   │   └── domain/        # Domain models
│   │       ├── analytics.go    # Click event, stats
│   │       └── errors.go
│   └── shared/            # Shared utilities (cross-service)
│       ├── config/        # Configuration loading
│       ├── logger/        # Logging setup
│       └── middleware/    # HTTP middleware (auth, tracing)
├── pkg/                   # Public libraries (importable by other projects)
│   └── dapr/              # Dapr client wrappers/helpers
│       ├── state.go       # State management helpers
│       └── pubsub.go      # Pub/sub helpers
├── deployments/
│   ├── docker/            # Docker Compose for local dev
│   │   └── docker-compose.yml
│   └── dapr/              # Dapr component configurations
│       ├── state-sqlite.yaml      # SQLite state store config
│       └── pubsub-redis.yaml      # Redis pub/sub config
├── test/                  # Integration and e2e tests
│   ├── integration/       # Service-level integration tests
│   └── e2e/              # End-to-end scenario tests
├── scripts/               # Build, test, deployment scripts
├── .github/
│   └── workflows/         # CI/CD pipelines
│       └── ci.yml
├── go.mod
├── go.sum
├── Makefile              # Common tasks (test, build, docker, dapr)
└── README.md
```

### Structure Rationale

- **`cmd/`**: Each microservice gets its own entry point. Keeps `main.go` minimal (setup and wiring only).
- **`internal/`**: Private code organized by domain/service. Prevents external imports and enforces boundaries.
- **Clean Architecture layers**: Handler → Service → Repository separation allows isolated testing and swappable implementations.
- **`pkg/`**: Reusable Dapr helpers that could be extracted to a library. Use sparingly.
- **`deployments/`**: Infrastructure-as-code lives with the application for consistency.
- **Monorepo approach**: Two services in one repo simplifies local development and shared types. Can split later if needed.

## Architectural Patterns

### Pattern 1: Sidecar Architecture with Dapr

**What:** Each microservice runs alongside a Dapr sidecar process. The service communicates with Dapr via HTTP/gRPC on localhost. Dapr handles cross-cutting concerns (state, pub/sub, service invocation, observability).

**When to use:** Always with Dapr. This is the fundamental pattern.

**Trade-offs:**
- **Pro**: Decouples infrastructure concerns from business logic. Language-agnostic. Portable across environments.
- **Pro**: Built-in retries, circuit breakers, mTLS, distributed tracing without custom code.
- **Con**: Adds operational complexity (must deploy and monitor sidecars).
- **Con**: Local latency overhead (~1-2ms) for sidecar HTTP calls.

**Example:**
```go
// internal/url/repository/dapr_state_repository.go
package repository

import (
    "context"
    dapr "github.com/dapr/go-sdk/client"
)

type DaprURLRepository struct {
    client     dapr.Client
    storeName  string
}

func (r *DaprURLRepository) Save(ctx context.Context, code, longURL string) error {
    // Dapr handles connection pooling, retries, serialization
    return r.client.SaveState(ctx, r.storeName, code, []byte(longURL), nil)
}

func (r *DaprURLRepository) Get(ctx context.Context, code string) (string, error) {
    item, err := r.client.GetState(ctx, r.storeName, code, nil)
    if err != nil {
        return "", err
    }
    return string(item.Value), nil
}
```

### Pattern 2: Clean Architecture (Handler → Service → Repository)

**What:** Three-layer architecture with clear dependency direction (Handler depends on Service, Service depends on Repository interfaces). Domain logic in Service layer, I/O in Repository layer, HTTP in Handler layer.

**When to use:** For all microservices. Mandatory for maintainability and testability.

**Trade-offs:**
- **Pro**: Easy to test (mock Repository interface). Easy to swap implementations (SQLite → Postgres).
- **Pro**: Clear separation of concerns. Business logic isolated from frameworks.
- **Con**: More files and indirection compared to "everything in main.go" approach.
- **Con**: Can feel over-engineered for trivial CRUD operations.

**Example:**
```go
// internal/url/domain/url.go
package domain

type URL struct {
    Code      string
    LongURL   string
    CreatedAt time.Time
}

// internal/url/service/url_service.go
package service

type URLService struct {
    repo       URLRepository // interface, not concrete type
    eventBus   EventPublisher
}

func (s *URLService) ShortenURL(ctx context.Context, longURL string) (string, error) {
    // Business logic: validate, generate code, check collision
    code := generateShortCode()

    if err := s.repo.Save(ctx, code, longURL); err != nil {
        return "", err
    }

    return code, nil
}

func (s *URLService) RedirectURL(ctx context.Context, code string) (string, error) {
    longURL, err := s.repo.Get(ctx, code)
    if err != nil {
        return "", err
    }

    // Publish click event asynchronously
    s.eventBus.Publish(ctx, "url.clicked", ClickEvent{Code: code, Timestamp: time.Now()})

    return longURL, nil
}
```

### Pattern 3: Event-Driven Communication (Pub/Sub)

**What:** URL Service publishes `url.clicked` events when redirects occur. Analytics Service subscribes to this topic. Services are decoupled—URL Service doesn't know Analytics exists.

**When to use:** For asynchronous, non-critical workflows where services don't need immediate responses.

**Trade-offs:**
- **Pro**: Decouples services. URL redirection stays fast (no blocking on analytics writes).
- **Pro**: Easy to add more subscribers later (e.g., fraud detection, A/B testing).
- **Con**: Eventual consistency. Analytics may lag behind real-time clicks.
- **Con**: Debugging is harder (events are fire-and-forget).

**Example:**
```go
// URL Service publishes
func (s *URLService) RedirectURL(ctx context.Context, code string) (string, error) {
    // ... get longURL from repo ...

    event := ClickEvent{
        Code:      code,
        Timestamp: time.Now(),
        UserAgent: getUserAgent(ctx),
        IPAddress: getClientIP(ctx),
    }

    // Fire and forget - don't wait for analytics
    go s.daprClient.PublishEvent(ctx, "pubsub", "url.clicked", event)

    return longURL, nil
}

// Analytics Service subscribes
// cmd/analytics-service/main.go
func main() {
    s := daprd.NewService(":8081")

    subscription := &common.Subscription{
        PubsubName: "pubsub",
        Topic:      "url.clicked",
        Route:      "/events/url-clicked",
    }

    s.AddTopicEventHandler(subscription, clickEventHandler)
    s.Start()
}

func clickEventHandler(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
    var event ClickEvent
    json.Unmarshal(e.RawData, &event)

    // Increment click counter in state store
    return false, analyticsService.RecordClick(ctx, event)
}
```

### Pattern 4: State Store Abstraction

**What:** Use Dapr's state management API instead of direct database access. State is key-value, not relational. Dapr provides CRUD, transactions, TTL, and concurrency control via ETags.

**When to use:** For simple key-value access patterns. SQLite is the backing store, but code never touches SQL.

**Trade-offs:**
- **Pro**: Portable across state stores (Redis, Postgres, Cosmos DB). No SQL migrations.
- **Pro**: Built-in features (optimistic locking with ETags, TTL, bulk operations).
- **Con**: Limited querying (no complex joins or aggregations). Use Query API or native SQL for analytics.
- **Con**: Key design is critical (can't change keys easily).

**Example:**
```go
// Save URL with TTL (auto-expire after 30 days)
metadata := map[string]string{
    "ttlInSeconds": "2592000", // 30 days
}

err := daprClient.SaveState(ctx, "urlstore", code, longURL, metadata)

// Optimistic concurrency with ETag
item, _ := daprClient.GetState(ctx, "urlstore", code, nil)
etag := item.Etag

// Update only if ETag matches (no one else modified)
err = daprClient.SaveState(ctx, "urlstore", code, newValue, &dapr.StateOptions{
    Concurrency: dapr.StateConcurrencyFirstWrite,
    Etag:        &etag,
})
```

## Data Flow

### Request Flow: URL Shortening

```
POST /shorten {longURL: "https://example.com/very/long/path"}
    ↓
[URL Service Handler] → Validate request
    ↓
[URL Service] → Generate short code (e.g., "abc123")
    ↓
[URL Repository] → Save to Dapr State: key="abc123", value="https://example.com/..."
    ↓
[Dapr Sidecar] → HTTP POST to localhost:3500/v1.0/state/urlstore
    ↓
[SQLite State Store Component] → INSERT INTO state (key, value) VALUES (...)
    ↓
Response: {"shortURL": "http://short.ly/abc123"}
```

### Request Flow: URL Redirection + Analytics

```
GET /abc123
    ↓
[URL Service Handler] → Extract code "abc123"
    ↓
[URL Service] → Get long URL from repository
    ↓
[URL Repository] → Dapr State: GET localhost:3500/v1.0/state/urlstore/abc123
    ↓
[SQLite] → SELECT value FROM state WHERE key = "abc123"
    ↓
[URL Service] → Publish event: {topic: "url.clicked", data: {code: "abc123", ts: ...}}
    ↓
[Dapr Sidecar] → POST localhost:3500/v1.0/publish/pubsub/url.clicked
    ↓
[Pub/Sub Broker (Redis)] → Queue event for subscribers
    ↓
Response: HTTP 302 Redirect → "https://example.com/..."

    (Asynchronously in parallel)
    ↓
[Analytics Service] → Receives event from Dapr: POST /events/url-clicked
    ↓
[Analytics Service] → Increment click count in state store
    ↓
[Dapr Sidecar] → GET/POST localhost:3501/v1.0/state/analyticsstore/abc123
    ↓
[SQLite Analytics DB] → UPDATE analytics SET clicks = clicks + 1 WHERE code = "abc123"
```

### Service-to-Service Communication (Optional: Direct Invocation)

If URL Service needs to query analytics directly (not used in base design, but available):

```
[URL Service] → daprClient.InvokeMethod(ctx, "analytics-service", "analytics/abc123")
    ↓
[Dapr Sidecar 3500] → Service discovery: find analytics-service
    ↓
[Dapr Sidecar 3501] → HTTP GET http://localhost:8081/analytics/abc123
    ↓
[Analytics Service Handler] → Return click stats
```

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| **0-10k URLs** | Single instance of each service. SQLite is sufficient. In-memory pub/sub (no Redis needed). Run locally with `docker-compose`. |
| **10k-100k URLs** | Horizontal scale URL Service (stateless, easy). Switch to Redis/Kafka for pub/sub (durable, not in-memory). SQLite still OK for state stores (Dapr serializes writes). Add caching layer (Redis cache component) for hot URLs. |
| **100k+ URLs** | Migrate state stores to Postgres/Cosmos DB (better concurrent writes). Partition analytics by date/region. Consider CDN for redirect endpoint. Implement read replicas for analytics queries. |

### Scaling Priorities

1. **First bottleneck:** Redirect latency (hot URLs).
   - **Fix:** Add Dapr state store cache layer (Redis). Cache short code → long URL mappings with TTL.
   - **Implementation:** Dapr configuration change only (no code change). Set `queryIndexName` in state config.

2. **Second bottleneck:** Analytics writes under high traffic.
   - **Fix:** Batch click events before writing to state store. Use Dapr bulk state API.
   - **Implementation:** Buffer events in-memory (100 events or 1 second), then bulk write.

3. **Third bottleneck:** SQLite write contention.
   - **Fix:** Migrate to Postgres/MySQL with connection pooling. Change Dapr component config only.

## Anti-Patterns

### Anti-Pattern 1: Direct Service-to-Service HTTP Calls

**What people do:** URL Service makes direct HTTP calls to Analytics Service (e.g., `http.Post("http://analytics:8081/track", ...)`).

**Why it's wrong:**
- Tight coupling: URL Service needs to know Analytics Service's address and API contract.
- Synchronous blocking: Redirect waits for analytics write to complete (adds latency).
- Fragile: If Analytics is down, redirects fail (availability coupling).
- No retries or circuit breaking without custom code.

**Do this instead:** Use Dapr pub/sub for asynchronous events OR Dapr service invocation for synchronous RPC (if truly needed). For analytics, pub/sub is correct—redirects should never wait for analytics.

### Anti-Pattern 2: Shared Database Between Services

**What people do:** Both URL Service and Analytics Service write to the same SQLite database file or same tables.

**Why it's wrong:**
- Violates microservices principle: services should own their data.
- Schema coupling: changing URL schema breaks Analytics Service.
- Transaction boundaries unclear: who owns data consistency?
- SQLite doesn't support concurrent writes well across processes.

**Do this instead:** Each service has its own state store (can be same SQLite binary, but different component names and keyspaces). Use events/APIs to share data.

### Anti-Pattern 3: Business Logic in Handlers

**What people do:** HTTP handler directly calls Dapr client and implements URL generation logic.

**Why it's wrong:**
- Impossible to test without spinning up HTTP server and Dapr.
- Logic is tied to HTTP framework (can't reuse for gRPC, CLI).
- Hard to reason about business rules scattered across handlers.

**Do this instead:** Handlers are thin adapters. Parse request → call Service layer → serialize response. Service layer has zero HTTP dependencies.

### Anti-Pattern 4: Ignoring Dapr Health Checks and Graceful Shutdown

**What people do:** Service starts immediately without waiting for Dapr sidecar to be ready. On shutdown, service exits before flushing Dapr events.

**Why it's wrong:**
- Service crashes on startup if Dapr isn't ready (HTTP 500 errors).
- Published events may be lost during shutdown.
- Kubernetes sees service as healthy before it's actually ready.

**Do this instead:**
```go
// Wait for Dapr sidecar to be ready
func waitForDapr(ctx context.Context) error {
    client := &http.Client{Timeout: 1 * time.Second}
    daprPort := os.Getenv("DAPR_HTTP_PORT")
    if daprPort == "" {
        daprPort = "3500"
    }

    healthURL := fmt.Sprintf("http://localhost:%s/v1.0/healthz", daprPort)

    for {
        resp, err := client.Get(healthURL)
        if err == nil && resp.StatusCode == 204 {
            return nil
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(100 * time.Millisecond):
        }
    }
}

// Graceful shutdown
func main() {
    // ... setup ...

    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Close Dapr client (flushes pending events)
    daprClient.Close()

    // Shutdown HTTP server
    httpServer.Shutdown(ctx)
}
```

### Anti-Pattern 5: Not Using Dapr Component Scoping

**What people do:** All services can access all Dapr components (state stores, pub/sub topics).

**Why it's wrong:**
- URL Service could accidentally read Analytics state store.
- Security risk: no isolation between services.
- Harder to reason about dependencies.

**Do this instead:** Use Dapr component scoping in configuration:
```yaml
# deployments/dapr/state-url.yaml
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: urlstore
spec:
  type: state.sqlite
  metadata:
    - name: connectionString
      value: "data/urls.db"
scopes:
  - url-service  # Only URL Service can use this
```

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| **SQLite State Store** | Dapr state component | File-based DB. One file per service. Use `connectionString` metadata to separate. |
| **Redis Pub/Sub** | Dapr pub/sub component | For production. Use streams or pub/sub mode. Configure in `pubsub-redis.yaml`. |
| **Observability (Zipkin/Jaeger)** | Dapr configuration | Automatic distributed tracing. Enable in Dapr config, no app code changes. |
| **Prometheus** | Dapr metrics endpoint | Dapr exposes metrics at `:9090/metrics`. Scrape from both app and sidecar. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| **URL Service ↔ Analytics Service** | Dapr Pub/Sub (async) | `url.clicked` topic. URL Service publishes, Analytics subscribes. At-least-once delivery. |
| **Service ↔ Dapr Sidecar** | HTTP (localhost:3500/3501) | Minimal latency. Use Dapr SDK (wraps HTTP/gRPC). Retry logic built-in. |
| **Dapr ↔ State Store** | Dapr component plugin | SQLite component translates Dapr API to SQL. Pluggable (swap to Postgres without code change). |
| **Dapr ↔ Pub/Sub Broker** | Dapr component plugin | Redis component handles connection pooling, retries, message formatting (CloudEvents). |

## Build Order Implications

Based on dependencies and complexity, recommended implementation order:

### Phase 1: Foundation (Week 1)
1. **Project structure setup**: Create `cmd/`, `internal/`, `pkg/` directories. Initialize Go modules.
2. **Dapr local setup**: Install Dapr CLI, initialize Dapr (`dapr init`). Verify sidecar runs.
3. **Basic URL Service skeleton**: Handler → Service → Repository interfaces (in-memory implementation first).

**Why first:** Establishes patterns. No external dependencies yet.

### Phase 2: URL Service Core (Week 1-2)
4. **Dapr state integration**: Swap in-memory repository for Dapr state client. Configure SQLite component.
5. **URL shortening logic**: Implement code generation, collision handling, validation.
6. **Redirect endpoint**: GET /{code} handler with Dapr state lookup.

**Why next:** Core value delivery. Can demo/test basic shortening without analytics.

### Phase 3: Event-Driven Analytics (Week 2)
7. **Publish click events**: Add Dapr pub/sub publish to redirect handler.
8. **Analytics Service subscriber**: New service with pub/sub topic handler. Increment counters in state store.
9. **Analytics query endpoint**: GET /analytics/{code} handler to retrieve stats.

**Why next:** Builds on URL Service. Introduces async patterns. Can validate pub/sub works.

### Phase 4: Production Readiness (Week 3)
10. **Testing**: Unit tests (mock repositories), integration tests (real Dapr + SQLite).
11. **Docker Compose**: Package both services + Dapr sidecars + Redis.
12. **CI/CD**: GitHub Actions for linting, testing, Docker builds.
13. **Observability**: Add structured logging, Dapr tracing config, health check endpoints.

**Why last:** Depends on functional code. These are cross-cutting concerns.

**Dependencies:**
- Phase 2 blocks Phase 3 (need URL Service to publish events).
- Phase 4 blocks nothing (can be done incrementally).

## Sources

**High Confidence (Official Docs):**
- [Dapr Overview](https://docs.dapr.io/concepts/overview/) — Sidecar architecture, building blocks
- [Dapr Service Invocation](https://docs.dapr.io/developing-applications/building-blocks/service-invocation/service-invocation-overview/) — Service-to-service patterns
- [Dapr Pub/Sub Overview](https://docs.dapr.io/developing-applications/building-blocks/pubsub/pubsub-overview/) — Event-driven patterns
- [Dapr State Management](https://docs.dapr.io/developing-applications/building-blocks/state-management/state-management-overview/) — State store usage
- [Dapr SQLite Component](https://docs.dapr.io/reference/components-reference/supported-state-stores/setup-sqlite/) — SQLite configuration

**Medium Confidence (Reference Implementations):**
- [Dapr Store Reference App](https://code.benco.io/dapr-store/) — Go microservices with Dapr, Standard Project Layout
- [Dapr as Microservices Patterns Framework](https://www.diagrid.io/blog/dapr-as-the-ultimate-microservices-patterns-framework) — Sidecar, saga, retry patterns

**Medium Confidence (Best Practices):**
- [Go Clean Architecture](https://threedots.tech/post/introducing-clean-architecture/) — Handler/Service/Repository pattern
- [Go Standard Project Layout](https://github.com/golang-standards/project-layout) — Project structure conventions
- [URL Shortener System Design](https://systemdesignschool.io/problems/url-shortener/solution) — Architecture patterns for URL shorteners

---
*Architecture research for: Go + Dapr URL Shortener Microservices*
*Researched: 2026-02-14*
