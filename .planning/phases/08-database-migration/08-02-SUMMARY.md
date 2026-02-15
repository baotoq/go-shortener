---
phase: 08-database-migration
plan: 02
subsystem: url-api-wiring
tags: [url-api, postgresql, nanoid, uuidv7, problem-details]
dependencies:
  requires:
    - 08-01 (PostgreSQL infrastructure + models)
  provides:
    - URL API fully backed by PostgreSQL
    - All 5 endpoints using real database operations
    - RFC 7807 Problem Details JSON error responses
  affects:
    - services/url-api/ (config, svc, logic, handler)
    - pkg/problemdetails/
tech_stack:
  added:
    - github.com/google/uuid (UUIDv7)
    - github.com/matoous/go-nanoid/v2 (NanoID short codes)
  patterns:
    - UUIDv7 primary keys for time-ordered inserts
    - NanoID 8-char with collision retry (max 5 attempts)
    - Fire-and-forget click count increment via goroutine
    - ProblemDetail implements error interface with Body() for JSON serialization
key_files:
  modified:
    - pkg/problemdetails/problemdetails.go
    - services/url-api/url.go
    - services/url-api/etc/url.yaml
    - services/url-api/internal/config/config.go
    - services/url-api/internal/svc/servicecontext.go
    - services/url-api/internal/handler/redirect/redirecthandler.go
    - services/url-api/internal/logic/shorten/shortenlogic.go
    - services/url-api/internal/logic/redirect/redirectlogic.go
    - services/url-api/internal/logic/links/listlinkslogic.go
    - services/url-api/internal/logic/links/getlinkdetaillogic.go
    - services/url-api/internal/logic/links/deletelinklogic.go
    - go.mod
    - go.sum
decisions:
  - ProblemDetail.Error() method added for error interface compliance
  - ProblemDetail.Body() method returns non-error struct to avoid go-zero plaintext serialization
  - Error handler uses errors.As to detect ProblemDetail and return correct status + JSON body
  - Hard delete for DeleteLink (row removed, not soft delete)
  - Fire-and-forget goroutine for click count increment on redirect
metrics:
  tasks_completed: 2/2
  commits: 1
  files_modified: 14
---

# Phase 8 Plan 2: URL API Service Wiring

**One-liner:** Wire all 5 URL API endpoints to PostgreSQL with UUIDv7 keys, NanoID short codes, and RFC 7807 Problem Details error responses.

## What Was Built

1. **ServiceContext + Config**: Added `DataSource` config field and initialized `UrlModel` with PostgreSQL connection via `sqlx.NewSqlConn("postgres", c.DataSource)`.

2. **Shorten Logic**: Generate UUIDv7 primary key + NanoID 8-char short code from 62-char alphabet. Collision retry loop (max 5 attempts) detects PostgreSQL unique constraint violations and regenerates both ID and code.

3. **Redirect Logic**: Changed return signature to `(string, error)` to return original URL. Handler performs HTTP 302 redirect. Click count incremented atomically via fire-and-forget goroutine using `context.Background()`.

4. **ListLinks Logic**: Calls `UrlModel.ListWithPagination` with page/perPage/search/sort/order. Maps DB results to response types. Calculates totalPages from totalCount.

5. **GetLinkDetail Logic**: Looks up URL by short code via `FindOneByShortCode`. Returns 404 Problem Details if not found. Maps to response with `total_clicks` from denormalized `click_count`.

6. **DeleteLink Logic**: Looks up URL by short code to get UUID, then hard-deletes by ID. Returns 404 if short code not found.

7. **Problem Details Fix**: Added `Error()` method to `ProblemDetail` for error interface compliance. Added `Body()` method returning a non-error struct, because go-zero's `doHandleError` writes plaintext for any body that implements `error`. Error handler updated to use `errors.As` for ProblemDetail detection.

## Smoke Test Results

All 5 endpoints verified with curl:
- POST /api/v1/urls: 200 with short_code, short_url, original_url
- GET /api/v1/links: 200 with paginated list
- GET /api/v1/links/:code: 200 with detail including total_clicks
- GET /:code: 302 redirect + async click count increment
- DELETE /api/v1/links/:code: 200 hard delete
- 404 errors return proper RFC 7807 JSON (content-type: application/json)
- Validation errors return proper RFC 7807 JSON

## Deviations from Plan

- **Problem Details Body()**: Plan did not anticipate the go-zero behavior where returning an error-implementing struct from `SetErrorHandlerCtx` causes plaintext output. Added `Body()` method to work around this.

## Self-Check: PASSED

All endpoints compile, respond correctly, and return proper RFC 7807 JSON for both success and error cases.
