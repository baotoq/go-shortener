# Phase 2: Event-Driven Analytics - Research

**Researched:** 2026-02-15
**Domain:** Dapr pub/sub, Go multi-service architecture, event-driven analytics
**Confidence:** HIGH

## Summary

Phase 2 introduces asynchronous click tracking via Dapr pub/sub, requiring a multi-service Go architecture with two separate HTTP servers (URL Service and Analytics Service) communicating through Dapr's in-memory pub/sub component. The architecture follows clean layering patterns established in Phase 1, extended to support event publishing and subscription.

The core technical challenges are: (1) structuring a single Go module with two binaries under `cmd/`, (2) implementing fire-and-forget publish patterns with appropriate error handling, (3) creating programmatic subscriptions via the `/dapr/subscribe` endpoint, and (4) managing two separate SQLite databases with proper isolation. Dapr's CloudEvents 1.0 envelope wraps all published messages automatically, requiring event handler code to unwrap the payload.

The in-memory pub/sub component is suitable for Phase 2's learning objectives but has critical limitations: messages are lost on restart, and state is not replicated across sidecars. This is acceptable for development but must be acknowledged in implementation.

**Primary recommendation:** Use Dapr Go SDK's `client.PublishEvent()` for fire-and-forget publishing and `service.AddTopicEventHandler()` for subscriptions. Structure the repo with `cmd/url-service/` and `cmd/analytics-service/` binaries sharing common event types from `internal/shared/events/`. Configure each service with distinct Dapr app-ids and ports via `dapr.yaml` multi-app template for simplified local development.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Click event payload
- Minimal payload: short code + timestamp only
- No IP, User-Agent, or Referer in Phase 2 — Phase 3 will extend the event schema when needed
- Individual click records stored (one row per click), not just counters
- Inline fire-and-forget publish during redirect handler — no internal queue
- On pub/sub failure: log the error and continue with redirect (click is lost, redirect succeeds)

#### Analytics API shape
- Endpoint: `GET /analytics/{code}` on the Analytics Service directly (separate port)
- Response: `{"short_code": "abc", "total_clicks": 42}` — total count only for Phase 2
- Zero clicks returns `{"short_code": "abc", "total_clicks": 0}` — not 404
- Analytics Service has its own HTTP server, not proxied through URL Service

#### Project structure
- Single Go module, two binaries: `cmd/url-service/` and `cmd/analytics-service/`
- Restructure repo: move existing URL Service code under `services/` directory layout
- Separate SQLite database file for Analytics Service (analytics.db) — true service data isolation
- Shared internal package for common types (e.g., click event struct in `internal/shared/` or `pkg/events/`)

#### Dapr pub/sub setup
- In-memory pub/sub component for local development (zero dependencies, messages lost on restart is acceptable)
- Dapr component files in root `dapr/components/` directory — shared across services
- Programmatic subscription via `/dapr/subscribe` endpoint in Analytics Service code
- Makefile targets for running services: `make run-url`, `make run-analytics` wrapping `dapr run`

### Claude's Discretion

Research options and recommend:
- Topic naming convention
- Dapr app-id naming for each service
- Exact port assignments for each service
- Analytics Service clean architecture layer structure (mirror URL Service patterns)
- SQLite schema for click records table
- Error response format for Analytics API (follow RFC 7807 pattern from Phase 1)

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope

</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Dapr Go SDK | Latest (dapr/go-sdk) | Pub/sub client + service integration | Official Dapr SDK with idiomatic Go patterns for publishing and subscribing |
| Chi router | v5 (go-chi/chi/v5) | HTTP routing (already in use) | Continue existing Phase 1 stack for consistency |
| SQLite + mattn/go-sqlite3 | Latest | Analytics database | Continue Phase 1 pattern with separate database file |
| sqlc | Latest | Type-safe SQL queries | Continue Phase 1 code generation pattern |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Zap logger | Latest | Structured logging (already in use) | All pub/sub error logging, event tracking |
| golang-migrate | Latest | Database migrations | Analytics Service schema setup |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Programmatic subscription | Declarative YAML subscription | Programmatic keeps all logic in code; declarative requires separate config file but simpler |
| In-memory pub/sub | Redis Streams | Redis requires external dependency; in-memory is zero-config for learning |

**Installation:**
```bash
go get github.com/dapr/go-sdk
# Other dependencies already installed in Phase 1
```

## Architecture Patterns

### Recommended Project Structure
```
go-shortener/
├── cmd/
│   ├── url-service/
│   │   └── main.go              # URL Service entry point
│   └── analytics-service/
│       └── main.go              # Analytics Service entry point
├── internal/
│   ├── shared/
│   │   └── events/
│   │       └── click_event.go   # Shared event struct
│   ├── urlservice/              # URL Service moved here
│   │   ├── handler/
│   │   ├── usecase/
│   │   └── repository/
│   └── analytics/               # Analytics Service (mirrors URL Service structure)
│       ├── handler/             # HTTP handlers
│       ├── usecase/             # Business logic
│       └── repository/          # SQLite repository
├── dapr/
│   └── components/
│       └── pubsub.yaml          # In-memory pub/sub component
├── migrations/
│   ├── url/                     # URL Service migrations (existing)
│   └── analytics/               # Analytics Service migrations (new)
└── dapr.yaml                    # Multi-app run template
```

### Pattern 1: Fire-and-Forget Publishing

**What:** Publish click events without waiting for acknowledgment or retries
**When to use:** After successful redirect, before returning HTTP response
**Example:**
```go
// Source: https://docs.dapr.io/developing-applications/building-blocks/pubsub/howto-publish-subscribe/
import (
	"context"
	"encoding/json"
	"log"
	dapr "github.com/dapr/go-sdk/client"
)

// In redirect handler (after successful redirect):
func (h *Handler) publishClickEvent(shortCode string) {
	client, err := dapr.NewClient()
	if err != nil {
		// Log error and continue - redirect already succeeded
		log.Printf("Failed to create Dapr client: %v", err)
		return
	}
	defer client.Close()

	event := events.ClickEvent{
		ShortCode: shortCode,
		Timestamp: time.Now().UTC(),
	}

	data, _ := json.Marshal(event)
	ctx := context.Background()

	if err := client.PublishEvent(ctx, "pubsub", "clicks", data); err != nil {
		// Log error and continue - click is lost, redirect succeeds
		log.Printf("Failed to publish click event: %v", err)
	}
}
```

### Pattern 2: Programmatic Subscription

**What:** Implement `/dapr/subscribe` endpoint to register topic subscriptions at startup
**When to use:** Analytics Service startup - subscriptions are read once, not dynamic
**Example:**
```go
// Source: https://docs.dapr.io/developing-applications/building-blocks/pubsub/subscription-methods/
import (
	"context"
	"log"
	"net/http"
	"github.com/dapr/go-sdk/service/common"
	daprd "github.com/dapr/go-sdk/service/http"
)

func main() {
	s := daprd.NewService(":8081") // Analytics Service port

	sub := &common.Subscription{
		PubsubName: "pubsub",
		Topic:      "clicks",
		Route:      "/events/click",
	}

	if err := s.AddTopicEventHandler(sub, handleClickEvent); err != nil {
		log.Fatalf("error adding topic subscription: %v", err)
	}

	if err := s.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("error listening: %v", err)
	}
}

func handleClickEvent(ctx context.Context, e *common.TopicEvent) (retry bool, err error) {
	// Dapr unwraps CloudEvent envelope automatically
	// e.Data contains the click event JSON
	var event events.ClickEvent
	if err := json.Unmarshal(e.RawData, &event); err != nil {
		log.Printf("Failed to unmarshal click event: %v", err)
		return false, nil // Don't retry on malformed events
	}

	// Process event (save to database)
	log.Printf("Received click event: %+v", event)
	return false, nil
}
```

### Pattern 3: Multi-Service Dapr Configuration

**What:** Use `dapr.yaml` to configure both services with unique app-ids and ports
**When to use:** Local development with `dapr run -f dapr.yaml`
**Example:**
```yaml
# Source: https://docs.dapr.io/developing-applications/local-development/multi-app-dapr-run/multi-app-template/
version: 1
common:
  resourcesPath: ./dapr/components
apps:
  - appID: url-service
    appDirPath: ./
    appPort: 8080
    daprHTTPPort: 3500
    daprGRPCPort: 50001
    command: ["go", "run", "./cmd/url-service"]

  - appID: analytics-service
    appDirPath: ./
    appPort: 8081
    daprHTTPPort: 3501
    daprGRPCPort: 50002
    command: ["go", "run", "./cmd/analytics-service"]
```

### Pattern 4: Shared Event Types in internal/

**What:** Common event structs in `internal/shared/events/` accessible to both services
**When to use:** Event definitions consumed by publisher and subscriber
**Example:**
```go
// internal/shared/events/click_event.go
package events

import "time"

// ClickEvent represents a URL click for analytics tracking
type ClickEvent struct {
	ShortCode string    `json:"short_code"`
	Timestamp time.Time `json:"timestamp"`
}
```

### Pattern 5: Clean Architecture for Analytics Service

**What:** Mirror URL Service's three-layer structure (handler → usecase → repository)
**When to use:** Analytics Service implementation
**Structure:**
```go
// internal/analytics/handler/analytics_handler.go
type AnalyticsHandler struct {
	analyticsUseCase usecase.AnalyticsUseCase
}

// internal/analytics/usecase/analytics_usecase.go
type AnalyticsUseCase interface {
	GetClickCount(ctx context.Context, shortCode string) (int64, error)
	RecordClick(ctx context.Context, event events.ClickEvent) error
}

// internal/analytics/repository/sqlite_repository.go
type ClickRepository interface {
	InsertClick(ctx context.Context, shortCode string, timestamp time.Time) error
	CountClicks(ctx context.Context, shortCode string) (int64, error)
}
```

### Anti-Patterns to Avoid

- **Blocking redirect on publish:** Never `time.Sleep()` or retry publish failures - fire-and-forget means log and continue
- **Shared database between services:** Each service must have its own SQLite file for true isolation
- **CloudEvents manual wrapping:** Dapr automatically wraps messages; don't manually create CloudEvent structs
- **Dynamic subscriptions at runtime:** Programmatic subscriptions are read once at startup; changing subscriptions requires restart

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Pub/sub messaging | Custom message queue, channels | Dapr pub/sub | Handles CloudEvents wrapping, retries, dead letters, broker abstraction |
| Event envelope format | Custom JSON wrapper | CloudEvents 1.0 (via Dapr) | Industry standard with tracing, versioning, metadata built-in |
| Multi-service orchestration | Shell scripts with & | `dapr run -f dapr.yaml` | Unified startup, port management, component sharing |
| Database connection pooling | Manual connection management | Continue existing patterns from Phase 1 | Already proven in URL Service |

**Key insight:** Dapr abstracts away pub/sub broker implementation details. Don't build custom retry logic, message formatting, or service discovery - Dapr handles these concerns. Focus on business logic (what to publish, how to handle events).

## Common Pitfalls

### Pitfall 1: /dapr/subscribe Endpoint Not Called

**What goes wrong:** Dapr cannot discover subscriptions, events are never delivered
**Why it happens:** Endpoint not registered, returns non-200 status, or malformed JSON
**How to avoid:** Use Dapr Go SDK's `service.AddTopicEventHandler()` - it automatically registers the endpoint correctly
**Warning signs:** Dapr logs show no subscriptions during startup, or "no active subscriptions" warnings

### Pitfall 2: Assuming Publish Succeeds

**What goes wrong:** Silent failures when Dapr sidecar is down or component misconfigured
**Why it happens:** Fire-and-forget means no error propagation back to caller
**How to avoid:** Log ALL publish errors with structured fields (short_code, error message). Monitor logs for patterns indicating Dapr unavailability
**Warning signs:** Zero click events in Analytics Service despite redirect traffic

### Pitfall 3: SQLite "Database is Locked" Errors

**What goes wrong:** Concurrent write transactions fail with SQLITE_BUSY
**Why it happens:** SQLite allows only one writer at a time, even in WAL mode
**How to avoid:** Keep transactions short, use single connection pool per database (already in Phase 1), INSERT operations should complete quickly
**Warning signs:** "database is locked" errors in logs during high click volume

### Pitfall 4: Forgetting Dapr Sidecar Restart After Component Changes

**What goes wrong:** Component configuration changes (e.g., topic name) don't take effect
**Why it happens:** Dapr reads component files at startup, not dynamically
**How to avoid:** Always restart `dapr run` after editing component YAML files
**Warning signs:** Old behavior persists despite configuration changes

### Pitfall 5: Port Conflicts Between Services

**What goes wrong:** Second service fails to start with "address already in use"
**Why it happens:** Both services or Dapr sidecars use same port
**How to avoid:** Explicitly configure unique ports in `dapr.yaml`: URL Service (8080, Dapr 3500/50001), Analytics Service (8081, Dapr 3501/50002)
**Warning signs:** Service startup failures, Dapr sidecar crashes

### Pitfall 6: CloudEvents Data Field Confusion

**What goes wrong:** Attempting to unmarshal entire CloudEvent envelope as application data
**Why it happens:** Not understanding Dapr wraps application payload in CloudEvents `data` field
**How to avoid:** Use `TopicEvent.RawData` or `TopicEvent.Data` from Dapr SDK - SDK handles unwrapping
**Warning signs:** JSON unmarshal errors, unexpected CloudEvent fields in application structs

### Pitfall 7: In-Memory Pub/Sub Message Loss

**What goes wrong:** Messages lost when Dapr sidecar restarts
**Why it happens:** In-memory component doesn't persist state - this is expected behavior
**How to avoid:** Accept this for Phase 2 learning; document limitation; don't rely on message durability
**Warning signs:** Missing click records after restart (this is normal, not a bug)

## Code Examples

Verified patterns from official sources:

### Dapr In-Memory Pub/Sub Component Configuration

```yaml
# Source: https://docs.dapr.io/reference/components-reference/supported-pubsub/setup-inmemory/
apiVersion: dapr.io/v1alpha1
kind: Component
metadata:
  name: pubsub
spec:
  type: pubsub.in-memory
  version: v1
  metadata: []
```

### CloudEvents Envelope Structure

```json
// Source: https://docs.dapr.io/developing-applications/building-blocks/pubsub/pubsub-cloudevents/
{
  "specversion": "1.0",
  "type": "com.dapr.event.sent",
  "source": "url-service",
  "id": "unique-event-id",
  "time": "2026-02-15T10:30:00Z",
  "datacontenttype": "application/json",
  "topic": "clicks",
  "pubsubname": "pubsub",
  "traceid": "00-trace-id-00",
  "traceparent": "00-trace-parent-00",
  "data": {
    "short_code": "abc123",
    "timestamp": "2026-02-15T10:30:00Z"
  }
}
```

### SQLite Click Records Schema

```sql
-- migrations/analytics/001_create_clicks_table.up.sql
-- Source: Best practices from https://www.sqliteforum.com/p/effective-schema-design-for-sqlite
CREATE TABLE IF NOT EXISTS clicks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code TEXT NOT NULL,
    clicked_at INTEGER NOT NULL,  -- Unix timestamp (seconds since epoch)
    FOREIGN KEY (short_code) REFERENCES urls(short_code) -- if cross-DB referential integrity needed
);

-- Index for fast count queries by short_code
CREATE INDEX idx_clicks_short_code ON clicks(short_code);

-- Index for time-range queries (Phase 3 preparation)
CREATE INDEX idx_clicks_timestamp ON clicks(clicked_at);
```

**Timestamp storage rationale:** Use INTEGER for Unix timestamps (4 bytes until 2038, then 6 bytes) rather than TEXT ISO8601 (20 bytes). Click analytics involves frequent timestamp calculations and range queries, making INTEGER more efficient. Store as UTC seconds since epoch.

### Analytics API Response Format

```go
// RFC 7807 error response (continue Phase 1 pattern)
// Source: https://github.com/lpar/problem
type AnalyticsResponse struct {
	ShortCode   string `json:"short_code"`
	TotalClicks int64  `json:"total_clicks"`
}

// Example success response:
// HTTP 200 OK
// {"short_code": "abc123", "total_clicks": 42}

// Example zero clicks (NOT 404):
// HTTP 200 OK
// {"short_code": "xyz789", "total_clicks": 0}
```

### Graceful Shutdown with Context (Both Services)

```go
// Source: https://victoriametrics.com/blog/go-graceful-shutdown/
import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Create context that cancels on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start HTTP server in goroutine
	srv := &http.Server{Addr: ":8080", Handler: router}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("Shutting down gracefully...")

	// Give server 30 seconds to finish in-flight requests
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
```

## Recommendations (Claude's Discretion)

### Topic Naming Convention
**Recommendation:** Use plural nouns for topics: `clicks`, `url-created`, `url-deleted`
**Rationale:** Topics represent streams of events (multiple occurrences). Plural naming is consistent with REST resource naming and clarifies that subscribers receive many events over time.

### Dapr App-ID Naming
**Recommendation:**
- URL Service: `url-service` (matches directory name)
- Analytics Service: `analytics-service` (matches directory name)

**Rationale:** Lowercase with hyphens follows Kubernetes/Docker naming conventions. App-ID should match the service's logical name and cmd/ directory for clarity. Cannot contain dots per Dapr requirements.

### Port Assignments
**Recommendation:**
```
URL Service:
  - HTTP: 8080 (application)
  - Dapr HTTP: 3500
  - Dapr gRPC: 50001

Analytics Service:
  - HTTP: 8081 (application)
  - Dapr HTTP: 3501
  - Dapr gRPC: 50002
```

**Rationale:**
- Application ports 8080/8081 are conventional HTTP alternates, easy to remember
- Dapr HTTP ports 3500/3501 follow Dapr defaults (3500) + increment
- Dapr gRPC ports 50001/50002 follow Dapr defaults (50001) + increment
- All ports are non-privileged (>1024), no sudo required

### Analytics Service Clean Architecture
**Recommendation:** Mirror URL Service three-layer structure exactly:

```
internal/analytics/
├── handler/           # HTTP layer (Chi routes, request/response)
│   └── analytics_handler.go
├── usecase/           # Business logic (click counting, event processing)
│   └── analytics_usecase.go
└── repository/        # Data access (SQLite queries via sqlc)
    └── sqlite_repository.go
```

**Rationale:** Consistency reduces cognitive load. Developers familiar with URL Service can navigate Analytics Service immediately. Same dependency injection patterns, same testing approach, same interface boundaries.

### SQLite Schema for Click Records
**Recommendation:** See "Code Examples" section above - separate table with indexes on short_code and timestamp, INTEGER Unix timestamps
**Rationale:** Individual row per click (not aggregated counters) supports Phase 3 enrichment (add columns for IP, User-Agent, etc. without schema changes). Indexes enable fast COUNT queries while preserving raw event data.

### Error Response Format for Analytics API
**Recommendation:** Continue RFC 7807 Problem Details pattern from Phase 1

**Success cases:** Return `AnalyticsResponse` JSON (200 OK)
**Error cases:** Return RFC 7807 Problem JSON with appropriate status codes:
- 400 Bad Request: Invalid short code format
- 500 Internal Server Error: Database failures
- 503 Service Unavailable: Database connection issues

**Rationale:** API consistency across services. Clients can use same error handling logic for both URL Service and Analytics Service.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Shell scripts to run multiple Dapr apps | Multi-App Run with dapr.yaml | Dapr 1.11+ (2023) | Single command starts all services with shared config |
| Declarative subscriptions (YAML) | Programmatic + Declarative + Streaming | Dapr 1.10+ (2023) | More flexibility, programmatic keeps subscription logic in code |
| Manual CloudEvent envelope creation | Automatic CloudEvents wrapping | CloudEvents 1.0 spec adopted early in Dapr | Developers work with application payloads, not envelopes |
| Go 1.13 context cancellation patterns | signal.NotifyContext | Go 1.16+ (2021) | Simpler graceful shutdown code, fewer goroutines |

**Deprecated/outdated:**
- Dapr standalone mode without multi-app run: Still works but requires multiple terminal windows and manual coordination
- Pre-1.0 CloudEvents formats: Dapr only supports CloudEvents 1.0 spec
- Go modules with replace directives for monorepos: Go workspaces (1.18+) provide cleaner solution, but not needed for single-module projects

## Open Questions

1. **Cross-service validation: Should Analytics Service validate short codes exist in URL Service database?**
   - What we know: Each service has separate database; no built-in referential integrity
   - What's unclear: Whether to call URL Service API to validate, or accept any short code
   - Recommendation: Accept any short code for Phase 2. Analytics Service is a passive recorder - it doesn't dictate business rules. If URL Service publishes an event, Analytics trusts it. Validation can be added in Phase 3 if needed.

2. **Event ordering: Does Analytics Service need to handle out-of-order events?**
   - What we know: In-memory pub/sub delivers messages in order within single publisher/subscriber
   - What's unclear: Whether to design for eventual consistency
   - Recommendation: For Phase 2, assume ordered delivery (single publisher, single subscriber, in-memory). Add event timestamp to support future reordering if needed.

3. **Retry behavior: Should Analytics Service retry failed database inserts?**
   - What we know: Event handler can return `(retry bool, err error)` to signal Dapr to retry
   - What's unclear: Appropriate retry policy for database failures
   - Recommendation: Don't retry on Phase 2. Log failure and return `(false, nil)` to acknowledge the event. This prevents infinite retry loops during database issues. Add retry policy in Phase 3 when introducing Dapr resiliency configuration.

## Sources

### Primary (HIGH confidence)
- Dapr Official Docs - [Programmatic subscriptions](https://docs.dapr.io/developing-applications/building-blocks/pubsub/subscription-methods/) - Implementation patterns
- Dapr Official Docs - [In-memory pub/sub component](https://docs.dapr.io/reference/components-reference/supported-pubsub/setup-inmemory/) - Component configuration
- Dapr Official Docs - [How to publish and subscribe](https://docs.dapr.io/developing-applications/building-blocks/pubsub/howto-publish-subscribe/) - Go SDK examples
- Dapr Official Docs - [CloudEvents format](https://docs.dapr.io/developing-applications/building-blocks/pubsub/pubsub-cloudevents/) - Message envelope structure
- Dapr Official Docs - [dapr run CLI reference](https://docs.dapr.io/reference/cli/dapr-run/) - Command-line flags
- Dapr Official Docs - [Multi-App Run template](https://docs.dapr.io/developing-applications/local-development/multi-app-dapr-run/multi-app-template/) - dapr.yaml format
- Go Official Docs - [golang-standards/project-layout](https://github.com/golang-standards/project-layout) - cmd/, internal/, pkg/ structure

### Secondary (MEDIUM confidence)
- VictoriaMetrics Blog - [Go Graceful Shutdown Patterns](https://victoriametrics.com/blog/go-graceful-shutdown/) - Context cancellation patterns (verified with multiple sources)
- Earthly Blog - [Building a Monorepo in Golang](https://earthly.dev/blog/golang-monorepo/) - Multi-binary project structure
- SQLite Forum - [Handling Timestamps in SQLite](https://blog.sqlite.ai/handling-timestamps-in-sqlite) - INTEGER vs TEXT storage (verified with official SQLite docs)
- SQLite Official - [WAL Mode Documentation](https://sqlite.org/wal.html) - Concurrency characteristics

### Tertiary (LOW confidence)
- Medium articles on Go clean architecture - Verified patterns against multiple sources, used for concept validation only
- GitHub examples of RFC 7807 Go implementations - Used to identify available libraries, not for specific recommendations

## Metadata

**Confidence breakdown:**
- Dapr pub/sub patterns: HIGH - All patterns verified against official documentation with code examples
- Go project structure: HIGH - Based on golang-standards/project-layout and verified with multiple current sources
- SQLite schema design: MEDIUM - Best practices derived from community sources and SQLite official docs, not specific to this use case
- Multi-service orchestration: HIGH - Dapr multi-app run is well-documented with official examples

**Research date:** 2026-02-15
**Valid until:** 2026-03-15 (30 days - Dapr and Go are stable; in-memory pub/sub component is unlikely to change significantly)

---

**Notes for Planner:**
- All locked decisions from CONTEXT.md are technically feasible with high confidence
- Dapr Go SDK provides idiomatic patterns for both publishing and subscribing
- In-memory pub/sub is appropriate for learning objectives despite message durability limitations
- Clean architecture patterns from Phase 1 map directly to Analytics Service
- Multi-app run simplifies development workflow significantly vs. manual dapr run commands
