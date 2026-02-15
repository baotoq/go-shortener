---
phase: 06-test-coverage-hardening
plan: 01
subsystem: url-service/delivery/http
tags: [testing, coverage, middleware, router, health-checks, event-publishing]
dependency_graph:
  requires: [05-05 (CI pipeline with coverage enforcement)]
  provides: [handler-package-80-percent-coverage, middleware-tests, router-tests, readyz-tests, publishClickEvent-tests]
  affects: [url-service, CI coverage thresholds]
tech_stack:
  added: []
  patterns: [in-memory-sqlite-for-tests, mock-dapr-client, waitgroup-for-goroutine-testing, chi-router-testing]
key_files:
  created:
    - internal/urlservice/delivery/http/middleware_test.go
    - internal/urlservice/delivery/http/router_test.go
  modified:
    - internal/urlservice/delivery/http/handler.go (refactored to use DaprClient interface)
    - internal/urlservice/delivery/http/handler_test.go (added 10 new tests + helper functions)
decisions:
  - "Refactored Handler to accept usecase.DaprClient interface instead of dapr.Client for testability"
  - "Used in-memory SQLite databases for Readyz tests (Option A vs sqlmock)"
  - "Accepted goroutine leak from RateLimiter.StartCleanup in tests (ticker fires every 10 minutes, won't fire during tests)"
  - "Simplified router tests to use redirect routes for rate limit testing (avoids complex POST mocks)"
metrics:
  duration: 7m 0s
  completed_at: 2026-02-15T15:37:48Z
  tasks_completed: 3
  tests_added: 21
  coverage_increase: 46.1pp (44.6% -> 90.7%)
---

# Phase 06 Plan 01: URL Service Handler Package Test Coverage Summary

Raised URL Service handler package test coverage from 44.6% to 90.7% through middleware, router, health check, and event publishing tests.

## Objective Achieved

Raised URL Service handler package test coverage from 44.6% to 90.7% by adding comprehensive unit tests for all previously uncovered code paths: middleware (rate limiter, logger), router setup, readiness probe, fire-and-forget click event publishing, and missing handler error paths.

## What Was Built

### Task 1: Middleware Unit Tests (Rate Limiter + Logger)

Created `middleware_test.go` with 6 tests covering:

**Rate Limiter Tests:**
- Within limit returns 200 with rate limit headers
- Exceeds limit returns 429 with RFC 7807 Problem Details
- Different IPs have independent limits (per-IP isolation)
- Headers set correctly (X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset)

**Logger Middleware Tests:**
- Logs request and calls next handler
- Preserves status code from inner handler

Coverage increased from 44.6% to 62.7% (+18.1pp).

Commit: `6646117`

### Task 2: Handler Gap Tests + Refactor for Testable Dapr Client

**Refactored Handler:**
- Changed `daprClient` field from `dapr.Client` to `usecase.DaprClient` interface
- Changed constructor to accept `usecase.DaprClient` instead of `dapr.Client`
- Removed direct dapr import (only needed via usecase package)
- All existing tests continue to pass (interface change is transparent)

**Added Helper Functions:**
- `setupTestHandlerWithDB(t)` - Creates handler with in-memory SQLite for Readyz tests
- `setupTestHandlerWithDapr(t)` - Creates handler with mock DaprClient for publishClickEvent tests

**Added 10 New Tests:**

*Readyz tests (3):*
- Database healthy, no Dapr → 200
- Database unavailable → 503
- Dapr sidecar unavailable → 503

*publishClickEvent tests (2):*
- With Dapr client, publishes click event (uses WaitGroup pattern)
- Publish event fails, still redirects (fire-and-forget behavior)

*Error path tests (5):*
- GetURLDetails not found → 404
- GetURLDetails server error → 500
- GetLinkDetail server error → 500
- Redirect server error → 500
- CreateShortURL short code conflict → 500

Coverage increased from 62.7% to 82.4% (+19.7pp).

Function-level coverage improvements:
- publishClickEvent: 0% → 75%
- Readyz: 0% → 68%
- GetURLDetails: 41.7% → 100%
- Redirect: 77.8% → 100%

Commit: `fe7d24e`

### Task 3: Router Integration Tests + Coverage Verification

Created `router_test.go` with 5 tests covering:

**Middleware Order Tests:**
- Health check route bypasses rate limiter
- Readyz route bypasses rate limiter
- Business routes are rate limited

**Route Registration Tests:**
- Redirect route (/{code}) is registered
- All API routes registered and functional (GET /api/v1/links, DELETE /api/v1/links/{code})

Coverage increased from 82.4% to 90.7% (+8.3pp).

Function-level coverage improvements:
- NewRouter: 0% → 100%

All 37 tests pass (16 original + 6 middleware + 10 handler + 5 router).

No race conditions detected via `go test -race ./...`.

Commit: `6cf0a7c`

## Deviations from Plan

None - plan executed exactly as written.

## Authentication Gates

None encountered.

## Technical Decisions

### 1. DaprClient Interface for Handler Testability

**Decision:** Refactored Handler to use `usecase.DaprClient` interface instead of `dapr.Client`.

**Reasoning:**
- The `dapr.Client` interface has ~40 methods, making it impossible to mock effectively
- The `usecase.DaprClient` interface already exists with only 3 methods (PublishEvent, InvokeMethod, Close)
- `testutil.MockDaprClient` already implements this interface
- This change enables testing `publishClickEvent` without mocking the entire Dapr SDK

**Impact:** All existing tests continue to pass (nil DaprClient is still supported).

### 2. In-Memory SQLite for Readyz Tests

**Decision:** Used in-memory SQLite databases for Readyz tests (Option A) instead of sqlmock (Option B).

**Reasoning:**
- SQLite is already a project dependency (modernc.org/sqlite)
- In-memory SQLite provides real DB behavior without adding new dependencies
- Simple to use: `sql.Open("sqlite", ":memory:")`
- Aligns with existing repository test patterns (see 05-03)

**Alternative Considered:** sqlmock would work but adds a new dependency.

### 3. Goroutine Leak from RateLimiter.StartCleanup

**Decision:** Accepted goroutine leak in tests from `RateLimiter.StartCleanup`.

**Reasoning:**
- `NewRateLimiter` calls `StartCleanup()` which starts a background goroutine with a 10-minute ticker
- The ticker won't fire during tests (tests complete in milliseconds)
- Fixing this requires production code changes (adding `StopCleanup` method)
- This is out of scope for coverage hardening (plan focused on tests, not refactoring)
- The cleanup goroutine covers only 6 lines and is low-risk code

**Impact:** Minor goroutine leak in tests, no functional issues.

### 4. Simplified Router Tests with Redirect Routes

**Decision:** Used redirect routes (GET /{code}) to exhaust rate limit in router tests instead of POST /api/v1/urls.

**Reasoning:**
- POST /api/v1/urls requires mocking both `FindByOriginalURL` and `Save`
- Redirect route only requires mocking `FindByShortCode`
- Simpler mocks = clearer tests
- Same coverage outcome (tests middleware order, not business logic)

**Impact:** Tests are simpler and more maintainable.

## Coverage Verification

**Final Coverage:** 90.7% (up from 44.6%)

**Per-Function Coverage:**
- NewRateLimiter: >0% (was 0%)
- getLimiter: >0% (was 0%)
- Middleware: >0% (was 0%)
- LoggerMiddleware: >0% (was 0%)
- NewRouter: 100% (was 0%)
- Readyz: 68% (was 0%)
- publishClickEvent: 75% (was 0%)
- GetURLDetails: 100%
- Redirect: 100%

**Uncovered Code:**
- `StartCleanup` goroutine (lines 94-108): 6 lines, ticker loop, hard to test without production changes
- Some error handling branches in Readyz (lines within 68% coverage)

**Project-Wide Coverage:**
- All tests pass: 37 tests in handler package, all project tests pass
- No race conditions detected
- Coverage thresholds (80% total, 70% package) now pass for handler package

## Files Modified

**Created:**
- `.planning/phases/06-test-coverage-hardening/06-01-SUMMARY.md` (this file)
- `internal/urlservice/delivery/http/middleware_test.go` (201 lines, 6 tests)
- `internal/urlservice/delivery/http/router_test.go` (188 lines, 5 tests)

**Modified:**
- `internal/urlservice/delivery/http/handler.go` (refactored DaprClient interface)
- `internal/urlservice/delivery/http/handler_test.go` (added 299 lines, 10 tests + 2 helpers)

## Self-Check: PASSED

**Created files exist:**
- FOUND: internal/urlservice/delivery/http/middleware_test.go
- FOUND: internal/urlservice/delivery/http/router_test.go
- FOUND: .planning/phases/06-test-coverage-hardening/06-01-SUMMARY.md

**Commits exist:**
- FOUND: 6646117 (Task 1: middleware tests)
- FOUND: fe7d24e (Task 2: handler gap tests)
- FOUND: 6cf0a7c (Task 3: router tests)

**Coverage verified:**
- Command: `go test -coverprofile=/tmp/coverage.out -covermode=atomic ./internal/urlservice/delivery/http/ && go tool cover -func=/tmp/coverage.out | grep total`
- Output: `total: (statements) 90.7%`
- Target: >= 80%
- Status: PASSED (90.7% >= 80%)

**Tests verified:**
- Command: `go test -v ./internal/urlservice/delivery/http/`
- Output: `PASS ok go-shortener/internal/urlservice/delivery/http 0.397s`
- Tests: 37 tests pass

**Race conditions verified:**
- Command: `go test -race ./...`
- Output: All packages pass, no race conditions detected
- Status: PASSED

## Next Steps

Continue to Phase 06 Plan 02: Analytics Service Test Coverage Hardening (enrichment, delivery, missing error paths).
