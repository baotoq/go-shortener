---
phase: 03-enhanced-analytics
plan: 02
subsystem: analytics
tags:
  - database
  - migration
  - repository
  - sqlc
  - pagination
dependency_graph:
  requires:
    - 02-03-SUMMARY (Analytics Service with click persistence)
  provides:
    - Enrichment columns in clicks table (country_code, device_type, traffic_source)
    - Database indexes for time-range and GROUP BY queries
    - Repository methods for enriched click storage and querying
    - Cursor-based pagination for click details
  affects:
    - internal/analytics/repository/sqlite/click_repository.go (extended with 5 new methods)
    - internal/analytics/usecase/click_repository.go (interface with 7 methods)
    - internal/analytics/usecase/analytics_service.go (temporary bridge for enrichment)
tech_stack:
  added:
    - SQLite ALTER TABLE with enrichment columns
    - Composite indexes for query optimization
    - Base64-encoded cursor pagination
  patterns:
    - Repository pattern with enrichment support
    - Cursor-based pagination using timestamp
    - Grouped aggregations (by country, device, source)
    - Time-range filtering with composite indexes
key_files:
  created:
    - internal/analytics/database/migrations/000002_add_enrichment_columns.up.sql
    - internal/analytics/database/migrations/000002_add_enrichment_columns.down.sql
  modified:
    - db/analytics_schema.sql (added 3 enrichment columns)
    - db/analytics_query.sql (6 new queries)
    - internal/analytics/repository/sqlite/sqlc/analytics_query.sql.go (sqlc-generated)
    - internal/analytics/repository/sqlite/sqlc/models.go (Click model with enrichment)
    - internal/analytics/repository/sqlite/sqlc/querier.go (query interface)
    - internal/analytics/usecase/click_repository.go (7-method interface)
    - internal/analytics/repository/sqlite/click_repository.go (all methods implemented)
    - internal/analytics/usecase/analytics_service.go (temporary bridge)
decisions:
  - decision: Use DEFAULT values for enrichment columns in migration
    rationale: Ensures existing Phase 2 click records migrate cleanly without data loss
    alternatives: NOT NULL without DEFAULT would fail on existing data
  - decision: Keep idx_clicks_short_code from migration 000001
    rationale: Dropping indexes in SQLite ALTER is cumbersome; covered by composite indexes anyway
    alternatives: Could drop if migration complexity was worth it
  - decision: Use base64-encoded timestamp for cursor pagination
    rationale: Simple, URL-safe, works with ORDER BY clicked_at DESC
    alternatives: Opaque cursor (encrypted), offset-based (poor performance)
  - decision: Fetch limit+1 for hasMore detection
    rationale: Single query determines if more pages exist without COUNT(*)
    alternatives: Separate COUNT query (2 queries), always assume hasMore (false positives)
  - decision: Temporary empty strings in analytics_service.go
    rationale: Keeps Plan 02 self-contained; Plan 03 will wire enrichment services
    alternatives: Skip temporary bridge (project wouldn't compile)
metrics:
  duration: 150s
  tasks_completed: 2
  files_created: 2
  files_modified: 8
  commits: 2
  completed_at: 2026-02-15T07:26:41Z
---

# Phase 03 Plan 02: Enrichment Data Layer Summary

**One-liner:** Extended analytics database with country_code, device_type, and traffic_source columns, plus repository methods for grouped counts and cursor-paginated details.

## What Was Built

**Database Schema:**
- Migration 000002 adds 3 enrichment columns (country_code, device_type, traffic_source) with DEFAULT values for backward compatibility
- 5 new indexes for time-range queries and GROUP BY optimizations (idx_clicks_time_range, idx_clicks_country, idx_clicks_device, idx_clicks_source, idx_clicks_summary)
- Updated sqlc schema to reflect final table state

**sqlc Queries:**
- InsertEnrichedClick: 5-parameter insert with enrichment data
- CountClicksInRange: Total clicks within time range
- CountByCountryInRange: Grouped counts by country
- CountByDeviceInRange: Grouped counts by device type
- CountBySourceInRange: Grouped counts by traffic source
- GetClickDetails: Cursor-paginated individual click records (ORDER BY clicked_at DESC, LIMIT N+1)
- CountClicksByShortCode: Retained for backward compatibility

**Repository Layer:**
- ClickRepository interface extended from 2 to 7 methods
- All methods implemented in SQLite repository
- Cursor pagination using base64-encoded timestamps
- Type conversions: sqlc rows → usecase.GroupCount and usecase.ClickDetail
- Compile-time interface verification maintained

**Temporary Bridge:**
- analytics_service.go updated with empty strings for enrichment fields
- TODO(03-03) comment added for enrichment service wiring

## How It Works

**Migration Strategy:**
1. ALTER TABLE adds columns with NOT NULL DEFAULT (ensures existing Phase 2 records migrate)
2. CREATE INDEX statements optimize all query patterns (time-range, GROUP BY)
3. Down migration drops indexes first, then columns (proper cleanup order)

**Cursor Pagination:**
1. Client calls GetClickDetails with cursorTimestamp (max int64 for first page)
2. Repository fetches limit+1 records WHERE clicked_at < cursor ORDER BY clicked_at DESC
3. If len(rows) > limit: hasMore = true, trim to limit, encode last ClickedAt as nextCursor
4. Client passes nextCursor for subsequent pages

**Query Patterns:**
- Time-range filtering: WHERE short_code = ? AND clicked_at >= ? AND clicked_at <= ?
- Grouped aggregations: GROUP BY [dimension] ORDER BY count DESC
- Composite index (idx_clicks_summary) covers: (short_code, clicked_at, country_code, device_type, traffic_source)

## Integration Points

**Upstream Dependencies:**
- Phase 02-03: Analytics Service with click event persistence
- Existing clicks table schema from migration 000001

**Downstream Consumers:**
- Plan 03-03: Will wire enrichment services to populate country_code, device_type, traffic_source
- Plan 03-03: Will create HTTP handlers using new repository methods

**Data Flow:**
```
ClickEvent → analytics_service.RecordClick → repo.InsertClick (with enrichment)
HTTP GET /analytics/{code}/details → repo.GetClickDetails → PaginatedClicks response
HTTP GET /analytics/{code}/summary → repo.CountByCountryInRange, etc. → Summary response
```

## Deviations from Plan

None - plan executed exactly as written. All tasks completed, all verification steps passed.

## Testing Strategy

**Manual Verification:**
- ✓ go build ./... — entire project compiles
- ✓ go vet ./... — no issues
- ✓ sqlc generate — regenerated code successfully
- ✓ Grepped for InsertEnrichedClick, CountByCountryInRange, GetClickDetails in generated code

**Integration Testing (Plan 03-03):**
- Will verify migration runs cleanly on existing analytics.db
- Will test cursor pagination with actual click data
- Will verify grouped counts match expected values
- Will test time-range filtering with various date ranges

## Performance Considerations

**Index Strategy:**
- idx_clicks_time_range: Covers (short_code, clicked_at) for time-range queries
- idx_clicks_country/device/source: Individual dimension queries
- idx_clicks_summary: Composite index for multi-dimension queries
- Trade-off: More indexes = faster reads, slightly slower writes (acceptable for analytics)

**Pagination Efficiency:**
- Cursor-based avoids OFFSET N (which scans+discards N rows)
- ORDER BY clicked_at DESC uses index (no filesort)
- Limit+1 technique avoids separate COUNT(*) query

**Default Values:**
- 'Unknown' and 'Direct' strings ensure no NULL checks needed in application code
- Enables queries without enrichment services (graceful degradation)

## Next Steps

1. **Plan 03-03**: Implement enrichment services (GeoIP, UA parsing, referer extraction)
2. **Plan 03-03**: Wire enrichment to analytics_service.RecordClick (replace empty strings)
3. **Plan 03-03**: Create HTTP handlers for enriched analytics endpoints
4. **Testing**: Run migration 000002 on existing analytics.db, verify backward compatibility
5. **Monitoring**: Track query performance with indexes, optimize if needed

## Files Created

- internal/analytics/database/migrations/000002_add_enrichment_columns.up.sql (9 lines: 3 ALTER TABLE, 5 CREATE INDEX)
- internal/analytics/database/migrations/000002_add_enrichment_columns.down.sql (8 lines: 5 DROP INDEX, 3 ALTER TABLE DROP COLUMN)

## Files Modified

- db/analytics_schema.sql: Added 3 columns to CREATE TABLE
- db/analytics_query.sql: Replaced 2 queries with 7 (5 new, 2 updated)
- internal/analytics/repository/sqlite/sqlc/analytics_query.sql.go: sqlc-generated (260 lines)
- internal/analytics/repository/sqlite/sqlc/models.go: Click struct with 6 fields
- internal/analytics/repository/sqlite/sqlc/querier.go: Interface with 7 query methods
- internal/analytics/usecase/click_repository.go: Interface + 3 types (ClickDetail, GroupCount, PaginatedClicks)
- internal/analytics/repository/sqlite/click_repository.go: 7 methods (160 lines total)
- internal/analytics/usecase/analytics_service.go: RecordClick temporary bridge

## Commits

- 26231f9: feat(03-02): add database migration and sqlc queries for enrichment
- 3f014b9: feat(03-02): extend ClickRepository with enrichment methods

## Self-Check: PASSED

**Created files exist:**
```
✓ internal/analytics/database/migrations/000002_add_enrichment_columns.up.sql
✓ internal/analytics/database/migrations/000002_add_enrichment_columns.down.sql
```

**Modified files contain expected content:**
```
✓ db/analytics_schema.sql contains country_code, device_type, traffic_source
✓ db/analytics_query.sql contains InsertEnrichedClick, CountByCountryInRange, GetClickDetails
✓ internal/analytics/usecase/click_repository.go has 7 methods
✓ internal/analytics/repository/sqlite/click_repository.go implements all 7 methods
```

**Commits exist:**
```
✓ 26231f9 (Task 1: migration + sqlc)
✓ 3f014b9 (Task 2: repository implementation)
```

**Build verification:**
```
✓ go build ./... succeeds
✓ go vet ./... succeeds
```
