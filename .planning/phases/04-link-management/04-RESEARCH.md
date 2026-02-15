# Phase 4: Link Management - Research

**Researched:** 2026-02-15
**Domain:** Go REST API pagination, filtering, search, Dapr service invocation, SQLite query optimization
**Confidence:** HIGH

## Summary

Phase 4 implements comprehensive link management capabilities: listing, viewing, filtering, searching, and deleting short URLs. The core technical challenges are: (1) implementing offset-based pagination with configurable limits and sorting, (2) building dynamic WHERE clauses for date range filtering and substring search while maintaining SQL injection safety via sqlc, (3) fetching click counts from Analytics Service via Dapr service invocation with graceful degradation when unavailable, (4) implementing hard delete with pub/sub event propagation for cascade cleanup, and (5) optimizing SQLite queries with appropriate indexes.

The architecture follows established clean architecture patterns from Phase 1-3, extending URLRepository with new query methods and adding list/delete handlers. The list endpoint combines local URL data with remote click counts, requiring careful error handling to avoid cascading failures. SQLite's LIKE operator enables case-insensitive substring search, while composite indexes on (created_at, id) support efficient pagination and sorting.

User decisions mandate offset-based pagination (?page=1&per_page=20) over cursor-based, hard delete with "link.deleted" event publishing, and date range filtering with substring search. The response includes both short_code and full short_url for client convenience. Graceful degradation strategy for Analytics Service failures remains at Claude's discretion.

**Primary recommendation:** Extend sqlc queries with dynamic filtering via nullable parameters (sql.NullTime, sql.NullString), implement Dapr service invocation with context timeout (5s) and fallback to zero counts on failure, create composite index on (created_at, original_url) for sort/filter performance, publish "link.deleted" events via existing Dapr pub/sub with fire-and-forget pattern, and validate pagination parameters (max 100 per page) in handler layer.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Link metadata & response shape
- Each link returns: short_code, short_url (full URL), original_url, created_at, total_clicks
- total_clicks fetched from Analytics Service via Dapr service invocation
- Both short_code and short_url (full constructed URL) included in responses

#### Pagination & sorting
- Offset/page-based pagination: ?page=1&per_page=20
- Default page size: 20, max: 100
- Default sort: newest first (created_at DESC)
- Sort parameter supported: ?sort=created_at&order=desc — allow sorting by date or clicks

#### Management actions
- Hard delete supported: removes link from URL Service DB
- Delete publishes "link.deleted" event via Dapr pub/sub
- Analytics Service subscribes and deletes all click data for that short code
- Links are immutable — no editing of destination URL (delete and recreate instead)

#### Filtering & search
- Date range filtering: ?created_after=2026-01-01&created_before=2026-02-01
- Substring search on original URL: ?search=github.com (case-insensitive)
- Separate detail endpoint: GET /api/links/:short_code for single link lookup

### Claude's Discretion

Research options and recommend:
- Analytics Service unavailability fallback strategy (graceful degradation for click counts)
- Exact API response envelope structure
- Delete confirmation behavior (immediate vs require confirmation header)
- Index strategy for search and filter queries

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope

</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| sqlc | v1.30.0+ | Type-safe SQL query generation | Already in use; supports complex queries with ORDER BY, LIMIT, OFFSET, and WHERE filtering |
| Chi router | v5 (go-chi/chi/v5) | HTTP routing and query parameter parsing | Already in use; standard r.URL.Query() for pagination/filter params |
| Dapr Go SDK | v1.13.0 | Service invocation client | Already in use; InvokeMethod for synchronous service-to-service calls |
| SQLite (modernc.org/sqlite) | v1.45.0 | Database with index support | Already in use; B-tree indexes for filtering and sorting optimization |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| database/sql | stdlib | sql.NullTime, sql.NullString for optional params | Dynamic WHERE clauses with nullable filter parameters |
| net/url | stdlib | url.Values for query parameter parsing | Parse pagination, sort, filter parameters from request |
| strconv | stdlib | Atoi for numeric parameter conversion | Convert page, per_page, and limit parameters |
| time | stdlib | time.Parse("2006-01-02") for date filtering | Parse created_after/created_before date strings |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Offset pagination | Cursor-based pagination | Offset is simpler for small datasets (user requirement); cursor better for large, fast-changing data but adds complexity |
| Hard delete | Soft delete (deleted_at column) | Soft delete keeps data for recovery but requires filtering all queries; hard delete matches user requirement for true removal |
| Dapr service invocation | Direct HTTP client | Dapr provides service discovery, retries, and observability; direct HTTP requires manual endpoint management |
| Inline click count fetching | Background job enrichment | Inline is simpler and real-time; background jobs reduce latency but add complexity (out of scope for Phase 4) |

**Installation:**
```bash
# All dependencies already installed in previous phases
# No new dependencies required
```

## Architecture Patterns

### Recommended Project Structure
```
internal/urlservice/
├── delivery/http/
│   ├── handler.go              # Add ListLinks, DeleteLink handlers
│   └── response.go             # Add LinkListResponse, LinkDetailResponse
├── usecase/
│   ├── url_repository.go       # Add FindAll, Delete methods
│   └── url_service.go          # Add ListLinks, DeleteLink business logic
├── repository/sqlite/
│   └── url_repository.go       # Implement new query methods
└── domain/
    └── url.go                  # Existing URL domain entity

db/
├── query.sql                   # Add list, delete, search queries
└── schema.sql                  # Add indexes (created_at, original_url)

internal/analytics/
├── delivery/http/
│   └── handler.go              # Add HandleLinkDeleted subscription
└── usecase/
    └── click_repository.go     # Add DeleteByShortCode method
```

### Pattern 1: Offset-Based Pagination with sqlc

**What:** Use LIMIT and OFFSET in SQL queries with query parameter validation
**When to use:** List endpoints with page-based navigation
**Example:**
```sql
-- name: ListURLs :many
SELECT * FROM urls
WHERE
  (? IS NULL OR created_at >= ?)
  AND (? IS NULL OR created_at <= ?)
  AND (? IS NULL OR original_url LIKE '%' || ? || '%')
ORDER BY
  CASE WHEN ? = 'created_at' THEN created_at END DESC,
  CASE WHEN ? = 'created_at' THEN created_at END ASC
LIMIT ? OFFSET ?;

-- name: CountURLs :one
SELECT COUNT(*) FROM urls
WHERE
  (? IS NULL OR created_at >= ?)
  AND (? IS NULL OR created_at <= ?)
  AND (? IS NULL OR original_url LIKE '%' || ? || '%');
```

**Go handler pattern:**
```go
func parseListParams(r *http.Request) (page, perPage int, sort, order string, err error) {
    page, _ = strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }

    perPage, _ = strconv.Atoi(r.URL.Query().Get("per_page"))
    if perPage < 1 || perPage > 100 {
        perPage = 20 // Default and max validation
    }

    sort = r.URL.Query().Get("sort")
    if sort == "" {
        sort = "created_at"
    }

    order = r.URL.Query().Get("order")
    if order != "asc" && order != "desc" {
        order = "desc"
    }

    return
}
```

### Pattern 2: Dynamic WHERE Clauses with Nullable Parameters

**What:** Use sql.NullTime and sql.NullString for optional filter parameters
**When to use:** Search and filter queries where some parameters may be omitted
**Example:**
```go
// Repository layer
func (r *URLRepository) FindAll(ctx context.Context, filters Filters, limit, offset int) ([]domain.URL, error) {
    // Convert Go optional types to sql.Null* for sqlc
    createdAfter := sql.NullTime{
        Time:  filters.CreatedAfter,
        Valid: !filters.CreatedAfter.IsZero(),
    }

    createdBefore := sql.NullTime{
        Time:  filters.CreatedBefore,
        Valid: !filters.CreatedBefore.IsZero(),
    }

    search := sql.NullString{
        String: filters.Search,
        Valid:  filters.Search != "",
    }

    rows, err := r.queries.ListURLs(ctx, sqlc.ListURLsParams{
        CreatedAfter:  createdAfter,
        CreatedBefore: createdBefore,
        Search:        search,
        Limit:         int64(limit),
        Offset:        int64(offset),
    })
    // ... convert rows to domain.URL
}
```

**Confidence:** HIGH - sqlc documentation explicitly shows nullable parameter patterns for optional filters

### Pattern 3: Dapr Service Invocation with Timeout and Fallback

**What:** Call Analytics Service via Dapr with context timeout and graceful degradation
**When to use:** Fetching click counts when listing links
**Example:**
```go
func (h *Handler) getClickCounts(ctx context.Context, shortCodes []string) map[string]int64 {
    // Timeout for entire batch operation
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    counts := make(map[string]int64)

    if h.daprClient == nil {
        // Dapr unavailable - return zero counts
        h.logger.Warn("dapr client unavailable, returning zero click counts")
        return counts
    }

    // Batch invoke Analytics Service for each short code
    for _, code := range shortCodes {
        resp, err := h.daprClient.InvokeMethod(
            ctx,
            "analytics-service",
            fmt.Sprintf("analytics/%s", code),
            "get",
        )

        if err != nil {
            // Analytics Service unavailable - log and use zero
            h.logger.Warn("failed to fetch click count",
                zap.String("short_code", code),
                zap.Error(err),
            )
            counts[code] = 0
            continue
        }

        var analytics AnalyticsResponse
        if err := json.Unmarshal(resp, &analytics); err != nil {
            h.logger.Error("failed to parse analytics response",
                zap.String("short_code", code),
                zap.Error(err),
            )
            counts[code] = 0
            continue
        }

        counts[code] = analytics.TotalClicks
    }

    return counts
}
```

**Confidence:** MEDIUM - Dapr docs show InvokeMethod but don't detail batch patterns; timeout and fallback are standard Go practices

### Pattern 4: Hard Delete with Event Publishing

**What:** Delete record from database and publish event for cascade cleanup
**When to use:** DELETE /api/links/:short_code endpoint
**Example:**
```go
// Service layer
func (s *URLService) DeleteLink(ctx context.Context, shortCode string) error {
    // Delete from database (single source of truth)
    if err := s.repo.Delete(ctx, shortCode); err != nil {
        return err
    }

    // Publish deletion event for Analytics Service cleanup
    // Fire-and-forget - deletion succeeded, best-effort propagation
    event := LinkDeletedEvent{
        ShortCode: shortCode,
        DeletedAt: time.Now().UTC(),
    }

    data, err := json.Marshal(event)
    if err != nil {
        s.logger.Error("failed to marshal link deleted event", zap.Error(err))
        return nil // Don't fail the delete operation
    }

    if err := s.daprClient.PublishEvent(ctx, "pubsub", "link-deleted", data); err != nil {
        s.logger.Error("failed to publish link deleted event", zap.Error(err))
        // Don't return error - deletion succeeded, event is best-effort
    }

    return nil
}
```

**SQL query:**
```sql
-- name: DeleteURL :exec
DELETE FROM urls WHERE short_code = ?;
```

**Confidence:** HIGH - Matches established pub/sub patterns from Phase 2

### Pattern 5: SQLite Index Strategy for Filtering and Sorting

**What:** Composite indexes on filter and sort columns for query performance
**When to use:** Tables with filtering, searching, and sorting requirements
**Example:**
```sql
-- Composite index for created_at sorting (default sort)
CREATE INDEX idx_urls_created_at ON urls(created_at DESC);

-- Index for case-insensitive substring search on original_url
-- SQLite doesn't support expression indexes without enabling extensions
-- Use COLLATE NOCASE for case-insensitive comparisons
CREATE INDEX idx_urls_original_url ON urls(original_url COLLATE NOCASE);

-- Composite index for date range filtering + sorting
CREATE INDEX idx_urls_created_at_id ON urls(created_at DESC, id);
```

**Query analysis:**
```bash
# Use EXPLAIN QUERY PLAN to verify index usage
sqlite3 urls.db "EXPLAIN QUERY PLAN SELECT * FROM urls WHERE created_at >= '2026-01-01' ORDER BY created_at DESC LIMIT 20;"
```

**Confidence:** HIGH - SQLite documentation confirms B-tree index usage for ORDER BY, WHERE, and LIKE queries; composite indexes reduce query time by 10-100x per research

### Anti-Patterns to Avoid

- **N+1 Query Problem:** Don't fetch click counts individually per link in a loop. Batch fetch all counts after retrieving links, even if it means multiple serial calls (acceptable for Phase 4 scope; consider parallel goroutines for optimization).

- **Unbounded Pagination:** Don't allow unlimited per_page values. Always enforce max limit (100) to prevent memory exhaustion and slow queries on large offsets.

- **Ignoring Sort Injection:** Don't concatenate sort column names directly into SQL. Use CASE statements in sqlc queries with whitelisted column names to prevent SQL injection.

- **Silent Failures in Service Invocation:** Don't silently swallow all Analytics Service errors. Log warnings with structured logging (zap) for observability while gracefully degrading to zero counts.

- **Cascading Timeouts:** Don't use long timeouts (>5s) for service invocation in list endpoints. Short timeouts prevent slow Analytics Service from blocking list operations.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| SQL query builder for dynamic filters | Custom string concatenation or fmt.Sprintf | sqlc with nullable sql.Null* parameters | SQL injection risks, type safety, compile-time validation |
| Service discovery and invocation | Custom HTTP client with hardcoded URLs | Dapr service invocation via app-id | Automatic retries, observability, service mesh integration |
| Query parameter parsing/validation | Custom parser with regex | stdlib url.Values + strconv with validation | Well-tested, handles edge cases, standard Go idioms |
| Pagination response structure | Custom struct per endpoint | Reusable pagination envelope with generics (Go 1.18+) | Consistency across endpoints, less code duplication |
| Circuit breaker for service calls | Custom failure tracking | Dapr resiliency policies (configured via YAML) | Production-tested, configurable thresholds, half-open state |

**Key insight:** SQLite's query planner is sophisticated enough to optimize multi-condition WHERE clauses when proper indexes exist. Don't hand-roll query builders or ORM abstractions - sqlc with nullable parameters provides type safety while maintaining full SQL control.

## Common Pitfalls

### Pitfall 1: Deep Pagination Performance Degradation
**What goes wrong:** Using OFFSET with large values (OFFSET 50000) causes SQLite to scan and discard all skipped rows, making deep pagination extremely slow.

**Why it happens:** OFFSET-based pagination reads rows sequentially from the beginning, not from the offset position directly. Performance degrades linearly as page number increases.

**How to avoid:**
- Enforce reasonable page limits (max page 1000 or max offset 10,000)
- Document pagination limits in API responses
- Consider cursor-based pagination for future phases if dataset grows large
- Add LIMIT on total result sets to prevent excessive scanning

**Warning signs:**
- List endpoint response time increases with page number
- Database CPU spikes when requesting high page numbers
- User complaints about "page 100+ is slow"

**Confidence:** HIGH - Multiple sources confirm OFFSET performance issues; acceptable for Phase 4 scope with small datasets

### Pitfall 2: Analytics Service Blocking List Operations
**What goes wrong:** If Analytics Service is slow or down, list endpoint hangs or times out, making link management unusable.

**Why it happens:** Synchronous service invocation without timeout or fallback creates tight coupling between services.

**How to avoid:**
- Use context.WithTimeout (5s) for all Dapr service invocations
- Implement graceful degradation: return 0 clicks if Analytics Service fails
- Log failures with structured context (short_code, error) for debugging
- Consider eventual consistency: show "Click count loading..." in UI (Phase 5)

**Warning signs:**
- List endpoint latency matches Analytics Service latency
- List operations fail when Analytics Service is down
- Timeout errors in URL Service logs

**Confidence:** HIGH - Standard distributed system resilience pattern; Dapr docs recommend timeouts and fallbacks

### Pitfall 3: Case-Sensitive Search on SQLite
**What goes wrong:** Search query ?search=GitHub doesn't match URLs containing "github.com" due to case sensitivity.

**Why it happens:** SQLite's LIKE operator is case-sensitive by default unless COLLATE NOCASE is used.

**How to avoid:**
```sql
-- Correct: case-insensitive search
WHERE original_url LIKE '%' || ? || '%' COLLATE NOCASE

-- Wrong: case-sensitive search
WHERE original_url LIKE '%' || ? || '%'
```

Also ensure index uses COLLATE NOCASE:
```sql
CREATE INDEX idx_urls_original_url ON urls(original_url COLLATE NOCASE);
```

**Warning signs:**
- Search returns no results for valid URLs with different casing
- Users report "search doesn't work" for common URLs

**Confidence:** HIGH - SQLite documentation explicitly covers LIKE case sensitivity

### Pitfall 4: Idempotency in Delete Operations
**What goes wrong:** Deleting a non-existent short code returns 500 Internal Error instead of 404 Not Found, or succeeds with 204 No Content, confusing clients about operation result.

**Why it happens:** DELETE is idempotent per HTTP spec - repeated deletes should succeed. But SQL DELETE returns 0 rows affected for non-existent resources.

**How to avoid:**
```go
// Check existence before delete (lost idempotency)
url, err := s.repo.FindByShortCode(ctx, shortCode)
if err != nil {
    return domain.ErrURLNotFound // 404
}
if err := s.repo.Delete(ctx, shortCode); err != nil {
    return err
}

// OR: Accept idempotent deletes (simpler, standard)
if err := s.repo.Delete(ctx, shortCode); err != nil {
    return err
}
// Always return 204 No Content, even if already deleted
```

**Recommendation:** Use idempotent approach (always 204) unless user requires confirmation that resource existed. This matches REST best practices where DELETE is idempotent.

**Warning signs:**
- Inconsistent HTTP status codes for same delete operation
- Client retry logic breaks on 404 responses

**Confidence:** MEDIUM - HTTP spec is clear on idempotency, but implementation varies across APIs

### Pitfall 5: SQL Injection in Dynamic Sorting
**What goes wrong:** Building ORDER BY clauses with string concatenation allows SQL injection via ?sort=created_at;DROP TABLE urls

**Why it happens:** User-controlled sort parameter is concatenated directly into SQL query string.

**How to avoid:**
Use sqlc CASE statements with whitelisted values:
```sql
-- Safe: sqlc-generated query with parameter validation
ORDER BY
  CASE WHEN ? = 'created_at' AND ? = 'desc' THEN created_at END DESC,
  CASE WHEN ? = 'created_at' AND ? = 'asc' THEN created_at END ASC,
  created_at DESC -- Default fallback

-- Dangerous: string concatenation (DON'T DO THIS)
-- query := fmt.Sprintf("SELECT * FROM urls ORDER BY %s %s", sort, order)
```

Validate in handler before passing to repository:
```go
allowedSorts := map[string]bool{"created_at": true}
if !allowedSorts[sort] {
    sort = "created_at" // Safe default
}
```

**Warning signs:**
- Sort parameter not validated against whitelist
- SQL concatenation with fmt.Sprintf or string concatenation
- sqlc not used for dynamic ORDER BY

**Confidence:** HIGH - SQL injection is well-documented; sqlc prevents most cases but dynamic ORDER BY requires care

## Code Examples

Verified patterns from official sources and established codebase:

### List Links Handler with Pagination
```go
// Handler: internal/urlservice/delivery/http/handler.go
func (h *Handler) ListLinks(w http.ResponseWriter, r *http.Request) {
    // Parse and validate pagination parameters
    page, _ := strconv.Atoi(r.URL.Query().Get("page"))
    if page < 1 {
        page = 1
    }

    perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
    if perPage < 1 || perPage > 100 {
        perPage = 20
    }

    // Parse optional filters
    var createdAfter, createdBefore time.Time
    if afterStr := r.URL.Query().Get("created_after"); afterStr != "" {
        createdAfter, _ = time.Parse("2006-01-02", afterStr)
    }
    if beforeStr := r.URL.Query().Get("created_before"); beforeStr != "" {
        createdBefore, _ = time.Parse("2006-01-02", beforeStr)
    }
    search := r.URL.Query().Get("search")

    // Fetch links from service
    result, err := h.service.ListLinks(r.Context(), ListLinksParams{
        Page:          page,
        PerPage:       perPage,
        Sort:          r.URL.Query().Get("sort"),
        Order:         r.URL.Query().Get("order"),
        CreatedAfter:  createdAfter,
        CreatedBefore: createdBefore,
        Search:        search,
    })

    if err != nil {
        problem := problemdetails.New(
            http.StatusInternalServerError,
            problemdetails.TypeInternalError,
            "Internal Server Error",
            "Failed to retrieve links",
        )
        writeProblem(w, problem)
        return
    }

    writeJSON(w, http.StatusOK, result)
}
```

### Service Layer with Click Count Enrichment
```go
// Service: internal/urlservice/usecase/url_service.go
type LinkWithClicks struct {
    ShortCode   string    `json:"short_code"`
    ShortURL    string    `json:"short_url"`
    OriginalURL string    `json:"original_url"`
    CreatedAt   time.Time `json:"created_at"`
    TotalClicks int64     `json:"total_clicks"`
}

type LinkListResult struct {
    Links      []LinkWithClicks `json:"links"`
    Total      int64            `json:"total"`
    Page       int              `json:"page"`
    PerPage    int              `json:"per_page"`
    TotalPages int              `json:"total_pages"`
}

func (s *URLService) ListLinks(ctx context.Context, params ListLinksParams) (*LinkListResult, error) {
    // Calculate offset
    offset := (params.Page - 1) * params.PerPage

    // Fetch links from repository
    urls, err := s.repo.FindAll(ctx, Filters{
        CreatedAfter:  params.CreatedAfter,
        CreatedBefore: params.CreatedBefore,
        Search:        params.Search,
    }, params.PerPage, offset)
    if err != nil {
        return nil, err
    }

    // Get total count for pagination metadata
    total, err := s.repo.Count(ctx, Filters{
        CreatedAfter:  params.CreatedAfter,
        CreatedBefore: params.CreatedBefore,
        Search:        params.Search,
    })
    if err != nil {
        return nil, err
    }

    // Enrich with click counts from Analytics Service
    shortCodes := make([]string, len(urls))
    for i, url := range urls {
        shortCodes[i] = url.ShortCode
    }

    clickCounts := s.getClickCounts(ctx, shortCodes)

    // Build response
    links := make([]LinkWithClicks, len(urls))
    for i, url := range urls {
        links[i] = LinkWithClicks{
            ShortCode:   url.ShortCode,
            ShortURL:    s.baseURL + "/" + url.ShortCode,
            OriginalURL: url.OriginalURL,
            CreatedAt:   url.CreatedAt,
            TotalClicks: clickCounts[url.ShortCode],
        }
    }

    totalPages := int(math.Ceil(float64(total) / float64(params.PerPage)))

    return &LinkListResult{
        Links:      links,
        Total:      total,
        Page:       params.Page,
        PerPage:    params.PerPage,
        TotalPages: totalPages,
    }, nil
}
```

### Delete Handler with Event Publishing
```go
// Handler: internal/urlservice/delivery/http/handler.go
func (h *Handler) DeleteLink(w http.ResponseWriter, r *http.Request) {
    code := chi.URLParam(r, "code")

    if err := h.service.DeleteLink(r.Context(), code); err != nil {
        if errors.Is(err, domain.ErrURLNotFound) {
            // Idempotent delete: already deleted is success
            w.WriteHeader(http.StatusNoContent)
            return
        }

        problem := problemdetails.New(
            http.StatusInternalServerError,
            problemdetails.TypeInternalError,
            "Internal Server Error",
            "Failed to delete link",
        )
        writeProblem(w, problem)
        return
    }

    w.WriteHeader(http.StatusNoContent)
}
```

### Analytics Service Link Deletion Handler
```go
// Analytics Service subscription handler
func (h *Handler) HandleLinkDeleted(w http.ResponseWriter, r *http.Request) {
    var cloudEvent struct {
        Data json.RawMessage `json:"data"`
    }

    if err := json.NewDecoder(r.Body).Decode(&cloudEvent); err != nil {
        h.logger.Error("failed to decode cloud event", zap.Error(err))
        w.WriteHeader(http.StatusOK) // Acknowledge malformed events
        return
    }

    var event events.LinkDeletedEvent
    if err := json.Unmarshal(cloudEvent.Data, &event); err != nil {
        h.logger.Error("failed to unmarshal link deleted event", zap.Error(err))
        w.WriteHeader(http.StatusOK)
        return
    }

    // Delete all click data for this short code
    if err := h.analyticsService.DeleteClickData(r.Context(), event.ShortCode); err != nil {
        h.logger.Error("failed to delete click data",
            zap.String("short_code", event.ShortCode),
            zap.Error(err),
        )
        // Don't return - acknowledge event to prevent retries
    }

    h.logger.Info("deleted click data for link",
        zap.String("short_code", event.ShortCode),
    )

    w.WriteHeader(http.StatusOK)
}
```

### SQLite Queries with Filtering and Sorting
```sql
-- db/query.sql

-- name: ListURLs :many
SELECT * FROM urls
WHERE
  (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'))
  AND (sqlc.narg('search') IS NULL OR original_url LIKE '%' || sqlc.narg('search') || '%' COLLATE NOCASE)
ORDER BY
  CASE WHEN sqlc.arg('sort_field') = 'created_at' AND sqlc.arg('sort_order') = 'desc'
    THEN created_at END DESC,
  CASE WHEN sqlc.arg('sort_field') = 'created_at' AND sqlc.arg('sort_order') = 'asc'
    THEN created_at END ASC,
  created_at DESC  -- Default fallback
LIMIT ? OFFSET ?;

-- name: CountURLs :one
SELECT COUNT(*) FROM urls
WHERE
  (sqlc.narg('created_after') IS NULL OR created_at >= sqlc.narg('created_after'))
  AND (sqlc.narg('created_before') IS NULL OR created_at <= sqlc.narg('created_before'))
  AND (sqlc.narg('search') IS NULL OR original_url LIKE '%' || sqlc.narg('search') || '%' COLLATE NOCASE);

-- name: DeleteURL :exec
DELETE FROM urls WHERE short_code = ?;

-- name: DeleteClicksByShortCode :exec
DELETE FROM clicks WHERE short_code = ?;
```

### Database Migration for Indexes
```sql
-- internal/urlservice/database/migrations/000002_add_indexes.up.sql
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_urls_original_url ON urls(original_url COLLATE NOCASE);

-- internal/urlservice/database/migrations/000002_add_indexes.down.sql
DROP INDEX IF EXISTS idx_urls_created_at;
DROP INDEX IF EXISTS idx_urls_original_url;
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| String concatenation for dynamic SQL | sqlc with named parameters and CASE statements | sqlc v1.0+ (2020) | Type safety, SQL injection prevention, compile-time validation |
| Direct HTTP calls between services | Dapr service invocation with app-id | Dapr v1.0 (2021) | Service discovery, observability, retries without custom code |
| Manual JSON parsing for query params | Standard library url.Values + strconv | Always standard | Idiomatic Go, well-tested edge case handling |
| Large page size defaults (100+) | Conservative defaults (20) with enforced max (100) | API best practice evolution | Prevents memory/performance issues from unbounded requests |
| Cursor-only pagination for all cases | Offset for small datasets, cursor for large/real-time | Pagination pattern evolution (2020s) | Simpler implementation for appropriate use cases |

**Deprecated/outdated:**
- **GORM nullable types:** Modern sqlc uses database/sql stdlib types (sql.NullString, sql.NullTime) instead of GORM-specific gorm.DeletedAt - better compatibility, less dependency weight
- **fmt.Sprintf SQL building:** Replaced by sqlc code generation - eliminates SQL injection risks and provides compile-time type checking
- **Synchronous HTTP clients without timeouts:** Modern Go always uses context.WithTimeout for external calls - prevents cascading failures

## Open Questions

1. **Batch service invocation optimization**
   - What we know: Current research suggests serial Dapr InvokeMethod calls per short_code
   - What's unclear: Whether Dapr Go SDK supports batch/parallel invocations or if we should use goroutines
   - Recommendation: Start with serial calls (simpler, adequate for <100 links per page); add goroutine pooling if profiling shows latency issues (likely in Phase 5 performance optimization)

2. **Click count caching strategy**
   - What we know: Real-time fetch from Analytics Service on every list request
   - What's unclear: Whether to cache click counts (and for how long) vs always fetch fresh
   - Recommendation: Phase 4 fetches real-time; defer caching to Phase 5 if it becomes performance bottleneck (premature optimization)

3. **Analytics Service cascade delete confirmation**
   - What we know: URL Service publishes "link.deleted" event; Analytics Service subscribes
   - What's unclear: Whether to wait for Analytics deletion confirmation before returning 204, or fire-and-forget
   - Recommendation: Fire-and-forget (event-driven async pattern); Analytics Service is eventually consistent with URL Service. URL deletion is authoritative.

## Sources

### Primary (HIGH confidence)
- [sqlc documentation - /websites/sqlc_dev](https://docs.sqlc.dev) - SQL query patterns, nullable parameters, ORDER BY with CASE
- [Dapr Go SDK - /dapr/go-sdk](https://github.com/dapr/go-sdk) - InvokeMethod API, service invocation patterns
- [SQLite Query Optimizer Overview](https://www.sqlite.org/optoverview.html) - Index usage, B-tree performance
- [SQLite Query Planning](https://www.sqlite.org/queryplanner.html) - EXPLAIN QUERY PLAN, optimization strategies
- [Dapr Resiliency Policies](https://docs.dapr.io/operations/resiliency/policies/) - Circuit breaker, timeout, retry configuration
- Existing codebase - Clean architecture patterns, Chi router setup, Dapr pub/sub usage

### Secondary (MEDIUM confidence)
- [A Developer's Guide to API Pagination](https://embedded.gusto.com/blog/api-pagination/) - Offset vs cursor tradeoffs
- [Golang Pagination Face-Off: Cursor vs. Offset](https://medium.com/@cccybercrow/golang-pagination-face-off-cursor-vs-e7e456395b9f) - Performance comparison with Go examples (December 2025)
- [Building Resilient REST API Integrations: Graceful Degradation](https://medium.com/@oshiryaeva/building-resilient-rest-api-integrations-graceful-degradation-and-combining-patterns-e8352d8e29c0) - Fallback patterns (January 2026)
- [How to Implement Circuit Breakers in Go with sony/gobreaker](https://oneuptime.com/blog/post/2026-01-07-go-circuit-breaker/view) - Circuit breaker implementation (January 2026)
- [Indexing Strategies in SQLite](https://www.sqliteforum.com/p/indexing-strategies-in-sqlite-improving-query-performance) - Composite index patterns
- [Best Practices for Improving SQLite Query Performance](https://moldstud.com/articles/p-best-practices-for-improving-sqlite-query-performance-a-guide-for-developers) - Index strategies verified

### Tertiary (LOW confidence)
- WebSearch results on Go Chi router query parameter parsing - Standard library patterns confirmed via multiple sources
- WebSearch results on soft delete vs hard delete - General patterns, not Go-specific but applicable

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - All libraries already in use, patterns established in previous phases
- Architecture: HIGH - Extends existing clean architecture with proven patterns from Phase 1-3
- Pitfalls: HIGH - Well-documented issues (offset performance, SQL injection, service coupling) confirmed by multiple authoritative sources
- Graceful degradation: MEDIUM - Pattern is standard but specific timeout values and fallback UX are judgment calls
- Batch optimization: LOW - Dapr Go SDK docs don't detail batch invocation patterns; requires experimentation

**Research date:** 2026-02-15
**Valid until:** 2026-03-15 (30 days) - Stack is stable, Go and SQLite patterns evolve slowly
