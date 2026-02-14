# Phase 1: Foundation & URL Service Core - Context

**Gathered:** 2026-02-14
**Status:** Ready for planning

<domain>
## Phase Boundary

API service for shortening URLs and redirecting users. Users can submit a long URL via API and receive a short link with an auto-generated code. Visiting the short link redirects (302) to the original URL. Includes input validation, error handling, rate limiting, clean architecture layers, and SQLite storage. Analytics, link management, and production infrastructure are separate phases.

</domain>

<decisions>
## Implementation Decisions

### Short code design
- 8 characters long (alphanumeric: a-z, A-Z, 0-9 — 62 chars, case-sensitive)
- NanoID-style generation with custom alphabet
- Retry up to 5 times on collision, then fail with error
- ~218 trillion possible combinations — extremely collision-resistant

### API response shape
- Flat JSON for success responses (e.g., `{"short_code": "abc123XY", "original_url": "https://..."}`)
- RFC 7807 Problem Details for error responses (`{"type": "...", "title": "...", "status": 400, "detail": "..."}`)
- Versioned path prefix: `/api/v1/...` (e.g., `POST /api/v1/urls`, `GET /api/v1/urls/:code`)
- Redirect endpoint at root level: `GET /:code` (not versioned)

### Rate limiting strategy
- All API endpoints rate limited (both creation and redirects)
- Per IP address, 100 requests per minute
- Standard rate limit headers: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`
- Returns 429 Too Many Requests when exceeded (with RFC 7807 error body)

### URL validation rules
- Accept both HTTP and HTTPS schemes
- Allow localhost and private IPs (development convenience)
- Max URL length: 2048 characters
- Duplicate URLs return the same short code (deduplicate)

### Claude's Discretion
- Whether to return both short_code and full short_url in creation response, or just the code
- HTTP router/framework selection
- SQLite driver and migration approach
- Rate limiter implementation (in-memory token bucket, sliding window, etc.)
- Exact clean architecture directory layout
- Request logging and middleware chain

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

*Phase: 01-foundation-url-service-core*
*Context gathered: 2026-02-14*
