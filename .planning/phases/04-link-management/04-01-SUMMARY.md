---
phase: 04-link-management
plan: 01
subsystem: url-service
tags: [api, link-management, pagination, dapr]
status: complete
completed_at: 2026-02-15

dependency_graph:
  requires:
    - 01-01-SUMMARY.md  # URL Service database schema and repository patterns
    - 01-02-SUMMARY.md  # URLService patterns and validation
    - 01-03-SUMMARY.md  # HTTP handler patterns and RFC 7807 error handling
    - 02-02-SUMMARY.md  # Dapr client integration patterns
    - 02-03-SUMMARY.md  # Analytics Service endpoint structure for click counts
  provides:
    - GET /api/v1/links (paginated list with filters)
    - GET /api/v1/links/:code (single link detail)
    - DELETE /api/v1/links/:code (hard delete with event)
    - LinkDeletedEvent for analytics cleanup
    - Click count enrichment via service invocation
  affects:
    - internal/urlservice/usecase/url_service.go (extended with list/detail/delete methods)
    - internal/urlservice/delivery/http/handler.go (new endpoints)
    - db/query.sql (ListURLs, ListURLsAsc, CountURLs, DeleteURL queries)

tech_stack:
  added:
    - SQLite indexes: idx_urls_created_at, idx_urls_original_url
    - Migration: 000002_add_list_indexes
  patterns:
    - Separate sqlc queries for ASC/DESC sort order (sqlc limitation workaround)
    - Nullable parameter handling: interface{} for optional filters
    - Click count enrichment with 5s timeout and graceful degradation
    - Fire-and-forget event publishing in DeleteLink
    - Idempotent delete (always returns 204)
    - End-of-day adjustment for created_before filter (23:59:59)

key_files:
  created:
    - internal/urlservice/database/migrations/000002_add_list_indexes.up.sql
    - internal/urlservice/database/migrations/000002_add_list_indexes.down.sql
    - internal/shared/events/link_deleted_event.go
  modified:
    - db/schema.sql (added indexes)
    - db/query.sql (added ListURLs, ListURLsAsc, CountURLs, DeleteURL)
    - internal/urlservice/usecase/url_repository.go (extended interface)
    - internal/urlservice/repository/sqlite/url_repository.go (implemented FindAll, Count, Delete)
    - internal/urlservice/usecase/url_service.go (added ListLinks, GetLinkDetail, DeleteLink)
    - internal/urlservice/delivery/http/handler.go (added handlers)
    - internal/urlservice/delivery/http/router.go (added routes)
    - cmd/url-service/main.go (wired daprClient, logger, baseURL)

decisions:
  - type: technical
    decision: "Split ListURLs into two queries (ListURLs DESC, ListURLsAsc) instead of dynamic ORDER BY"
    rationale: "sqlc doesn't support dynamic ORDER BY direction; this avoids SQL injection risk while maintaining type safety"
  - type: technical
    decision: "5-second timeout for Analytics Service invocation with fallback to 0 clicks"
    rationale: "Graceful degradation ensures link listing always succeeds even if Analytics is unavailable"
  - type: technical
    decision: "Fire-and-forget link.deleted event publishing in goroutine"
    rationale: "Deletion already succeeded in database; event failure shouldn't block user response"
  - type: technical
    decision: "Idempotent delete always returns 204 even if link doesn't exist"
    rationale: "REST idempotency principle - repeated DELETE has same effect as single DELETE"
  - type: technical
    decision: "End-of-day adjustment for created_before filter (add 23:59:59)"
    rationale: "User-friendly: ?created_before=2026-02-15 includes entire day, not just midnight"

metrics:
  duration: 6m 2s
  tasks_completed: 2
  commits: 2
  files_created: 5
  files_modified: 8
---

# Phase 04 Plan 01: Link Management API Summary

**One-liner:** Paginated link listing with search/date filters and click count enrichment via Dapr service invocation, plus detail/delete endpoints with Analytics cleanup events.

## Objective Achieved

Added three link management endpoints to URL Service:
- **GET /api/v1/links** - Paginated list with filters (search, date range, sort order) and click counts
- **GET /api/v1/links/:code** - Single link detail with click count
- **DELETE /api/v1/links/:code** - Idempotent hard delete with link.deleted event

Click counts are fetched from Analytics Service via Dapr service invocation with 5-second timeout and graceful degradation to 0 when unavailable.

## Implementation Details

### Task 1: Database Indexes and Repository Layer

**Created migration 000002:**
```sql
CREATE INDEX IF NOT EXISTS idx_urls_created_at ON urls(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_urls_original_url ON urls(original_url COLLATE NOCASE);
```

**Added sqlc queries:**
- `ListURLs` / `ListURLsAsc` - Paginated list with optional filters (created_after, created_before, search)
- `CountURLs` - Total count with same filters for pagination metadata
- `DeleteURL` - Hard delete by short_code

**Extended URLRepository interface:**
```go
FindAll(ctx context.Context, params FindAllParams) ([]domain.URL, error)
Count(ctx context.Context, params CountParams) (int64, error)
Delete(ctx context.Context, shortCode string) error
```

**Implementation notes:**
- Used `interface{}` for nullable sqlc parameters (nil = no filter)
- Time.Time converted to RFC3339 strings for SQLite compatibility
- Split queries for ASC/DESC to work around sqlc dynamic ORDER BY limitation
- Delete is idempotent (no error if short_code doesn't exist)

**Commit:** `f36daaa`

### Task 2: Service Layer and HTTP Handlers

**Service layer additions:**
```go
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
```

**Click count enrichment:**
- Calls `daprClient.InvokeMethod(ctx, "analytics-service", "analytics/"+code, "get")` for each short code
- 5-second timeout context
- On error (timeout, parse, unavailable): log warning, use 0
- Returns `map[string]int64` for batch enrichment

**Event publishing:**
```go
type LinkDeletedEvent struct {
    ShortCode string    `json:"short_code"`
    DeletedAt time.Time `json:"deleted_at"`
}
```
- Published to topic "link-deleted" via Dapr pub/sub
- Fire-and-forget in goroutine after delete succeeds
- Failure logged but doesn't affect delete response

**HTTP handlers:**
- `ListLinks` - Parses query params (page, per_page, search, created_after, created_before, order), validates, calls service
- `GetLinkDetail` - Returns single link or 404
- `DeleteLink` - Returns 204 always (idempotent)

**Validation:**
- page >= 1
- 1 <= per_page <= 100 (default 20)
- sort whitelist: ["created_at"]
- order whitelist: ["asc", "desc"]
- created_before gets end-of-day adjustment (+23:59:59)

**Commit:** `4b14290`

## Verification Results

Manual testing confirmed:
1. GET /api/v1/links returns paginated JSON with links[], total, page, per_page, total_pages
2. Pagination works: ?page=1&per_page=2 returns max 2 links with total_pages=2
3. Search works case-insensitive: ?search=example matches "example.com" and "Example.com"
4. Search works: ?search=github filters correctly
5. GET /api/v1/links/:code returns single link with total_clicks
6. DELETE /api/v1/links/:code returns 204
7. Link removed from list after delete (count decreases)
8. Repeated DELETE returns 204 (idempotent)
9. Order works: ?order=asc shows oldest first, ?order=desc shows newest first
10. Click counts show 0 when Analytics Service unavailable (graceful degradation)
11. No errors in logs (warnings expected for Dapr unavailability)

## Deviations from Plan

None - plan executed exactly as written.

### Auto-fixed Issues

No blocking issues encountered. Implementation followed plan specification completely.

### Authentication Gates

None.

## Key Learnings

1. **sqlc dynamic ORDER BY limitation:** sqlc doesn't support parameterized ORDER BY direction. Solution: create separate queries (ListURLs DESC, ListURLsAsc ASC) and choose based on params.SortOrder.

2. **Nullable parameter pattern:** Use `interface{}` for sqlc.narg() parameters. Pass `nil` for no filter, actual value for filter. Cleaner than sql.NullString in repository layer.

3. **End-of-day filter UX:** User expectation for ?created_before=2026-02-15 is "include entire day", not "midnight only". Add 23:59:59 in handler.

4. **Graceful degradation pattern:** Analytics Service unavailability shouldn't break link listing. 5s timeout + zero fallback = good UX.

5. **Fire-and-forget events:** Delete already succeeded when event publishes. Event failure = log warning, not user error.

## Self-Check: PASSED

Created files exist:
```bash
[ -f "internal/urlservice/database/migrations/000002_add_list_indexes.up.sql" ] && echo "FOUND"
[ -f "internal/urlservice/database/migrations/000002_add_list_indexes.down.sql" ] && echo "FOUND"
[ -f "internal/shared/events/link_deleted_event.go" ] && echo "FOUND"
```

Commits exist:
```bash
git log --oneline --all | grep -q "f36daaa" && echo "FOUND: f36daaa"
git log --oneline --all | grep -q "4b14290" && echo "FOUND: 4b14290"
```

Modified files have expected content:
- db/query.sql contains ListURLs, ListURLsAsc, CountURLs, DeleteURL
- internal/urlservice/usecase/url_service.go contains ListLinks, GetLinkDetail, DeleteLink
- internal/urlservice/delivery/http/handler.go contains handlers
- internal/urlservice/delivery/http/router.go contains /api/v1/links routes

All files verified present and containing expected patterns.

## Next Steps

Plan 04-02 will add:
- Custom alias support for short codes
- Expiration/TTL for temporary links
- Link editing (URL replacement)
- Bulk operations

## Self-Check Results

All verification checks passed:

**Files created:**
- ✓ internal/urlservice/database/migrations/000002_add_list_indexes.up.sql
- ✓ internal/urlservice/database/migrations/000002_add_list_indexes.down.sql
- ✓ internal/shared/events/link_deleted_event.go

**Commits:**
- ✓ f36daaa - Task 1: Database indexes and repository layer
- ✓ 4b14290 - Task 2: Service layer and HTTP handlers

**Content verification:**
- ✓ db/query.sql contains ListURLs, ListURLsAsc, CountURLs, DeleteURL
- ✓ internal/urlservice/usecase/url_service.go contains ListLinks, GetLinkDetail, DeleteLink
- ✓ internal/urlservice/delivery/http/handler.go contains new handlers
- ✓ internal/urlservice/delivery/http/router.go contains /api/v1/links routes

**Self-Check: PASSED**
