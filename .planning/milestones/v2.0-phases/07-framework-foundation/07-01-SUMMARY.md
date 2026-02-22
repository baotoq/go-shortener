---
phase: 07-framework-foundation
plan: 01
subsystem: framework-scaffold
tags: [cleanup, go-zero, scaffold]
dependency_graph:
  requires: []
  provides: [go-zero-framework, monorepo-structure, shared-events]
  affects: [all-future-plans]
tech_stack:
  added: [go-zero-v1.10.0, goctl-v1.9.2]
  patterns: [code-generation, monorepo]
key_files:
  created:
    - services/url-api/.gitkeep
    - services/analytics-rpc/.gitkeep
    - common/events/events.go
  modified:
    - go.mod
    - go.sum
    - CLAUDE.md
    - Makefile
    - pkg/problemdetails/problemdetails.go
  deleted:
    - internal/
    - cmd/
    - dapr/
    - db/
    - test/
    - bin/
    - dapr.yaml
    - sqlc.yaml
    - Dockerfile.*
    - docker-compose.yml
decisions:
  - Enhanced ClickEvent with IP/UserAgent/Referer fields
  - Added FieldError and NewValidation to problemdetails
  - Kept nanoid, useragent, geoip2 dependencies for Phase 9
metrics:
  duration: 196s
  tasks_completed: 2/2
  files_modified: 82
  completed: 2026-02-16T00:35:05Z
---

# Phase 7 Plan 1: Framework Scaffold Summary

**One-liner:** Complete v1.0 code removal and go-zero framework installation with monorepo structure.

## What Was Built

Clean break from v1.0 (Chi/Dapr/SQLite) to v2.0 (go-zero framework). Removed all old code and dependencies. Installed go-zero v1.10.0, goctl CLI v1.9.2, and protoc plugins. Created monorepo directory structure (services/, common/, pkg/) ready for service generation in Plans 02-03.

## Task Execution

### Task 1: Remove v1.0 code and dependencies
**Status:** Complete
**Commit:** db857e9
**Duration:** ~98s

Deleted all v1.0 code (10,031 lines removed):
- Removed internal/, cmd/, dapr/, db/, test/, bin/ directories
- Deleted dapr.yaml, sqlc.yaml, Dockerfiles, docker-compose.yml
- Cleaned go.mod: removed Chi, Dapr, SQLite, Zap, golang-migrate dependencies
- Kept nanoid, useragent, geoip2 (needed in Phase 9 analytics enrichment)
- Replaced Makefile with help-only placeholder

**Verification:**
- All 5 v1.0 directories report "No such file or directory"
- Old dependencies not present in go.mod (grep count = 0)
- go mod tidy succeeded

### Task 2: Install go-zero and create project scaffold
**Status:** Complete
**Commit:** aa7a681
**Duration:** ~98s

Installed go-zero framework and tooling:
- go-zero v1.10.0 added to go.mod
- goctl CLI v1.9.2 installed and verified
- protoc-gen-go and protoc-gen-go-grpc installed for zRPC

Created monorepo structure:
- services/url-api/ (empty, ready for Plan 02)
- services/analytics-rpc/ (empty, ready for Plan 03)
- common/events/events.go (shared ClickEvent type)
- pkg/problemdetails/problemdetails.go (enhanced with validation support)

**Key changes:**
- ClickEvent enhanced with IP, UserAgent, Referer fields (moved from enrichment layer to event)
- problemdetails now supports FieldError and NewValidation for per-field validation errors
- CLAUDE.md updated to reflect go-zero v2.0 architecture and patterns

**Verification:**
- goctl --version returns 1.9.2
- go-zero present in go.mod
- All scaffold directories exist
- Code compiles: `go build ./pkg/... ./common/...`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical Functionality] Enhanced ClickEvent structure**
- **Found during:** Task 2 (common/events creation)
- **Issue:** Original v1.0 ClickEvent only had short_code and timestamp. IP, UserAgent, Referer were extracted in analytics enrichment layer, requiring extra DB lookups and complexity.
- **Fix:** Added IP, UserAgent, Referer to ClickEvent type. Enrichment data now travels with the event.
- **Rationale:** Simplifies consumer logic. URL service already has this data at redirect time. Phase 9 will populate these fields when publishing.
- **Files modified:** common/events/events.go
- **Impact:** Cleaner separation of concerns. Analytics service becomes pure consumer, no need to reconstruct request context.

**2. [Rule 2 - Missing Critical Functionality] Added validation error support to problemdetails**
- **Found during:** Task 2 (problemdetails update)
- **Issue:** Original problemdetails package lacked per-field validation error support. go-zero validators need structured field errors.
- **Fix:** Added FieldError type and NewValidation constructor with Errors []FieldError field.
- **Rationale:** go-zero .api validation produces field-level errors. RFC 7807 allows optional fields for problem-specific data. This follows the spec.
- **Files modified:** pkg/problemdetails/problemdetails.go
- **Commit:** aa7a681 (same as Task 2)

## Verification Results

All success criteria met:

✅ All v1.0 code removed (internal/, cmd/, dapr/, db/, test/, Dockerfiles, docker-compose)
✅ Old dependencies removed from go.mod (Chi, Dapr, SQLite, Zap, golang-migrate)
✅ go-zero installed and verified via `goctl --version`
✅ Directory scaffold exists: services/url-api/, services/analytics-rpc/, common/events/, pkg/problemdetails/
✅ Shared event types and problem details compile: `go build ./pkg/... ./common/...`

**Overall verification checks:**
1. `ls internal/ cmd/ dapr/ db/ test/ 2>&1 | grep -c "No such file"` = 5 ✅
2. `goctl --version` = goctl version 1.9.2 darwin/arm64 ✅
3. `grep -c "go-zero" go.mod` = 1 ✅
4. `grep -c "dapr|go-chi|..." go.mod` = 0 ✅
5. `go build ./pkg/... ./common/...` = success ✅
6. All scaffold directories exist ✅

## Self-Check: PASSED

**Created files exist:**
- ✅ /Users/baotoq/Work/go-shortener/services/url-api/.gitkeep
- ✅ /Users/baotoq/Work/go-shortener/services/analytics-rpc/.gitkeep
- ✅ /Users/baotoq/Work/go-shortener/common/events/events.go
- ✅ /Users/baotoq/Work/go-shortener/pkg/problemdetails/problemdetails.go (modified)

**Commits exist:**
- ✅ db857e9 (Task 1: Remove v1.0 code)
- ✅ aa7a681 (Task 2: Install go-zero and scaffold)

**Dependencies verified:**
- ✅ go-zero v1.10.0 in go.mod
- ✅ goctl v1.9.2 installed
- ✅ protoc plugins installed

All artifacts verified. Plan execution complete.

## Technical Notes

**go.mod dependency strategy:**
- go-zero marked as `// indirect` because no code imports it yet
- Plans 02-03 will generate code that imports go-zero, making it a direct dependency
- go mod tidy would remove unused dependencies, but we keep it for next plans

**Directory structure rationale:**
- services/ follows go-zero monorepo convention
- common/ for cross-service types (events, utils)
- pkg/ for reusable libraries (problemdetails)

**Code generation readiness:**
- goctl installed and ready to generate REST service (Plan 02)
- protoc plugins ready to generate zRPC service (Plan 03)
- .gitkeep files ensure empty directories are tracked

## Next Steps

**Plan 02 (URL API Service):**
- Define url.api spec
- Generate REST service with goctl
- Implement URL shortening, redirect, list, delete logic
- PostgreSQL integration via sqlx

**Plan 03 (Analytics RPC Service):**
- Define analytics.proto spec
- Generate zRPC service with goctl
- Implement click aggregation logic
- PostgreSQL integration

**Blockers cleared:**
- None. go-zero framework ready. Scaffold complete.
