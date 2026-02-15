# Roadmap: Go URL Shortener

## Milestones

- ✅ **v1.0 MVP** - Phases 1-6 (shipped 2026-02-15)
- 🚧 **v2.0 Go-Zero Adoption** - Phases 7-10 (in progress)

## Phases

<details>
<summary>✅ v1.0 MVP (Phases 1-6) - SHIPPED 2026-02-15</summary>

- [x] Phase 1: Foundation & URL Service Core (3/3 plans) - completed 2026-02-15
- [x] Phase 2: Event-Driven Analytics (3/3 plans) - completed 2026-02-15
- [x] Phase 3: Enhanced Analytics (3/3 plans) - completed 2026-02-15
- [x] Phase 4: Link Management (2/2 plans) - completed 2026-02-15
- [x] Phase 5: Production Readiness (5/5 plans) - completed 2026-02-15
- [x] Phase 6: Test Coverage Hardening (2/2 plans) - completed 2026-02-15

Full details: `.planning/milestones/v1.0-ROADMAP.md`

</details>

### 🚧 v2.0 Go-Zero Adoption (In Progress)

**Milestone Goal:** Full stack rewrite from Chi/Dapr/SQLite to go-zero/zRPC/Kafka/PostgreSQL with feature parity.

#### Phase 7: Framework Foundation
**Goal**: Establish go-zero code generation patterns and service structure
**Depends on**: Phase 6
**Requirements**: FWRK-01, FWRK-02, FWRK-03, FWRK-04, FWRK-05, FWRK-06, FWRK-07, FWRK-08, FWRK-09
**Success Criteria** (what must be TRUE):
  1. URL Service API spec (.api file) generates handlers, routes, and types without syntax errors
  2. Analytics Service RPC spec (.proto file) generates server and client stubs correctly
  3. Request validation works via .api tag rules (no manual validation code needed)
  4. Error responses use RFC 7807 Problem Details format via custom httpx.SetErrorHandler
  5. Services start successfully and load configuration from YAML files
**Plans**: 3 plans

Plans:
- [ ] 07-01-PLAN.md — Clean v1.0 code, install go-zero, create project scaffold
- [ ] 07-02-PLAN.md — URL API service (.api spec, code gen, error handler, stub logic)
- [ ] 07-03-PLAN.md — Analytics RPC service (.proto spec, zRPC gen, stub logic)

#### Phase 8: Database Migration
**Goal**: Migrate from SQLite to PostgreSQL with goctl-generated models
**Depends on**: Phase 7
**Requirements**: DBMG-01, DBMG-02, DBMG-03, DBMG-04, DBMG-05, DBMG-06, DBMG-07
**Success Criteria** (what must be TRUE):
  1. PostgreSQL schema exists with urls and clicks tables matching current SQLite structure
  2. URL Service can create short URLs and store them in PostgreSQL
  3. Analytics Service can write click events to PostgreSQL
  4. Cursor pagination works correctly with PostgreSQL (list links endpoint)
  5. Search and filtering operations work with PostgreSQL (search by URL, filter by status)
**Plans**: TBD

Plans:
- [ ] 08-01: TBD
- [ ] 08-02: TBD

#### Phase 9: Messaging Migration
**Goal**: Replace Dapr pub/sub with Kafka via go-queue and add zRPC service communication
**Depends on**: Phase 8
**Requirements**: MSNG-01, MSNG-02, MSNG-03, MSNG-04, MSNG-05, MSNG-06, SRVC-01, SRVC-02, SRVC-03, SRVC-04
**Success Criteria** (what must be TRUE):
  1. URL Service publishes ClickEvent to Kafka after redirect (fire-and-forget pattern preserved)
  2. Analytics Consumer processes ClickEvent from Kafka and enriches with GeoIP, device/browser, referer
  3. Duplicate click events are handled gracefully without inflating analytics counts
  4. Analytics Service exposes GetClickCount via zRPC and URL Service can call it successfully
  5. Redirect response time is unaffected by Kafka publishing (user never blocked by analytics)
**Plans**: TBD

Plans:
- [ ] 09-01: TBD
- [ ] 09-02: TBD

#### Phase 10: Resilience & Infrastructure
**Goal**: Enable go-zero production features and Docker Compose orchestration
**Depends on**: Phase 9
**Requirements**: RESL-01, RESL-02, RESL-03, RESL-04, RESL-05, RESL-06, INFR-01, INFR-02, INFR-03, INFR-04, TEST-01, TEST-02, TEST-03, TEST-04
**Success Criteria** (what must be TRUE):
  1. Services expose Prometheus metrics on /metrics endpoint with request duration and status code data
  2. Circuit breaker protects PostgreSQL and Kafka calls (fails fast when downstream is unhealthy)
  3. Docker Compose starts all services (PostgreSQL, Zookeeper, Kafka, url-service, analytics-rpc, analytics-consumer) successfully
  4. End-to-end flow works: shorten URL → visit short link → redirect → click event published → analytics enriched → counts queryable via zRPC
  5. Test coverage maintains 80%+ across services with unit and integration tests passing in CI
**Plans**: TBD

Plans:
- [ ] 10-01: TBD
- [ ] 10-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 7 → 8 → 9 → 10

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Foundation & URL Service Core | v1.0 | 3/3 | ✓ Complete | 2026-02-15 |
| 2. Event-Driven Analytics | v1.0 | 3/3 | ✓ Complete | 2026-02-15 |
| 3. Enhanced Analytics | v1.0 | 3/3 | ✓ Complete | 2026-02-15 |
| 4. Link Management | v1.0 | 2/2 | ✓ Complete | 2026-02-15 |
| 5. Production Readiness | v1.0 | 5/5 | ✓ Complete | 2026-02-15 |
| 6. Test Coverage Hardening | v1.0 | 2/2 | ✓ Complete | 2026-02-15 |
| 7. Framework Foundation | v2.0 | 0/3 | Not started | - |
| 8. Database Migration | v2.0 | 0/TBD | Not started | - |
| 9. Messaging Migration | v2.0 | 0/TBD | Not started | - |
| 10. Resilience & Infrastructure | v2.0 | 0/TBD | Not started | - |

---
*Roadmap created: 2026-02-14*
*Last updated: 2026-02-16 after Phase 7 planning*
