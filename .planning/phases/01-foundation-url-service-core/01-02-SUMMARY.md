---
phase: 01-foundation-url-service-core
plan: 02
subsystem: business-logic
tags: [go, clean-architecture, domain-driven-design, nanoid, validation]

# Dependency graph
requires:
  - phase: 01-foundation-url-service-core
    plan: 01
    provides: SQLite schema and sqlc-generated queries
provides:
  - Domain entities (URL, sentinel errors)
  - URLRepository interface (dependency inversion)
  - SQLite repository implementation using sqlc
  - URLService with validation, deduplication, NanoID generation, collision retry
affects: [01-03]

# Tech tracking
tech-stack:
  added: [github.com/matoous/go-nanoid/v2]
  patterns: [dependency-inversion, repository-pattern, domain-errors]

key-files:
  created:
    - internal/domain/url.go
    - internal/domain/errors.go
    - internal/usecase/url_repository.go
    - internal/repository/sqlite/url_repository.go
    - internal/usecase/url_service.go
  modified:
    - go.mod
    - go.sum

key-decisions:
  - "URLRepository interface defined in usecase package (dependency inversion principle)"
  - "Repository Save method takes primitives (shortCode, originalURL) and returns domain.URL with DB-populated fields"
  - "Validate URLs: http/https only, host required, max 2048 chars, localhost allowed"
  - "Generate 8-char NanoID with 62-char alphabet (a-z, A-Z, 0-9)"
  - "Deduplication: FindByOriginalURL returns existing code for duplicate URLs"
  - "Collision retry: up to 5 attempts, detect via UNIQUE constraint error message"

patterns-established:
  - "Dependency inversion: service depends on interface, repository implements interface"
  - "Domain errors: sentinel errors in domain package, wrapped by service layer"
  - "Clean separation: domain (entities), usecase (interfaces + services), repository (implementations)"

# Metrics
duration: 2m 12s
completed: 2026-02-14
---

# Phase 01 Plan 02: Business Logic Layer Summary

**Complete URL shortening business logic with validation, NanoID generation, deduplication, and collision retry using clean architecture**

## Performance

- **Duration:** 2m 12s
- **Started:** 2026-02-14T18:19:29Z
- **Completed:** 2026-02-14T18:21:41Z
- **Tasks:** 2
- **Files created:** 5
- **Files modified:** 2

## Accomplishments

- Created domain entities and sentinel errors for URL shortening
- Defined URLRepository interface in usecase package (dependency inversion)
- Implemented SQLite repository using sqlc-generated queries
- Built URLService with complete business logic: validation, deduplication, NanoID generation, collision retry
- Established clean architecture pattern: domain → usecase → repository

## Task Commits

Each task was committed atomically:

1. **Task 1: Create domain entities and repository interface** - `f3f3a1b` (feat)
2. **Task 2: Implement SQLite repository and URL service with business logic** - `cde6b9a` (feat)

## Files Created/Modified

**Created:**
- `internal/domain/url.go` - URL entity with ID, ShortCode, OriginalURL, CreatedAt
- `internal/domain/errors.go` - Sentinel errors (ErrURLNotFound, ErrInvalidURL, ErrShortCodeConflict)
- `internal/usecase/url_repository.go` - URLRepository interface with Save, FindByShortCode, FindByOriginalURL
- `internal/repository/sqlite/url_repository.go` - SQLite implementation using sqlc, converts between sqlc models and domain entities
- `internal/usecase/url_service.go` - URLService with CreateShortURL and GetByShortCode methods

**Modified:**
- `go.mod` - Added github.com/matoous/go-nanoid/v2 dependency
- `go.sum` - Dependency checksums

## Decisions Made

- **Dependency inversion:** URLRepository interface lives in usecase package (not repository), so the service depends on an abstraction. The repository package imports usecase to implement the interface. This follows clean architecture principles.

- **Repository signature:** Save method takes primitives (shortCode, originalURL strings) rather than domain.URL, because the caller provides these values and the database populates ID and CreatedAt. This makes the contract clear.

- **URL validation rules:**
  - Scheme must be http or https (case-insensitive)
  - Host must not be empty
  - Maximum length 2048 characters
  - Localhost and private IPs allowed (development convenience)

- **Short code generation:** 8-character NanoID with 62-character alphabet (a-z, A-Z, 0-9). Provides ~218 trillion possible combinations.

- **Deduplication:** Before generating a new code, check if original URL already exists via FindByOriginalURL. If found, return existing short code. This prevents duplicate entries per user requirements.

- **Collision handling:** Retry up to 5 times on UNIQUE constraint violation (detected via error message string). After 5 failures, return ErrShortCodeConflict. Check context cancellation between retries.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing go-nanoid dependency**
- **Found during:** Task 2 verification
- **Issue:** go build failed with "no required module provides package github.com/matoous/go-nanoid/v2"
- **Fix:** Installed missing dependency via `go get github.com/matoous/go-nanoid/v2`
- **Files modified:** go.mod, go.sum
- **Commit:** cde6b9a (bundled with Task 2)
- **Root cause:** Plan 01 Task 1 was supposed to install dependencies but didn't. This was a blocking issue preventing Task 2 completion, so auto-fixed per Rule 3.

## Issues Encountered

None - after adding missing dependency, all packages compiled successfully and passed go vet.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Domain entities and errors defined
- Repository interface established with clean dependency inversion
- SQLite repository implementation complete
- URL service with all business logic ready (validation, deduplication, NanoID, retry)
- Ready for Plan 03 (HTTP transport layer with chi router and RFC 7807 error responses)

## Self-Check: PASSED

All files verified:
```bash
[ -f "internal/domain/url.go" ] && echo "FOUND: internal/domain/url.go" || echo "MISSING: internal/domain/url.go"
[ -f "internal/domain/errors.go" ] && echo "FOUND: internal/domain/errors.go" || echo "MISSING: internal/domain/errors.go"
[ -f "internal/usecase/url_repository.go" ] && echo "FOUND: internal/usecase/url_repository.go" || echo "MISSING: internal/usecase/url_repository.go"
[ -f "internal/repository/sqlite/url_repository.go" ] && echo "FOUND: internal/repository/sqlite/url_repository.go" || echo "MISSING: internal/repository/sqlite/url_repository.go"
[ -f "internal/usecase/url_service.go" ] && echo "FOUND: internal/usecase/url_service.go" || echo "MISSING: internal/usecase/url_service.go"
```
