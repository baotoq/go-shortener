---
phase: 12-tech-debt-cleanup
plan: 01
subsystem: api
tags: [go-zero, kafka, docker-compose, dead-code-removal, model]

# Dependency graph
requires:
  - phase: 09-messaging-migration
    provides: Kafka-based click counting that replaced IncrementClickCount
provides:
  - UrlsModel interface without IncrementClickCount (dead code removed)
  - MockUrlsModel without IncrementClickCount
  - Kafka click-events topic with 24h/1GB retention bounds
affects: [13-tracing, 14-metrics, 15-grafana]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dead code removed from model interface/implementation/mock simultaneously to keep compile-time assertion valid"
    - "Kafka retention configured at broker level via env vars — applies to all auto-created topics"

key-files:
  created: []
  modified:
    - services/url-api/model/urlsmodel.go
    - services/url-api/model/mock_urlsmodel.go
    - test/integration/shorten_test.go
    - docker-compose.yml

key-decisions:
  - "Removed IncrementClickCount from interface, implementation, mock, and integration test atomically — click counting is now exclusively handled by analytics-consumer via Kafka"
  - "Kafka retention set to 24h time limit and 1GB per-partition size limit with 100MB segments — reduces default 7-day retention to prevent unbounded disk growth"

patterns-established:
  - "Model dead code: remove from interface + implementation + mock + tests simultaneously to keep var _ UrlsModel = (*customUrlsModel)(nil) assertion passing"

requirements-completed:
  - DEBT-01
  - DEBT-03

# Metrics
duration: 2min
completed: 2026-02-22
---

# Phase 12 Plan 01: Dead Code Removal and Kafka Retention Summary

**Removed IncrementClickCount dead method from url-api model layer and bounded Kafka click-events topic to 24h/1GB retention in Docker Compose**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-22T08:29:55Z
- **Completed:** 2026-02-22T08:32:00Z
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- Removed IncrementClickCount from UrlsModel interface, customUrlsModel implementation, MockUrlsModel struct/method, and integration test block
- Added KAFKA_LOG_RETENTION_HOURS=24, KAFKA_LOG_RETENTION_BYTES=1073741824, KAFKA_LOG_SEGMENT_BYTES=104857600 to docker-compose.yml kafka service
- go build ./... passes clean with no compile errors after removal

## Task Commits

Each task was committed atomically:

1. **Task 1: Remove dead IncrementClickCount method from urlsmodel and mock** - `27121b3` (refactor)
2. **Task 2: Configure Kafka topic retention in docker-compose.yml** - `23437df` (chore)

**Plan metadata:** (docs commit follows)

## Files Created/Modified
- `services/url-api/model/urlsmodel.go` - Removed IncrementClickCount from UrlsModel interface and customUrlsModel implementation
- `services/url-api/model/mock_urlsmodel.go` - Removed IncrementClickCountFunc field and IncrementClickCount method from MockUrlsModel
- `test/integration/shorten_test.go` - Removed "Test: Increment click count" block
- `docker-compose.yml` - Added 3 Kafka retention env vars to kafka service

## Decisions Made
- Removed IncrementClickCount from all four locations atomically (interface, impl, mock, test) to maintain the compile-time interface assertion `var _ UrlsModel = (*MockUrlsModel)(nil)` — removing only some would cause a compile error.
- Kafka retention set at broker-level (not per-topic) since click-events is auto-created — broker defaults apply automatically to new topics.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None - pre-existing local changes to `services/analytics-consumer/internal/mqs/clickeventconsumer_test.go` (GeoIP mock tests added outside this plan's scope) were noted but not included in task commits per scope boundary rules.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- Phase 12 Plan 01 complete: dead code removed, Kafka bounded
- url-api model is clean with no vestigial Phase 8 methods
- Ready for Phase 13 (Tracing) — model interface is now correct and minimal

---
*Phase: 12-tech-debt-cleanup*
*Completed: 2026-02-22*
