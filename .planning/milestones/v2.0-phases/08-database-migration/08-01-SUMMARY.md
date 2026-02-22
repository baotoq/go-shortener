---
phase: 08-database-migration
plan: 01
subsystem: database-infrastructure
tags: [postgresql, docker-compose, goctl-model, schema]
dependencies:
  requires: []
  provides:
    - PostgreSQL 16 via Docker Compose
    - urls and clicks tables with UUIDv7-compatible schema
    - goctl-generated CRUD models for both tables
    - Custom query methods (pagination, search, click count)
  affects:
    - services/url-api/model/
    - services/analytics-rpc/model/
    - Makefile
tech_stack:
  added:
    - postgres:16-alpine (Docker)
    - github.com/lib/pq v1.11.2
    - goctl model pg datasource
  patterns:
    - goctl two-file model pattern (custom + generated)
    - Atomic click count increment (SET click_count = click_count + 1)
    - OFFSET/LIMIT pagination with ILIKE search
key_files:
  created:
    - docker-compose.yml
    - services/migrations/000001_create_urls.up.sql
    - services/migrations/000001_create_urls.down.sql
    - services/migrations/000002_create_clicks.up.sql
    - services/migrations/000002_create_clicks.down.sql
    - services/url-api/model/urlsmodel.go
    - services/url-api/model/urlsmodel_gen.go
    - services/url-api/model/vars.go
    - services/analytics-rpc/model/clicksmodel.go
    - services/analytics-rpc/model/clicksmodel_gen.go
    - services/analytics-rpc/model/vars.go
  modified:
    - Makefile
    - go.mod
    - go.sum
decisions:
  - Port 5433 for PostgreSQL (5432 was occupied by existing container)
  - OFFSET/LIMIT pagination matching current API contract (page/per_page)
  - Deferred enrichment columns on clicks to Phase 9
  - TIMESTAMPTZ for all timestamp columns
  - idx_clicks_short_code index for GetClickCount performance
metrics:
  tasks_completed: 2/2
  commits: 2
  files_created: 11
  files_modified: 3
---

# Phase 8 Plan 1: PostgreSQL Infrastructure & Model Generation

**One-liner:** PostgreSQL Docker Compose setup with DDL migrations and goctl-generated models including custom pagination, search, and click count methods.

## What Was Built

1. **Docker Compose** with PostgreSQL 16 Alpine on port 5433 (5432 occupied by existing container), healthcheck, and named volume for persistence.

2. **SQL Migrations** for two tables:
   - `urls`: UUID PK, short_code UNIQUE VARCHAR(8), original_url TEXT, click_count BIGINT DEFAULT 0, created_at TIMESTAMPTZ
   - `clicks`: UUID PK, short_code VARCHAR(8) with index, clicked_at TIMESTAMPTZ

3. **goctl-generated models** via `goctl model pg datasource`:
   - `urlsmodel_gen.go`: Insert, FindOne, FindOneByShortCode, Update, Delete
   - `clicksmodel_gen.go`: Insert, FindOne, Update, Delete

4. **Custom model methods** (safe from regeneration):
   - `UrlsModel.ListWithPagination()`: OFFSET/LIMIT with ILIKE search, sort, order
   - `UrlsModel.IncrementClickCount()`: Atomic `click_count = click_count + 1`
   - `ClicksModel.CountByShortCode()`: `SELECT COUNT(*) WHERE short_code = $1`

5. **Makefile targets**: db-up, db-down, db-migrate, gen-url-model, gen-clicks-model

## Deviations from Plan

- **Port changed to 5433**: Port 5432 was occupied by an existing `litellm_db` PostgreSQL container. Used 5433 instead. All config and Makefile targets updated accordingly.

## Self-Check: PASSED

All files exist, both models compile, PostgreSQL running with correct schema.
