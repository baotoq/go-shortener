---
phase: 14-log-aggregation
plan: 01
subsystem: infra
tags: [loki, alloy, grafana, docker, logging, observability]

# Dependency graph
requires:
  - phase: 13-distributed-tracing
    provides: Jaeger + OTel tracing infrastructure already in docker-compose.yml
provides:
  - Structured JSON log encoding in all 3 services (6 YAML configs)
  - Loki 3.0 single-binary filesystem storage configuration
  - Grafana Alloy Docker container log collection pipeline
  - Loki + Alloy services integrated into Docker Compose with health dependencies
affects: [15-metrics, future-observability, grafana-dashboards]

# Tech tracking
tech-stack:
  added: [grafana/loki:3.0.0, grafana/alloy:v1.7.5]
  patterns:
    - Alloy Docker socket discovery with service label extraction from com.docker.compose.service metadata
    - Loki single-binary mode with tsdb schema v13 and filesystem storage for local dev

key-files:
  created:
    - infra/loki/loki-config.yaml
    - infra/alloy/config.alloy
  modified:
    - services/url-api/etc/url.yaml
    - services/url-api/etc/url-docker.yaml
    - services/analytics-rpc/etc/analytics.yaml
    - services/analytics-rpc/etc/analytics-docker.yaml
    - services/analytics-consumer/etc/consumer.yaml
    - services/analytics-consumer/etc/consumer-docker.yaml
    - docker-compose.yml

key-decisions:
  - "Loki 3.0 requires tsdb store + schema v13 (not boltdb/v11 from older docs)"
  - "auth_enabled: false — no multi-tenancy needed for local dev environment"
  - "Alloy uses loki.source.docker (not promtail) — Alloy is the modern Grafana agent"
  - "service label derived from __meta_docker_container_label_com_docker_compose_service — auto-set by Docker Compose"
  - "Alloy depends_on loki with service_healthy condition — ensures Loki ready before log ingestion starts"

patterns-established:
  - "Alloy Docker discovery pattern: discovery.docker -> discovery.relabel -> loki.source.docker -> loki.write"
  - "Loki write endpoint uses Docker Compose service name (http://loki:3100) not localhost"

requirements-completed: [LOG-01, LOG-02, LOG-03]

# Metrics
duration: 2min
completed: 2026-02-22
---

# Phase 14 Plan 01: Log Aggregation Infrastructure Summary

**Loki 3.0 + Grafana Alloy pipeline collecting structured JSON logs from all 3 services via Docker socket discovery with service-labeled Loki streams**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-22T08:07:30Z
- **Completed:** 2026-02-22T08:09:42Z
- **Tasks:** 2
- **Files modified:** 9 (6 YAML configs, 2 new infra files, docker-compose.yml)

## Accomplishments

- Added explicit `Encoding: json` to Log block in all 6 service YAML configs (url-api, analytics-rpc, analytics-consumer — both local and docker variants)
- Created `infra/loki/loki-config.yaml` with single-binary filesystem storage (auth_enabled: false, tsdb schema v13, /tmp/loki path prefix)
- Created `infra/alloy/config.alloy` with Docker socket discovery, service label extraction from Compose metadata, and Loki push endpoint
- Added `loki` and `alloy` services to docker-compose.yml with `loki-data` named volume and proper health dependency chain

## Task Commits

Each task was committed atomically:

1. **Task 1: Add JSON encoding + create Loki/Alloy infra configs** - `26f2061` (feat)
2. **Task 2: Add Loki and Alloy services to Docker Compose** - `2312ab7` (feat)

**Plan metadata:** (docs commit — see final_commit)

## Files Created/Modified

- `infra/loki/loki-config.yaml` - Loki 3.0 single-binary config with tsdb schema v13 and filesystem storage
- `infra/alloy/config.alloy` - Alloy pipeline: Docker discovery -> service label relabel -> loki.source.docker -> loki.write
- `docker-compose.yml` - Added loki (port 3100) and alloy (port 12345) services with loki-data volume
- `services/url-api/etc/url.yaml` - Added Encoding: json to Log block
- `services/url-api/etc/url-docker.yaml` - Added Encoding: json to Log block
- `services/analytics-rpc/etc/analytics.yaml` - Added Encoding: json to Log block
- `services/analytics-rpc/etc/analytics-docker.yaml` - Added Encoding: json to Log block
- `services/analytics-consumer/etc/consumer.yaml` - Added Encoding: json to Log block
- `services/analytics-consumer/etc/consumer-docker.yaml` - Added Encoding: json to Log block

## Decisions Made

- Loki 3.0 requires `store: tsdb` and `schema: v13` — older docs show boltdb/v11 which no longer works
- `auth_enabled: false` for local dev — no multi-tenancy complexity needed
- Alloy (not promtail) — Alloy is the current-generation Grafana agent; promtail is legacy
- `service` label extracted from `__meta_docker_container_label_com_docker_compose_service` — Docker Compose automatically sets this label on all containers, enabling `{service="url-api"}` log queries
- Alloy `depends_on: loki: condition: service_healthy` — guarantees Loki is ready before Alloy attempts to push logs

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. Run `docker compose up` to start the full log aggregation stack.

## Next Phase Readiness

- Log aggregation stack complete and ready for use
- After `docker compose up`, query logs in Loki with `{service="url-api"}`, `{service="analytics-rpc"}`, or `{service="analytics-consumer"}`
- Loki endpoint at http://localhost:3100/ready can be added as Grafana datasource in Phase 15 (Metrics)
- Phase 15 (Metrics) can provision Loki as a datasource alongside Prometheus in Grafana

## Self-Check: PASSED

- infra/loki/loki-config.yaml: FOUND
- infra/alloy/config.alloy: FOUND
- 14-01-SUMMARY.md: FOUND
- Commit 26f2061: FOUND
- Commit 2312ab7: FOUND

---
*Phase: 14-log-aggregation*
*Completed: 2026-02-22*
