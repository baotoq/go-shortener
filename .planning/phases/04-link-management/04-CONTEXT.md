# Phase 4: Link Management - Context

**Gathered:** 2026-02-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can list, view, and delete their created short links via API. The list endpoint returns link metadata with click counts (fetched from Analytics Service). Links are immutable once created — no editing of destination URL. Pagination, sorting, filtering, and search are supported.

</domain>

<decisions>
## Implementation Decisions

### Link metadata & response shape
- Each link returns: short_code, short_url (full URL), original_url, created_at, total_clicks
- total_clicks fetched from Analytics Service via Dapr service invocation
- Both short_code and short_url (full constructed URL) included in responses

### Pagination & sorting
- Offset/page-based pagination: ?page=1&per_page=20
- Default page size: 20, max: 100
- Default sort: newest first (created_at DESC)
- Sort parameter supported: ?sort=created_at&order=desc — allow sorting by date or clicks

### Management actions
- Hard delete supported: removes link from URL Service DB
- Delete publishes "link.deleted" event via Dapr pub/sub
- Analytics Service subscribes and deletes all click data for that short code
- Links are immutable — no editing of destination URL (delete and recreate instead)

### Filtering & search
- Date range filtering: ?created_after=2026-01-01&created_before=2026-02-01
- Substring search on original URL: ?search=github.com (case-insensitive)
- Separate detail endpoint: GET /api/links/:short_code for single link lookup

### Claude's Discretion
- Analytics Service unavailability fallback strategy (graceful degradation for click counts)
- Exact API response envelope structure
- Delete confirmation behavior (immediate vs require confirmation header)
- Index strategy for search and filter queries

</decisions>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 04-link-management*
*Context gathered: 2026-02-15*
