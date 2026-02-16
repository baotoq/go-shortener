---
phase: 11-ci-pipeline-docker-hardening
plan: 02
subsystem: infra
tags: [kafka, docker, docker-compose, networking, advertised-listeners]

requires:
  - phase: 10-resilience-infrastructure
    provides: Docker Compose with Kafka dual listeners, kq.Pusher in url-api
provides:
  - Working Kafka publishing in Docker Compose environment
  - Correct advertised listener configuration for both Docker and local dev
affects: [docker-compose, messaging, analytics-pipeline]

tech-stack:
  added: []
  patterns: [kafka advertised listener matching host port mapping]

key-files:
  created: []
  modified:
    - docker-compose.yml

key-decisions:
  - "Fix PLAINTEXT_HOST advertised listener to match host port mapping (localhost:9092 not localhost:29092)"
  - "Enable auto.create.topics for reliable topic creation in Docker"

patterns-established:
  - "Kafka port mapping: host port must match advertised listener port for client metadata reconnection"

duration: 2min
completed: 2026-02-16
---

# Plan 11-02: Fix Kafka Docker Publishing Summary

**Fixed Kafka advertised listener mismatch preventing kq.Pusher metadata reconnection in Docker and local dev**

## Performance

- **Duration:** 2 min
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Fixed PLAINTEXT_HOST advertised listener from localhost:29092 to localhost:9092
- Enabled auto.create.topics for reliable topic creation in containerized environment
- End-to-end click pipeline should now work: redirect -> Kafka -> consumer -> PostgreSQL

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix Kafka Docker networking** - `4db59d8` (fix)

## Files Created/Modified
- `docker-compose.yml` - Fixed KAFKA_ADVERTISED_LISTENERS, added KAFKA_AUTO_CREATE_TOPICS_ENABLE

## Root Cause Analysis

The Kafka advertised listener configuration had a mismatch:
- **Host port mapping:** `9092:29092` (host 9092 -> container 29092)
- **PLAINTEXT_HOST advertised:** `localhost:29092` (no host port 29092 exists!)

When a Kafka client connects to `localhost:9092` (host) -> container port 29092 (PLAINTEXT_HOST listener), Kafka returns `localhost:29092` as the broker address in metadata. The client then tries to reconnect to `localhost:29092` which has no port mapping, causing silent connection failure.

**Fix:** Changed advertised listener to `PLAINTEXT_HOST://localhost:9092` so the metadata address matches the actual host port mapping.

## Decisions Made
- Kept dual listener architecture (PLAINTEXT for Docker internal, PLAINTEXT_HOST for host)
- Added auto.create.topics as safety net for first-run topic creation
- Fire-and-forget pattern preserved (no code changes to redirect logic)

## Deviations from Plan
None - the root cause was identified as advertised listener mismatch as hypothesized.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Kafka publishing fixed for both Docker and local dev environments
- End-to-end click pipeline ready for verification

---
*Phase: 11-ci-pipeline-docker-hardening*
*Completed: 2026-02-16*
