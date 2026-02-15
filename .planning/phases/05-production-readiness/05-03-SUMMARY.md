---
phase: 05-production-readiness
plan: 03
subsystem: health-checks-testing
tags: [health-checks, integration-tests, readiness, liveness, sqlite-testing]
dependency-graph:
  requires: [database-connection, dapr-sidecar, migrations]
  provides: [health-endpoints, repository-integration-tests]
  affects: [url-service, analytics-service]
tech-stack:
  added: [testify, in-memory-sqlite]
  patterns: [health-probes, integration-testing]
key-files:
  created:
    - internal/urlservice/repository/sqlite/url_repository_test.go
    - internal/analytics/repository/sqlite/click_repository_test.go
  modified:
    - internal/urlservice/delivery/http/handler.go
    - internal/urlservice/delivery/http/router.go
    - internal/analytics/delivery/http/handler.go
    - internal/analytics/delivery/http/router.go
    - cmd/url-service/main.go
    - cmd/analytics-service/main.go
decisions:
  - Health checks bypass rate limiting on URL Service
  - Readiness checks verify DB connectivity and Dapr sidecar availability
  - Liveness always returns 200 (process alive)
  - Repository tests use in-memory SQLite with real migrations
  - 2-second timeout for health check database pings
metrics:
  duration: 5m 46s
  tasks-completed: 2
  tests-added: 23
  coverage-url-repo: 75.5%
  coverage-click-repo: 90.0%
  completed: 2026-02-15
---

# Phase 05 Plan 03: Health Checks & Repository Testing Summary

Health check endpoints added to both services, and comprehensive repository integration tests written against in-memory SQLite with real migrations.

## What Was Built

### Task 1: Health Check Endpoints

**URL Service:**
- `/healthz` (liveness): Always returns 200 `{"status": "ok"}` — process is alive
- `/readyz` (readiness): Checks database connectivity and Dapr sidecar health
  - Database: `PingContext` with 2-second timeout
  - Dapr: HTTP GET to `http://localhost:3500/v1.0/healthz`
  - Returns 200 `{"status": "ready"}` when all checks pass
  - Returns 503 `{"status": "unavailable", "reason": "..."}` on failure
- Health checks registered before rate limiter middleware (no rate limiting)

**Analytics Service:**
- `/healthz` (liveness): Always returns 200 `{"status": "ok"}`
- `/readyz` (readiness): Checks database connectivity only
  - Database: `PingContext` with 2-second timeout
  - Returns 200/503 appropriately
- No Dapr client check (Analytics receives events, doesn't invoke services)

**Implementation Details:**
- Added `db *sql.DB` field to both Handler structs
- Updated `NewHandler` constructors to accept database connection
- Health response format: `{"status": "ok|ready|unavailable", "reason": "..." (only on failure)}`
- All responses use `application/json` content type

### Task 2: Repository Integration Tests

**URL Repository Tests (12 tests, 75.5% coverage):**
- `TestURLRepository_Save_CreatesRecord` — basic insertion
- `TestURLRepository_FindByShortCode_Exists` — find existing record
- `TestURLRepository_FindByShortCode_NotFound` — returns `domain.ErrURLNotFound`
- `TestURLRepository_FindByOriginalURL_Exists` — deduplication lookup
- `TestURLRepository_FindByOriginalURL_NotFound` — returns error
- `TestURLRepository_Save_DuplicateShortCode_ReturnsError` — UNIQUE constraint
- `TestURLRepository_FindAll_Pagination` — limit/offset behavior
- `TestURLRepository_FindAll_SortOrder` — asc vs desc ordering
- `TestURLRepository_FindAll_SearchFilter` — substring search
- `TestURLRepository_Count_WithFilters` — count with search filter
- `TestURLRepository_Delete_RemovesRecord` — deletion works
- `TestURLRepository_Delete_Idempotent` — deleting non-existent code succeeds

**Click Repository Tests (11 tests, 90.0% coverage):**
- `TestClickRepository_InsertClick_StoresRecord` — basic insertion
- `TestClickRepository_CountByShortCode_ReturnsCount` — total click count
- `TestClickRepository_CountByShortCode_NoClicks_ReturnsZero` — zero state
- `TestClickRepository_CountInRange_FiltersCorrectly` — time-range filtering
- `TestClickRepository_CountByCountryInRange_GroupsCorrectly` — country grouping
- `TestClickRepository_CountByDeviceInRange_GroupsCorrectly` — device type grouping
- `TestClickRepository_CountBySourceInRange_GroupsCorrectly` — traffic source grouping
- `TestClickRepository_GetClickDetails_ReturnsPaginated` — pagination with hasMore
- `TestClickRepository_GetClickDetails_Cursor` — cursor-based pagination
- `TestClickRepository_DeleteByShortCode_RemovesAll` — cascade deletion
- `TestClickRepository_DeleteByShortCode_OnlyAffectsTargetCode` — isolation

**Test Infrastructure:**
- `setupTestDB(t)` helper: Opens `:memory:` SQLite, runs migrations, auto-cleanup
- Uses `database.RunMigrations` from each service's database package
- Import `_ "modernc.org/sqlite"` for driver registration
- Uses `testify/require` for setup, `testify/assert` for assertions
- Each test creates its own data (no shared state)

## Deviations from Plan

None - plan executed exactly as written.

## Verification Results

All verification steps passed:

```bash
# Build succeeded
go build ./cmd/url-service && go build ./cmd/analytics-service
# ✅ Both services compile

# Health check handlers exist
grep -n "Healthz\|Readyz" internal/urlservice/delivery/http/handler.go
# ✅ Lines 376-383: Both methods present

grep -n "Healthz\|Readyz" internal/analytics/delivery/http/handler.go
# ✅ Lines 337-344: Both methods present

# Health check routes registered
grep -n "healthz\|readyz" internal/urlservice/delivery/http/router.go
# ✅ Lines 22-23: Routes registered before rate limiter

grep -n "healthz\|readyz" internal/analytics/delivery/http/router.go
# ✅ Lines 21-22: Routes registered

# Repository tests pass
go test ./internal/urlservice/repository/sqlite/ -v
# ✅ 12/12 tests PASS

go test ./internal/analytics/repository/sqlite/ -v
# ✅ 11/11 tests PASS

# Coverage checks
go test ./internal/urlservice/repository/sqlite/ -cover
# ✅ 75.5% coverage

go test ./internal/analytics/repository/sqlite/ -cover
# ✅ 90.0% coverage

# Full build
go build ./...
# ✅ All packages compile
```

## Success Criteria

All criteria met:

- ✅ Both services have /healthz (200 always) and /readyz (200 when healthy, 503 when not)
- ✅ Health checks bypass rate limiting on URL Service
- ✅ URL repository has 12 integration tests with 75.5% coverage
- ✅ Click repository has 11 integration tests with 90.0% coverage
- ✅ All tests use in-memory SQLite (no file system state)
- ✅ Dapr service invocation verified for click count enrichment (no changes needed)

## Implementation Notes

**Health Check Design:**
- Liveness vs Readiness distinction follows Kubernetes best practices
- Liveness: "Is the process alive?" (always 200 unless crashed)
- Readiness: "Can the service accept traffic?" (checks dependencies)
- URL Service checks both DB and Dapr (publishes events)
- Analytics Service checks only DB (receives events via sidecar)

**Testing Strategy:**
- Integration tests over unit tests for data layer (sqlc-generated code)
- Real SQLite with migrations catches SQL syntax errors, schema mismatches
- In-memory database keeps tests fast (<1s for 23 tests)
- Each test is isolated (own data, no shared state)
- Tests verify business logic, edge cases, error handling

**Dapr Service Invocation:**
- Already implemented in Phase 04 (URL Service → Analytics Service for click counts)
- Uses `daprClient.InvokeMethod` with 5-second timeout
- Fallback to 0 clicks on error
- No code changes needed for this plan

## Files Created/Modified

### Created (2 files):
- `internal/urlservice/repository/sqlite/url_repository_test.go` (265 lines)
- `internal/analytics/repository/sqlite/click_repository_test.go` (290 lines)

### Modified (6 files):
- `internal/urlservice/delivery/http/handler.go` — Added db field, Healthz/Readyz methods
- `internal/urlservice/delivery/http/router.go` — Health routes before rate limiter
- `internal/analytics/delivery/http/handler.go` — Added db field, Healthz/Readyz methods
- `internal/analytics/delivery/http/router.go` — Health routes
- `cmd/url-service/main.go` — Pass db to NewHandler
- `cmd/analytics-service/main.go` — Pass db to NewHandler

## Next Steps

This plan prepares for:
- **Plan 04 (Docker Compose):** Health checks enable `healthcheck` directive in compose
- **Plan 05 (Production Deployment):** Kubernetes liveness/readiness probes use these endpoints
- Repository tests validate data layer before production deployment
- SQLite-specific tests will need adaptation when migrating to PostgreSQL (Plan 05)

## Self-Check: PASSED

**Created files verified:**
```bash
[ -f "/Users/baotoq/Work/go-shortener/internal/urlservice/repository/sqlite/url_repository_test.go" ] && echo "FOUND"
# FOUND

[ -f "/Users/baotoq/Work/go-shortener/internal/analytics/repository/sqlite/click_repository_test.go" ] && echo "FOUND"
# FOUND
```

**Commits verified:**
```bash
git log --oneline | grep -q "86d954c" && echo "FOUND: Task 1 commit"
# FOUND: Task 1 commit

git log --oneline | grep -q "1911262" && echo "FOUND: Task 2 commit"
# FOUND: Task 2 commit
```

All artifacts present and committed.
