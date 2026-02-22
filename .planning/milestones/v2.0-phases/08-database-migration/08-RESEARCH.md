# Phase 8: Database Migration - Research

**Researched:** 2026-02-16
**Domain:** PostgreSQL persistence with goctl-generated models
**Confidence:** HIGH

## Summary

Phase 8 replaces stub/mock data in both URL API and Analytics RPC services with real PostgreSQL persistence. The go-zero framework provides `goctl model pg datasource` to generate type-safe CRUD models from a live PostgreSQL schema. The generated code follows a two-file pattern: `*model.go` (safe to edit, add custom methods) and `*model_gen.go` (regenerated, hands off). Custom queries for pagination, search, and aggregation go in the custom model file.

The key design decisions are already locked: UUIDv7 primary keys (via `google/uuid.NewV7()`), hard deletes, no Redis cache, ILIKE search, denormalized `click_count` on the urls table, and cursor pagination using UUIDv7's natural time ordering.

**Primary recommendation:** Write DDL migration SQL manually, stand up PostgreSQL via Docker Compose, run `goctl model pg datasource` against it, then add custom query methods (pagination, search, click count increment) in the custom model files.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- Hard delete — DELETE endpoint removes the row, no soft delete/status column
- No expiration — short links live forever (or until manually deleted)
- UUIDv7 primary keys on both tables — time-ordered, natural cursor pagination, no sequence coordination
- Minimal indexes — unique constraint on short_code and primary keys only. Add more when performance demands.
- No Redis cache for now — direct PostgreSQL queries
- Keep the architecture simple with fewer moving parts
- Follow go-zero recommended approach for running PostgreSQL locally
- Cursor pagination based on UUIDv7 id (time-ordered, single field cursor)
- URL search via ILIKE prefix match on original_url — simple, no full-text search setup
- Denormalized click_count column on urls table for fast count reads

### Claude's Discretion
- Clicks table: whether to use TIMESTAMPTZ for clicked_at and whether to add enrichment columns (ip_address, user_agent, referer) now vs Phase 9
- Migration tooling choice (manual DDL, golang-migrate, or other)
- Local dev PostgreSQL setup (Docker Compose vs other)
- Whether to add analytics breakdown queries (GetClicksByCountry, GetClicksOverTime) in this phase or defer
- Index additions beyond the minimum if query patterns clearly require them

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope
</user_constraints>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/zeromicro/go-zero` | v1.10.0 | Framework (already in go.mod) | Project framework, includes `core/stores/sqlx` for DB |
| `github.com/google/uuid` | v1.6.0 | UUIDv7 generation via `uuid.NewV7()` | Already in go.mod (indirect), RFC 9562 compliant |
| `github.com/lib/pq` | latest | PostgreSQL driver for `database/sql` | Standard Go PG driver, go-zero sqlx uses it |
| `github.com/matoous/go-nanoid/v2` | v2 | NanoID 8-char short code generation | Crypto-safe, URL-friendly, simple API |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/golang-migrate/migrate/v4` | v4 | Schema migration versioning | Run DDL up/down migrations |
| Docker (`postgres:16-alpine`) | 16 | Local dev PostgreSQL | Docker Compose for local dev |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `golang-migrate` | Manual DDL scripts | Manual is simpler for 2 tables but no rollback support; golang-migrate adds versioned migrations with up/down |
| `lib/pq` | `pgx` | pgx is more modern but go-zero sqlx works with `database/sql` interface; lib/pq is the path of least resistance |

**Recommendation (Claude's discretion: migration tooling):** Use `golang-migrate` with file-based SQL migrations. It provides versioned up/down migrations, integrates well with CI, and is the most popular Go migration tool (3,299 importers). For only 2 tables this is modest overhead but establishes good practice.

**Recommendation (Claude's discretion: local dev PostgreSQL):** Docker Compose with `postgres:16-alpine`. Simple, reproducible, matches what Phase 10 will expand for full stack orchestration.

## Architecture Patterns

### goctl Model Two-File Pattern
```
services/url-api/model/
├── urlsmodel.go          # Custom model — ADD methods here (safe to edit)
├── urlsmodel_gen.go      # Generated CRUD — NEVER edit (regenerated)
└── vars.go               # Error variables
```

**How it works:**
- `goctl model pg datasource` generates `*model_gen.go` with basic CRUD: `Insert`, `FindOne`, `Update`, `Delete`, `FindOneByShortCode` (for unique indexes)
- `*model.go` contains `customUrlsModel` struct that embeds `defaultUrlsModel`
- Custom queries (pagination, search, click count) go in `*model.go`
- Re-running goctl overwrites `*model_gen.go` only; your custom methods survive

### Generated Model Interface Pattern
```go
// Generated in urlsmodel_gen.go
type (
  urlsModel interface {
    Insert(ctx context.Context, data *Urls) (sql.Result, error)
    FindOne(ctx context.Context, id string) (*Urls, error)
    FindOneByShortCode(ctx context.Context, shortCode string) (*Urls, error)
    Update(ctx context.Context, data *Urls) error
    Delete(ctx context.Context, id string) error
  }

  defaultUrlsModel struct {
    conn  sqlx.SqlConn
    table string
  }
)
```

```go
// In urlsmodel.go (safe to edit)
type (
  UrlsModel interface {
    urlsModel // embed generated interface
    // Custom methods
    FindByShortCode(ctx context.Context, code string) (*Urls, error)
    ListWithPagination(ctx context.Context, cursor string, limit int, search string) ([]*Urls, error)
    IncrementClickCount(ctx context.Context, shortCode string) error
  }

  customUrlsModel struct {
    *defaultUrlsModel
  }
)
```

### ServiceContext Dependency Injection
```go
// services/url-api/internal/svc/servicecontext.go
type ServiceContext struct {
  Config    config.Config
  UrlModel  model.UrlsModel    // Add in Phase 8
}

func NewServiceContext(c config.Config) *ServiceContext {
  conn := sqlx.NewSqlConn("postgres", c.DataSource)
  return &ServiceContext{
    Config:   c,
    UrlModel: model.NewUrlsModel(conn),
  }
}
```

### Config Extension for PostgreSQL
```go
// services/url-api/internal/config/config.go
type Config struct {
  rest.RestConf
  BaseUrl    string
  DataSource string  // PostgreSQL connection string
}
```

```yaml
# services/url-api/etc/url.yaml
DataSource: postgres://postgres:postgres@localhost:5432/shortener?sslmode=disable
```

### Project Structure Addition
```
services/
├── url-api/
│   ├── model/                    # NEW: goctl-generated models
│   │   ├── urlsmodel.go         # Custom methods (pagination, search)
│   │   ├── urlsmodel_gen.go     # Generated CRUD
│   │   └── vars.go              # Error variables
│   └── internal/
│       ├── config/config.go     # Add DataSource field
│       ├── svc/servicecontext.go # Add UrlModel
│       └── logic/               # Replace stubs with DB calls
├── analytics-rpc/
│   ├── model/                    # NEW: goctl-generated models
│   │   ├── clicksmodel.go      # Custom methods (count, aggregate)
│   │   ├── clicksmodel_gen.go  # Generated CRUD
│   │   └── vars.go
│   └── internal/
│       ├── config/config.go     # Add DataSource field
│       ├── svc/servicecontext.go # Add ClickModel
│       └── logic/               # Replace stubs with DB calls
└── migrations/                   # NEW: SQL migration files
    ├── 000001_create_urls.up.sql
    ├── 000001_create_urls.down.sql
    ├── 000002_create_clicks.up.sql
    └── 000002_create_clicks.down.sql
```

### Anti-Patterns to Avoid
- **Editing `*_gen.go` files:** They get overwritten by goctl. Always add methods in the custom model file.
- **Putting SQL in logic layer:** All database access goes through the model layer. Logic calls model methods only.
- **Using `sql.NullString` everywhere:** go-zero generates proper types from the schema. Use NOT NULL columns where possible to avoid nullable types.
- **Skipping connection pooling:** go-zero's `sqlx.NewSqlConn` handles pooling, but you still need to set `MaxOpenConns`/`MaxIdleConns` in the PostgreSQL config.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Database CRUD | Manual SQL for basic ops | `goctl model pg datasource` | Generated, type-safe, consistent |
| Connection pooling | Manual pool management | go-zero `sqlx.NewSqlConn` | Built-in pool with proper lifecycle |
| Schema migrations | Ad-hoc SQL scripts | `golang-migrate` | Versioned, reversible, CI-friendly |
| Short ID generation | Custom random string | `go-nanoid/v2` (8-char) | Crypto-safe, URL-friendly, collision resistant |
| UUIDv7 generation | Custom timestamp+random | `google/uuid.NewV7()` | RFC 9562 compliant, monotonic, already in go.mod |

**Key insight:** goctl model generates 80% of the data layer. Only custom queries need manual implementation.

## Common Pitfalls

### Pitfall 1: UUIDv7 Storage as TEXT vs UUID Type
**What goes wrong:** Storing UUIDs as TEXT wastes space (36 bytes vs 16 bytes) and loses index performance.
**Why it happens:** goctl model may generate `string` type for UUID columns if the DDL uses `TEXT`.
**How to avoid:** Use `UUID` column type in PostgreSQL DDL. goctl maps this to `string` in Go, but PostgreSQL stores it efficiently as 16 bytes internally. The DDL must use `UUID` not `VARCHAR` or `TEXT`.
**Warning signs:** Schema DDL using `VARCHAR(36)` for ID columns.

### Pitfall 2: Missing Unique Index on short_code
**What goes wrong:** goctl only generates `FindOneByXxx` methods for columns with unique indexes. Without a unique index on `short_code`, you won't get `FindOneByShortCode`.
**Why it happens:** goctl inspects actual database constraints to decide what to generate.
**How to avoid:** DDL must include `UNIQUE` constraint on `short_code` column. Verify after generation that the method exists.
**Warning signs:** No `FindOneByShortCode` in generated code.

### Pitfall 3: Cursor Pagination Off-by-One
**What goes wrong:** Results include the cursor item itself, or skip items with the same timestamp.
**Why it happens:** Using `>=` instead of `>` in cursor comparison, or UUIDv7 millisecond collisions.
**How to avoid:** Use strict `>` comparison with the cursor ID. UUIDv7 includes a monotonic counter within the same millisecond, so ordering by ID alone is sufficient.
**Warning signs:** Duplicate items across pages, or missing items.

### Pitfall 4: click_count Race Condition
**What goes wrong:** Concurrent click events can lose increments if using read-modify-write pattern.
**Why it happens:** Two requests read count=5, both write count=6.
**How to avoid:** Use atomic SQL: `UPDATE urls SET click_count = click_count + 1 WHERE short_code = $1`. Never read-then-write.
**Warning signs:** Click counts lower than expected under load.

### Pitfall 5: goctl Model Regeneration Overwrites Custom Code
**What goes wrong:** Running goctl model regeneration wipes custom methods.
**Why it happens:** Confusion about which files are safe to edit.
**How to avoid:** Only `*model_gen.go` gets overwritten. Custom methods go in `*model.go`. Document this in Makefile comments.
**Warning signs:** Lost custom methods after running `make gen-models`.

## Code Examples

### UUIDv7 Generation for Primary Keys
```go
import "github.com/google/uuid"

func NewUUIDv7() string {
  id, err := uuid.NewV7()
  if err != nil {
    // Fallback to V4 if V7 fails (extremely unlikely)
    return uuid.New().String()
  }
  return id.String()
}
```

### NanoID Short Code Generation with Collision Retry
```go
import gonanoid "github.com/matoous/go-nanoid/v2"

const (
  shortCodeLength = 8
  maxRetries      = 5
)

// Alphabet: URL-safe characters (no look-alikes)
const alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

func GenerateShortCode() (string, error) {
  return gonanoid.Generate(alphabet, shortCodeLength)
}
```

### Cursor Pagination Query (Custom Model Method)
```go
func (m *customUrlsModel) ListWithPagination(ctx context.Context, cursor string, limit int, search string) ([]*Urls, error) {
  var resp []*Urls
  var err error

  query := fmt.Sprintf("SELECT %s FROM %s", urlsRows, m.table)

  conditions := make([]string, 0)
  args := make([]interface{}, 0)
  argIdx := 1

  if cursor != "" {
    conditions = append(conditions, fmt.Sprintf("id < $%d", argIdx))
    args = append(args, cursor)
    argIdx++
  }

  if search != "" {
    conditions = append(conditions, fmt.Sprintf("original_url ILIKE $%d", argIdx))
    args = append(args, search+"%")
    argIdx++
  }

  if len(conditions) > 0 {
    query += " WHERE " + strings.Join(conditions, " AND ")
  }

  query += fmt.Sprintf(" ORDER BY id DESC LIMIT $%d", argIdx)
  args = append(args, limit)

  err = m.conn.QueryRowsCtx(ctx, &resp, query, args...)
  return resp, err
}
```

### Atomic Click Count Increment
```go
func (m *customUrlsModel) IncrementClickCount(ctx context.Context, shortCode string) error {
  query := fmt.Sprintf("UPDATE %s SET click_count = click_count + 1 WHERE short_code = $1", m.table)
  _, err := m.conn.ExecCtx(ctx, query, shortCode)
  return err
}
```

### PostgreSQL DDL for urls Table
```sql
CREATE TABLE urls (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  short_code  VARCHAR(8) NOT NULL UNIQUE,
  original_url TEXT NOT NULL,
  click_count  BIGINT NOT NULL DEFAULT 0,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### PostgreSQL DDL for clicks Table
```sql
CREATE TABLE clicks (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  short_code  VARCHAR(8) NOT NULL,
  clicked_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

**Recommendation (Claude's discretion: clicks enrichment columns):** Defer enrichment columns (ip_address, user_agent, referer) to Phase 9 when the Kafka consumer is wired. Phase 8 focuses on basic persistence. The clicks table in Phase 8 only needs `id`, `short_code`, and `clicked_at`. Adding columns now would mean they're always NULL until Phase 9 enrichment exists.

**Recommendation (Claude's discretion: analytics breakdown queries):** Defer GetClicksByCountry and GetClicksOverTime to Phase 9. These require enrichment data (country, timestamps with meaningful aggregation) that won't exist until the Kafka consumer processes events. Phase 8 provides only GetClickCount via simple COUNT or the denormalized column.

**Recommendation (Claude's discretion: TIMESTAMPTZ):** Use TIMESTAMPTZ for `clicked_at` and `created_at`. It's PostgreSQL best practice and avoids timezone bugs.

**Recommendation (Claude's discretion: additional indexes):** Add an index on `clicks.short_code` since GetClickCount queries by it. This is the only additional index beyond the locked minimum (PK + unique short_code on urls).

### goctl Model Generation Command
```bash
# After PostgreSQL is running with tables created:
goctl model pg datasource \
  --url "postgres://postgres:postgres@localhost:5432/shortener?sslmode=disable" \
  --table "urls" \
  --dir services/url-api/model \
  --style gozero

goctl model pg datasource \
  --url "postgres://postgres:postgres@localhost:5432/shortener?sslmode=disable" \
  --table "clicks" \
  --dir services/analytics-rpc/model \
  --style gozero
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| UUIDv4 random | UUIDv7 time-ordered | RFC 9562 (2024) | Better index locality, natural sort order |
| `lib/pq` only | `pgx` gaining share | 2023+ | `lib/pq` still works fine with go-zero sqlx |
| Manual SQL models | goctl model generation | go-zero 1.x | 80% less boilerplate for CRUD |

## Open Questions

1. **goctl model pg with UUID column type**
   - What we know: goctl maps PostgreSQL types to Go types; UUID maps to `string`
   - What's unclear: Whether `DEFAULT gen_random_uuid()` in DDL affects generated Insert method (does it omit the ID field?)
   - Recommendation: Test generation against actual schema. If goctl includes ID in Insert, set it explicitly with `uuid.NewV7()` in logic layer before calling Insert.

2. **go-zero sqlx connection string format**
   - What we know: Config uses `DataSource` field with PostgreSQL connection string
   - What's unclear: Exact format go-zero expects (postgres:// vs postgresql://)
   - Recommendation: Use `postgres://user:pass@host:port/db?sslmode=disable` format, test on startup.

## Sources

### Primary (HIGH confidence)
- Context7 `/websites/go-zero_dev` - goctl model pg datasource usage, type mappings, flags
- https://go-zero.dev/docs/tutorials/cli/model - Official goctl model documentation
- https://pkg.go.dev/github.com/google/uuid - google/uuid NewV7 API

### Secondary (MEDIUM confidence)
- https://github.com/golang-migrate/migrate - golang-migrate GitHub (3,299 importers, actively maintained)
- https://github.com/matoous/go-nanoid - go-nanoid v2 for short code generation
- https://zeromicro.github.io/go-zero.dev/en/docs/build-tool/model/ - go-zero model two-file pattern

### Tertiary (LOW confidence)
- goctl model pg behavior with UUID DEFAULT columns (needs validation by running goctl)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - go-zero sqlx + goctl model pg well-documented, google/uuid NewV7 verified
- Architecture: HIGH - Two-file model pattern confirmed in official docs, ServiceContext DI pattern already in codebase
- Pitfalls: HIGH - Race condition, cursor pagination, and regeneration issues are well-documented patterns
- Custom queries: MEDIUM - Code examples follow documented patterns but exact goctl output needs testing

**Research date:** 2026-02-16
**Valid until:** 2026-03-16 (stable domain, go-zero v1.10.0 locked)
