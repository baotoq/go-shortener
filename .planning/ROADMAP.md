# Roadmap: Go URL Shortener

## Overview

This roadmap delivers a URL shortener built with Go and Dapr in 5 phases. We start with core value delivery (shorten + redirect), layer on event-driven analytics via Dapr pub/sub, enhance analytics with tracking features, add link management capabilities, and finish with production-ready deployment infrastructure. Each phase delivers verifiable user value while building toward a production-quality distributed system.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Foundation & URL Service Core** - Shorten URLs and redirect users with clean architecture
- [x] **Phase 2: Event-Driven Analytics** - Track clicks asynchronously via Dapr pub/sub
- [x] **Phase 3: Enhanced Analytics** - Geo-location, device, and traffic source tracking
- [x] **Phase 4: Link Management** - List and manage created short links
- [x] **Phase 5: Production Readiness** - Testing, CI/CD, Docker, and production deployment

## Phase Details

### Phase 1: Foundation & URL Service Core
**Goal**: Users can shorten URLs and be redirected reliably with proper error handling
**Depends on**: Nothing (first phase)
**Requirements**: URL-01, URL-02, URL-03, URL-04, INFR-03, INFR-04, INFR-05
**Success Criteria** (what must be TRUE):
  1. User submits a long URL via API and receives a short link with auto-generated code
  2. User visiting a short link is redirected (302) to the original URL in under 10ms
  3. API validates URLs and returns clear error messages for invalid input
  4. API returns 404 with meaningful error for non-existent short codes
  5. API rejects excessive requests from same client (rate limiting active)
**Plans**: 3 plans

Plans:
- [x] 01-01-PLAN.md -- Project scaffold, Go module, SQLite schema, sqlc code generation
- [x] 01-02-PLAN.md -- Domain entities, repository implementation, URL service with business logic
- [x] 01-03-PLAN.md -- HTTP handlers, middleware, router, rate limiter, main.go wiring

### Phase 2: Event-Driven Analytics
**Goal**: Click events are tracked asynchronously without blocking redirects
**Depends on**: Phase 1
**Requirements**: ANLT-01, ANLT-02, ANLT-03, INFR-01
**Success Criteria** (what must be TRUE):
  1. Each redirect publishes a click event via Dapr pub/sub (fire-and-forget)
  2. Analytics Service receives and processes click events independently
  3. User can query total click count for any short link via Analytics API
  4. Click tracking failures do not prevent redirects from completing
**Plans**: 3 plans

Plans:
- [x] 02-01-PLAN.md -- Restructure repo for multi-service, Dapr infrastructure, shared event types
- [x] 02-02-PLAN.md -- URL Service Dapr pub/sub click event publishing
- [x] 02-03-PLAN.md -- Analytics Service (schema, repository, usecase, handler, Dapr subscription)

### Phase 3: Enhanced Analytics
**Goal**: Analytics include rich context (geo-location, device, traffic source)
**Depends on**: Phase 2
**Requirements**: ANLT-04, ANLT-05, ANLT-06
**Success Criteria** (what must be TRUE):
  1. Analytics API returns country-level geo-location from IP address
  2. Analytics API returns device type and browser from User-Agent
  3. Analytics API returns traffic source from Referer header
  4. All enrichment happens in Analytics Service (redirect path stays fast)
**Plans**: 3 plans

Plans:
- [x] 03-01-PLAN.md -- Extend ClickEvent payload, enrichment services (GeoIP, User-Agent, Referer)
- [x] 03-02-PLAN.md -- Database migration, sqlc queries, repository for enriched click data
- [x] 03-03-PLAN.md -- AnalyticsService enrichment wiring, summary/detail HTTP endpoints

### Phase 4: Link Management
**Goal**: Users can discover and manage their created short links
**Depends on**: Phase 1
**Requirements**: MGMT-01
**Success Criteria** (what must be TRUE):
  1. User can list all created short links via API
  2. API returns link metadata (original URL, short code, creation date)
  3. List supports pagination for large result sets
**Plans**: 2 plans

Plans:
- [x] 04-01-PLAN.md -- URL Service list, detail, delete endpoints with pagination, filtering, search, and click count enrichment
- [x] 04-02-PLAN.md -- Analytics Service link-deleted event subscription and click data cascade deletion

### Phase 5: Production Readiness
**Goal**: Services are tested, containerized, and deployable to production
**Depends on**: Phase 1, Phase 2
**Requirements**: PROD-01, PROD-02, PROD-03, PROD-04, INFR-02
**Success Criteria** (what must be TRUE):
  1. All layers (handler, service, repository) have unit tests with >80% coverage
  2. Integration tests run against real Dapr sidecars in CI
  3. Docker Compose starts both services with Dapr sidecars locally
  4. CI pipeline (lint, test, build) runs on every commit via GitHub Actions
  5. Services use Dapr service invocation for cross-service calls when needed
**Plans**: 5 plans

Plans:
- [x] 05-01-PLAN.md -- Mockery setup, mock generation, URL Service usecase unit tests
- [x] 05-02-PLAN.md -- Analytics usecase tests, handler tests for both services
- [x] 05-03-PLAN.md -- Health check endpoints, repository integration tests
- [x] 05-04-PLAN.md -- Dockerfiles, Docker Compose with Dapr sidecars, golangci-lint config
- [x] 05-05-PLAN.md -- GitHub Actions CI pipeline, coverage thresholds, Makefile targets

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation & URL Service Core | 3/3 | ✓ Complete | 2026-02-15 |
| 2. Event-Driven Analytics | 3/3 | ✓ Complete | 2026-02-15 |
| 3. Enhanced Analytics | 3/3 | ✓ Complete | 2026-02-15 |
| 4. Link Management | 2/2 | ✓ Complete | 2026-02-15 |
| 5. Production Readiness | 5/5 | ✓ Complete | 2026-02-15 |

---
*Roadmap created: 2026-02-14*
*Last updated: 2026-02-15*
