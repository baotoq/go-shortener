---
phase: 04-link-management
plan: 02
subsystem: api
tags: [dapr, pub-sub, sqlc, event-driven, analytics]

# Dependency graph
requires:
  - phase: 04-01
    provides: Link deletion endpoint with link-deleted event publishing
  - phase: 02-01
    provides: Dapr pub/sub infrastructure and event patterns
provides:
  - Analytics Service subscribes to link-deleted topic
  - Cascade deletion of click data when link is deleted
  - Idempotent delete handler with malformed event acknowledgment
affects: [05-production-readiness]

# Tech tracking
tech-stack:
  added: []
  patterns: [Cascade deletion via pub/sub, CloudEvent unwrapping pattern reuse]

key-files:
  created: []
  modified:
    - db/analytics_query.sql
    - internal/analytics/repository/sqlite/sqlc/* (regenerated)
    - internal/analytics/usecase/click_repository.go
    - internal/analytics/usecase/analytics_service.go
    - internal/analytics/repository/sqlite/click_repository.go
    - internal/analytics/delivery/http/handler.go
    - internal/analytics/delivery/http/router.go

key-decisions:
  - "Idempotent delete handler always returns 200, even if short code doesn't exist"
  - "Malformed events acknowledged with 200 to prevent infinite retries"
  - "Reused CloudEvent unwrapping pattern from HandleClickEvent for consistency"

patterns-established:
  - "Pattern 1: Cascade deletion via asynchronous pub/sub events"
  - "Pattern 2: Fire-and-forget event acknowledgment (always 200) for non-critical failures"

# Metrics
duration: 2m 3s
completed: 2026-02-15
---

# Phase 04 Plan 02: Link Deletion Cascade Summary

**Analytics Service subscribes to link-deleted topic and cascades click data deletion via Dapr pub/sub**

## Performance

- **Duration:** 2m 3s
- **Started:** 2026-02-15T08:50:00Z
- **Completed:** 2026-02-15T08:52:03Z
- **Tasks:** 1
- **Files modified:** 8

## Accomplishments
- Added DeleteClicksByShortCode sqlc query for efficient bulk deletion
- Extended Analytics Service with DeleteClickData method
- Implemented HandleLinkDeleted event handler with CloudEvent unwrapping
- Subscribed to link-deleted topic via Dapr programmatic subscription
- Verified both click and link-deleted subscriptions returned from /dapr/subscribe

## Task Commits

Each task was committed atomically:

1. **Task 1: Add DeleteByShortCode to Analytics Service with Dapr link-deleted subscription** - `7e8acb8` (feat)

## Files Created/Modified
- `db/analytics_query.sql` - Added DeleteClicksByShortCode query for bulk deletion
- `internal/analytics/usecase/click_repository.go` - Extended ClickRepository interface with DeleteByShortCode
- `internal/analytics/repository/sqlite/click_repository.go` - Implemented DeleteByShortCode using sqlc
- `internal/analytics/usecase/analytics_service.go` - Added DeleteClickData service method
- `internal/analytics/delivery/http/handler.go` - Added HandleLinkDeleted event handler
- `internal/analytics/delivery/http/router.go` - Added link-deleted subscription and route
- `internal/analytics/repository/sqlite/sqlc/*` - Regenerated sqlc code

## Decisions Made

None - plan executed exactly as specified. All key decisions were pre-planned:
- Idempotent delete behavior (always 200)
- Malformed event acknowledgment strategy
- CloudEvent unwrapping pattern reuse

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - implementation proceeded smoothly with no blocking issues.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Link deletion cascade complete across both services
- Analytics data properly cleaned up when links are deleted
- Ready for Phase 05 (Production Readiness) work
- No blockers or concerns

## Self-Check: PASSED

All key files verified to exist:
- db/analytics_query.sql contains DeleteClicksByShortCode
- internal/analytics/usecase/click_repository.go contains DeleteByShortCode interface
- internal/analytics/repository/sqlite/click_repository.go contains DeleteByShortCode implementation
- internal/analytics/usecase/analytics_service.go contains DeleteClickData method
- internal/analytics/delivery/http/handler.go contains HandleLinkDeleted handler
- internal/analytics/delivery/http/router.go contains link-deleted subscription

Commit 7e8acb8 verified to exist in git log.

---
*Phase: 04-link-management*
*Completed: 2026-02-15*
