# Phase 3: Enhanced Analytics - Context

**Gathered:** 2026-02-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Enrich click tracking with geo-location, device type, and traffic source data. Analytics API returns enriched breakdowns and individual click details. All enrichment happens in the Analytics Service — the redirect path stays fast. Creating new analytics capabilities beyond enrichment belongs in other phases.

</domain>

<decisions>
## Implementation Decisions

### Analytics response shape
- Both summary and detail endpoints: summary for aggregated breakdowns, detail for individual click records
- Summary breakdowns show counts + percentages (e.g., "US: 42 clicks (58%)")
- Support custom time-range filtering with arbitrary start/end dates (?from=2026-01-01&to=2026-02-01)
- Individual click details endpoint uses cursor-based pagination

### Geo-location depth
- Country-level only — no city or region
- Use ISO country codes (e.g., "US", "DE", "JP") for storage and API responses
- Unresolvable IPs (private, VPN, unknown) grouped under single "Unknown" label

### Device categorization
- High-level categories only: Desktop, Mobile, Tablet, Bot
- Bots/crawlers tracked separately — counted but not mixed into human traffic
- Unrecognized User-Agent strings categorized as "Unknown"
- Summary shows all device categories (only 4 + Unknown, so no top-N needed)

### Traffic source classification
- Referers classified into categorized groups (not raw URLs)
- Category set and classification rules at Claude's discretion
- Missing/empty referer handling at Claude's discretion

### Claude's Discretion
- IP storage alongside resolved country (privacy vs re-processing trade-off)
- Traffic source category set (Social, Search, Direct, Other or similar)
- How to handle missing referer (Direct vs Unknown convention)
- Whether to store raw referer URL alongside classified category
- Enrichment timing (on ingest vs on query)
- Geo-IP library/database selection
- User-Agent parsing library selection

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

*Phase: 03-enhanced-analytics*
*Context gathered: 2026-02-15*
