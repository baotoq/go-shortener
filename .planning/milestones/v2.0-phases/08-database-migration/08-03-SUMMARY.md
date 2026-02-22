---
phase: 08-database-migration
plan: 03
subsystem: analytics-rpc-wiring
tags: [analytics-rpc, postgresql, grpc, click-count]
dependencies:
  requires:
    - 08-01 (PostgreSQL infrastructure + clicks model)
  provides:
    - Analytics RPC fully backed by PostgreSQL
    - GetClickCount returns real click counts from clicks table
  affects:
    - services/analytics-rpc/ (config, svc, logic)
tech_stack:
  patterns:
    - ServiceContext DI with ClicksModel
    - COUNT(*) query for click aggregation
    - Zero-value semantics (unknown codes return 0, not error)
key_files:
  modified:
    - services/analytics-rpc/internal/config/config.go
    - services/analytics-rpc/internal/svc/servicecontext.go
    - services/analytics-rpc/internal/logic/getclickcountlogic.go
    - services/analytics-rpc/etc/analytics.yaml
decisions:
  - Unknown short codes return 0 total_clicks (not error) matching v1.0 behavior
  - Analytics RPC counts clicks independently, does not validate URL existence
  - Port 5433 for PostgreSQL connection (consistent with 08-01)
metrics:
  tasks_completed: 2/2
  commits: 1
  files_modified: 4
---

# Phase 8 Plan 3: Analytics RPC Service Wiring

**One-liner:** Wire Analytics RPC GetClickCount to PostgreSQL, replacing hardcoded stub with real click count queries from the clicks table.

## What Was Built

1. **Config + ServiceContext**: Added `DataSource` config field and initialized `ClickModel` with PostgreSQL connection. Registered `lib/pq` driver via blank import.

2. **GetClickCount Logic**: Replaced stub (hardcoded 42) with real `CountByShortCode` query. For unknown short codes, COUNT returns 0 naturally (not an error), matching the v1.0 API contract.

## Smoke Test Results

Tested via grpcurl against live service:
- Non-existent short code: returns `shortCode` with no `totalClicks` (0, omitted by protobuf)
- After inserting 1 click: returns `totalClicks: 1`
- After inserting 2 clicks: returns `totalClicks: 2`
- Service starts without connection errors on port 8081

## Deviations from Plan

- None. Implementation matched plan exactly.

## Self-Check: PASSED

Service compiles, connects to PostgreSQL, returns real click counts, and handles unknown codes gracefully.
