# Roadmap: Go URL Shortener

## Milestones

- ✅ **v1.0 MVP** — Phases 1-6 (shipped 2026-02-15)
- ✅ **v2.0 Go-Zero Adoption** — Phases 7-11 (shipped 2026-02-22)
- 🚧 **v3.0 Production Readiness** — Phases 12-15 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) — SHIPPED 2026-02-15</summary>

- [x] Phase 1: Foundation & URL Service Core (3/3 plans) — completed 2026-02-15
- [x] Phase 2: Event-Driven Analytics (3/3 plans) — completed 2026-02-15
- [x] Phase 3: Enhanced Analytics (3/3 plans) — completed 2026-02-15
- [x] Phase 4: Link Management (2/2 plans) — completed 2026-02-15
- [x] Phase 5: Production Readiness (5/5 plans) — completed 2026-02-15
- [x] Phase 6: Test Coverage Hardening (2/2 plans) — completed 2026-02-15

Full details: `.planning/milestones/v1.0-ROADMAP.md`

</details>

<details>
<summary>✅ v2.0 Go-Zero Adoption (Phases 7-11) — SHIPPED 2026-02-22</summary>

- [x] Phase 7: Framework Foundation (3/3 plans) — completed 2026-02-16
- [x] Phase 8: Database Migration (3/3 plans) — completed 2026-02-16
- [x] Phase 9: Messaging Migration (3/3 plans) — completed 2026-02-16
- [x] Phase 10: Resilience & Infrastructure (3/3 plans) — completed 2026-02-16
- [x] Phase 11: CI Pipeline & Docker Hardening (3/3 plans) — completed 2026-02-16

Full details: `.planning/milestones/v2.0-ROADMAP.md`

</details>

### 🚧 v3.0 Production Readiness (In Progress)

**Milestone Goal:** Add full observability stack (distributed tracing, log aggregation, metrics and dashboards) and clean up v2.0 tech debt to make the system production-ready.

- [x] **Phase 12: Tech Debt Cleanup** - Remove dead code, harden consumer test coverage, configure Kafka retention (completed 2026-02-22)
- [x] **Phase 13: Distributed Tracing** - OpenTelemetry traces across all three services with Jaeger visualization (completed 2026-02-22)
- [ ] **Phase 14: Log Aggregation** - Loki + log shipper collecting structured JSON logs from all services
- [ ] **Phase 15: Metrics and Dashboards** - Prometheus scraping + Grafana auto-provisioned with RED metrics dashboard

## Phase Details

### Phase 12: Tech Debt Cleanup
**Goal**: Clean up known v2.0 debt — dead code removed, consumer enrichment logic tested, Kafka topic retention explicit
**Depends on**: Nothing (standalone cleanup, no infrastructure dependencies)
**Requirements**: DEBT-01, DEBT-02, DEBT-03
**Success Criteria** (what must be TRUE):
  1. `IncrementClickCount` method and any callers are gone from the codebase and `go build ./...` passes
  2. analytics-consumer test coverage is above 80% as reported by `go test -cover ./...`
  3. Kafka `click-events` topic has explicit retention settings in Docker Compose and does not grow unboundedly
**Plans**: 2 plans

Plans:
- [ ] 12-01-PLAN.md — Dead code removal (IncrementClickCount) + Kafka retention config
- [ ] 12-02-PLAN.md — Consumer test coverage improvement (GeoIPReader interface + tests)

### Phase 13: Distributed Tracing
**Goal**: All three services emit OpenTelemetry traces to Jaeger, including the async Kafka boundary between url-api and analytics-consumer
**Depends on**: Phase 12
**Requirements**: TRACE-01, TRACE-02, TRACE-03
**Success Criteria** (what must be TRUE):
  1. Jaeger UI at http://localhost:16686 shows all three services (url-api, analytics-rpc, analytics-consumer) listed as trace sources
  2. A single URL redirect request produces a connected three-hop trace in Jaeger: HTTP redirect span → Kafka publish → consumer insert
  3. Visiting a short link shows the full trace chain visible in Jaeger with span links across the Kafka async boundary
  4. zRPC calls from url-api to analytics-rpc appear as child spans within the same trace tree
**Plans**: 1 plan

Plans:
- [ ] 13-01-PLAN.md — Jaeger infrastructure + Telemetry YAML config + Kafka context fix

### Phase 14: Log Aggregation
**Goal**: All three services emit structured JSON logs that are collected by a log shipper and queryable in Loki
**Depends on**: Phase 13
**Requirements**: LOG-01, LOG-02, LOG-03
**Success Criteria** (what must be TRUE):
  1. All three services emit structured JSON log lines (not plaintext) visible in container stdout
  2. Loki container is running in Docker Compose and reachable at http://localhost:3100
  3. Log lines from all three services are queryable in Loki using `{service="url-api"}` style label selectors
  4. The log shipper (Alloy or Promtail) collects container logs and forwards them to Loki without manual intervention after `docker compose up`
**Plans**: TBD

Plans:
- [ ] 14-01: TBD

### Phase 15: Metrics and Dashboards
**Goal**: Prometheus scrapes built-in go-zero metrics from all services and Grafana auto-provisions a working RED metrics dashboard on startup
**Depends on**: Phase 14
**Requirements**: METRICS-01, METRICS-02, METRICS-03
**Success Criteria** (what must be TRUE):
  1. Prometheus Status > Targets shows all three service DevServer endpoints (url-api:6470, analytics-rpc:6471, analytics-consumer:6472) as UP
  2. Grafana at http://localhost:3000 starts with Jaeger, Loki, and Prometheus datasources pre-configured — no manual UI setup required
  3. Grafana includes a pre-built dashboard showing rate, error rate, and p99 latency per service using `histogram_quantile()` (not `avg()`)
  4. All panels have data after sending a few test requests — no empty panels on a fresh `docker compose up`
**Plans**: TBD

Plans:
- [ ] 15-01: TBD

## Progress

**Execution Order:** 12 → 13 → 14 → 15

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Foundation & URL Service Core | v1.0 | 3/3 | ✓ Complete | 2026-02-15 |
| 2. Event-Driven Analytics | v1.0 | 3/3 | ✓ Complete | 2026-02-15 |
| 3. Enhanced Analytics | v1.0 | 3/3 | ✓ Complete | 2026-02-15 |
| 4. Link Management | v1.0 | 2/2 | ✓ Complete | 2026-02-15 |
| 5. Production Readiness | v1.0 | 5/5 | ✓ Complete | 2026-02-15 |
| 6. Test Coverage Hardening | v1.0 | 2/2 | ✓ Complete | 2026-02-15 |
| 7. Framework Foundation | v2.0 | 3/3 | ✓ Complete | 2026-02-16 |
| 8. Database Migration | v2.0 | 3/3 | ✓ Complete | 2026-02-16 |
| 9. Messaging Migration | v2.0 | 3/3 | ✓ Complete | 2026-02-16 |
| 10. Resilience & Infrastructure | v2.0 | 3/3 | ✓ Complete | 2026-02-16 |
| 11. CI Pipeline & Docker Hardening | v2.0 | 3/3 | ✓ Complete | 2026-02-16 |
| 12. Tech Debt Cleanup | 2/2 | Complete    | 2026-02-22 | - |
| 13. Distributed Tracing | 1/1 | Complete   | 2026-02-22 | - |
| 14. Log Aggregation | v3.0 | 0/TBD | Not started | - |
| 15. Metrics and Dashboards | v3.0 | 0/TBD | Not started | - |

---
*Roadmap created: 2026-02-14*
*Last updated: 2026-02-22 after v3.0 roadmap creation*
