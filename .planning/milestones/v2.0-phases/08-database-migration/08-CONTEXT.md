# Phase 8: Database Migration - Context

**Gathered:** 2026-02-16
**Status:** Ready for planning

<domain>
## Phase Boundary

Migrate both services from stub/mock data to real PostgreSQL persistence. URL API and Analytics RPC get working data layers with goctl-generated models. No new API endpoints or capabilities — this phase wires existing stubs to a real database.

</domain>

<decisions>
## Implementation Decisions

### Schema design
- Hard delete — DELETE endpoint removes the row, no soft delete/status column
- No expiration — short links live forever (or until manually deleted)
- UUIDv7 primary keys on both tables — time-ordered, natural cursor pagination, no sequence coordination
- Minimal indexes — unique constraint on short_code and primary keys only. Add more when performance demands.

### Cache strategy
- No Redis cache for now — direct PostgreSQL queries
- Keep the architecture simple with fewer moving parts
- Cache can be added later via goctl model regeneration if needed

### Local dev setup
- Follow go-zero recommended approach for running PostgreSQL locally
- Migration tooling: Claude's discretion

### Query patterns
- Cursor pagination based on UUIDv7 id (time-ordered, single field cursor)
- URL search via ILIKE prefix match on original_url — simple, no full-text search setup
- Denormalized click_count column on urls table for fast count reads
- Analytics RPC additional methods beyond GetClickCount: Claude's discretion

### Claude's Discretion
- Clicks table: whether to use TIMESTAMPTZ for clicked_at and whether to add enrichment columns (ip_address, user_agent, referer) now vs Phase 9
- Migration tooling choice (manual DDL, golang-migrate, or other)
- Local dev PostgreSQL setup (Docker Compose vs other)
- Whether to add analytics breakdown queries (GetClicksByCountry, GetClicksOverTime) in this phase or defer
- Index additions beyond the minimum if query patterns clearly require them

</decisions>

<specifics>
## Specific Ideas

- UUIDv7 was explicitly chosen for primary keys — provides timestamp ordering in the ID itself, eliminating need for separate created_at-based cursor logic
- Denormalized click_count on urls table — the user wants fast reads over perfect consistency for click counts
- ILIKE search is acceptable for current scale — no need to over-engineer with full-text search

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 08-database-migration*
*Context gathered: 2026-02-16*
