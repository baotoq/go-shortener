# Phase 12: Tech Debt Cleanup - Research

**Researched:** 2026-02-22
**Domain:** Go dead code removal, test coverage, Kafka retention configuration
**Confidence:** HIGH

## Summary

Phase 12 is a pure cleanup phase with three independent requirements. No new infrastructure is introduced. DEBT-01 removes the dead `IncrementClickCount` method that predates the Kafka-based click counting architecture — it was replaced in the Phase 9 messaging migration but never deleted. DEBT-02 improves the analytics-consumer test coverage from 76.4% to >80% in the `internal/mqs` package (the only package with testable business logic). DEBT-03 adds explicit Kafka retention settings to the docker-compose.yml so the `click-events` topic does not grow unboundedly.

All three requirements are standalone and have no dependencies on each other or on any infrastructure. The work is focused on four files: `services/url-api/model/urlsmodel.go`, `services/url-api/model/mock_urlsmodel.go`, `test/integration/shorten_test.go`, `services/analytics-consumer/internal/mqs/clickeventconsumer_test.go`, and `docker-compose.yml`. A minor refactor of `services/analytics-consumer/internal/svc/servicecontext.go` (introduce `GeoIPReader` interface) enables mocking the GeoDB in tests.

**Primary recommendation:** Treat the three debts as three sequential plans — DEBT-01 (dead code removal), DEBT-02 (test coverage), DEBT-03 (Kafka retention) — each with a `go build ./...` or `go test` verification step.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| DEBT-01 | Remove dead IncrementClickCount method and unused click_count column logic | Full inventory of all call sites documented below; only 3 files need edits; generated files untouched |
| DEBT-02 | Improve analytics-consumer test coverage above 80% | Current 76.4% in mqs package; exact uncovered statements identified; GeoIPReader interface pattern enables mocking without test fixtures |
| DEBT-03 | Configure Kafka click-events topic with explicit retention settings in Docker Compose | Broker-level env vars (KAFKA_LOG_RETENTION_HOURS, KAFKA_LOG_RETENTION_BYTES) apply to auto-created topics; naming convention confirmed from existing docker-compose env var patterns |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| stdlib `testing` | Go 1.22 | Unit test framework | Built-in, no deps |
| `github.com/stretchr/testify` | already in go.mod | Assertions (`assert`, `require`) | Already used in all existing tests |
| `github.com/oschwald/geoip2-golang` | v1.13.0 | GeoIP country lookup | Already in use; introducing interface for testability |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go tool cover` | Go stdlib | Coverage analysis (`-coverprofile`, `-func`) | Verifying DEBT-02 success criterion |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| GeoIPReader interface | Real test .mmdb fixture | Interface is simpler, no binary test data committed, idiomatic Go |
| Broker-level Kafka retention | Topic-level init container script | Broker-level is simpler, covers click-events (auto-created), no additional container needed |

## Architecture Patterns

### DEBT-01: Dead Code Removal Map

The `IncrementClickCount` method was replaced by the Kafka → analytics-consumer → clicks table pipeline in Phase 9. It now has zero call sites in application code.

**Files to edit (non-generated):**

1. `/Users/baotoq/Work/go-shortener/services/url-api/model/urlsmodel.go`
   - Remove `IncrementClickCount(ctx context.Context, shortCode string) error` from `UrlsModel` interface
   - Remove `func (m *customUrlsModel) IncrementClickCount(ctx context.Context, shortCode string) error` implementation (lines 100-106)

2. `/Users/baotoq/Work/go-shortener/services/url-api/model/mock_urlsmodel.go`
   - Remove `IncrementClickCountFunc func(ctx context.Context, shortCode string) error` field
   - Remove `func (m *MockUrlsModel) IncrementClickCount(...)` method (lines 67-72)

3. `/Users/baotoq/Work/go-shortener/test/integration/shorten_test.go`
   - Remove the test block for `IncrementClickCount` (lines 78-84: `err = urlModel.IncrementClickCount(...)` and `assert.Equal(t, int64(1), updated.ClickCount)`)

**Files NOT to edit (generated, will be overwritten by goctl):**
- `services/url-api/model/urlsmodel_gen.go` — `ClickCount int64` field stays in `Urls` struct; the column still exists in the database schema and generated CRUD uses it. DEBT-01 only removes the custom `IncrementClickCount` method, not the generated field.
- `services/migrations/000001_create_urls.up.sql` — `click_count BIGINT` column stays; removing it would require a new migration and schema change, which is out of scope.

**Verification:** `go build ./...` passes.

### DEBT-02: Coverage Strategy

**Current state (confirmed by running `go test -coverprofile -func`):**

```
mqs/clickeventconsumer.go:23  NewClickEventConsumer  100.0%
mqs/clickeventconsumer.go:29  Consume                90.9%
mqs/clickeventconsumer.go:86  resolveCountry         16.7%
mqs/clickeventconsumer.go:110 resolveDeviceType      87.5%
mqs/clickeventconsumer.go:130 resolveTrafficSource   100.0%
total:                                               76.4%
```

**Total statements in mqs package: 55. Covered: 42. Need 3+ more for >80% (45/55 = 81.8%).**

**Uncovered code ranges and statement counts:**

| File:Lines | Stmts | Description | How to Cover |
|------------|-------|-------------|--------------|
| `49.16,52.3` | 2 | `uuid.NewV7()` error path in `Consume` | Cannot be triggered naturally — `uuid.NewV7()` does not fail under normal conditions. Skip. |
| `91.2,92.21` | 2 | `parsedIP := net.ParseIP(ip)` when GeoDB is non-nil | Need GeoIPReader interface for mock |
| `92.21,94.3` | 1 | `if parsedIP == nil` (GeoDB path) | Need GeoIPReader interface for mock |
| `96.2,97.16` | 2 | `svcCtx.GeoDB.Country(parsedIP)` call | Need GeoIPReader interface for mock |
| `97.16,99.3` | 1 | `if err != nil` (GeoDB.Country error path) | Need GeoIPReader interface for mock |
| `101.2,102.16` | 2 | `code := record.Country.IsoCode` | Need GeoIPReader interface for mock |
| `102.16,104.3` | 1 | `if code == ""` empty code path | Need GeoIPReader interface for mock |
| `106.2,106.13` | 1 | `return code` (happy path) | Need GeoIPReader interface for mock |
| `121.17,123.3` | 1 | `return "Mobile"` in `resolveDeviceType` | Add test with `ua.Mobile() == true` UA string |

**Approach: GeoIPReader interface + Mobile UA test**

Adding just the Mobile UA test gives 43/55 = 78.2% (still below 80%). We need the GeoDB tests too.

**Step 1: Define GeoIPReader interface in `svc` package**

```go
// services/analytics-consumer/internal/svc/servicecontext.go

// GeoIPReader abstracts the geoip2.Reader for testability.
type GeoIPReader interface {
    Country(ipAddress net.IP) (*geoip2.Country, error)
}

type ServiceContext struct {
    Config     config.Config
    ClickModel model.ClicksModel
    GeoDB      GeoIPReader  // was *geoip2.Reader
}
```

`*geoip2.Reader` naturally satisfies this interface because `geoip2.Reader.Country` has the exact signature `func (r *Reader) Country(ipAddress net.IP) (*Country, error)`. No change needed to `NewServiceContext` — assigning `*geoip2.Reader` to a `GeoIPReader` interface field works directly.

**Step 2: Add tests to `clickeventconsumer_test.go`**

```go
// Test 1: Mobile UA detection (covers line 121.17,123.3)
func TestResolveDeviceType_Mobile(t *testing.T) {
    // The mssola/useragent library detects mobile via "Mobile" substring in UA
    result := resolveDeviceType("Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.86 Mobile Safari/537.36")
    assert.Equal(t, "Mobile", result)
}
```

Note: The existing test for Android UA expects "Desktop" because the specific Android UA string used does NOT contain "Mobile". A proper mobile UA must include "Mobile" in the string to trigger `ua.Mobile()`.

```go
// Test 2: resolveCountry with mock GeoDB (covers lines 91-106)
type mockGeoIPReader struct {
    countryFunc func(ip net.IP) (*geoip2.Country, error)
}

func (m *mockGeoIPReader) Country(ip net.IP) (*geoip2.Country, error) {
    return m.countryFunc(ip)
}

func TestResolveCountry_WithGeoDB_Success(t *testing.T) {
    mock := &mockGeoIPReader{
        countryFunc: func(ip net.IP) (*geoip2.Country, error) {
            return &geoip2.Country{Country: struct{...}{IsoCode: "US", ...}}, nil
        },
    }
    svcCtx := &svc.ServiceContext{GeoDB: mock}
    result := resolveCountry(svcCtx, "8.8.8.8")
    assert.Equal(t, "US", result)
}

func TestResolveCountry_WithGeoDB_Error(t *testing.T) {
    mock := &mockGeoIPReader{
        countryFunc: func(ip net.IP) (*geoip2.Country, error) {
            return nil, errors.New("lookup failed")
        },
    }
    svcCtx := &svc.ServiceContext{GeoDB: mock}
    result := resolveCountry(svcCtx, "8.8.8.8")
    assert.Equal(t, "XX", result)
}

func TestResolveCountry_WithGeoDB_EmptyCode(t *testing.T) {
    mock := &mockGeoIPReader{
        countryFunc: func(ip net.IP) (*geoip2.Country, error) {
            return &geoip2.Country{}, nil // IsoCode empty
        },
    }
    svcCtx := &svc.ServiceContext{GeoDB: mock}
    result := resolveCountry(svcCtx, "8.8.8.8")
    assert.Equal(t, "XX", result)
}
```

These 4 new tests cover 1 (Mobile) + 10 (GeoDB paths) = 11 additional statements = 53/55 = **96.4%**.

**Important note on geoip2.Country struct construction in tests:**
The `geoip2.Country` struct has nested anonymous structs. For the success case, construct it as:
```go
country := &geoip2.Country{}
country.Country.IsoCode = "US"
```

**Verification:** `go test -cover ./services/analytics-consumer/internal/mqs/...` reports `coverage: >80% of statements`.

### DEBT-03: Kafka Topic Retention Configuration

**Approach: Broker-level defaults via environment variables**

The `apache/kafka:3.7.0` image follows the naming convention: Kafka broker property `log.retention.hours` → Docker env var `KAFKA_LOG_RETENTION_HOURS`. This is confirmed by existing env vars in the docker-compose (`KAFKA_NODE_ID` = `node.id`, `KAFKA_LISTENERS` = `listeners`).

Since `click-events` is auto-created by the url-api producer (via `KAFKA_AUTO_CREATE_TOPICS_ENABLE: "true"`), broker-level defaults apply to it automatically. No init container or `kafka-topics.sh` script is needed.

**Settings to add to the `kafka` service in `docker-compose.yml`:**

```yaml
KAFKA_LOG_RETENTION_HOURS: "24"           # Delete segments older than 24 hours
KAFKA_LOG_RETENTION_BYTES: "1073741824"   # Delete segments when partition exceeds 1GB
KAFKA_LOG_SEGMENT_BYTES: "104857600"      # Segment size 100MB (enables more granular cleanup)
```

**Retention semantics:**
- `log.retention.hours=24`: Messages older than 24 hours are eligible for deletion. This is a time-based bound.
- `log.retention.bytes=1073741824`: Per-partition size limit of 1GB. Combined with retention.hours, deletion triggers when EITHER limit is hit.
- `log.segment.bytes=104857600`: Smaller segments (100MB) mean retention cleanup runs more frequently (Kafka only deletes full segments, not individual messages within a segment).

**Default values (for reference):**
- `log.retention.hours` default: 168 (7 days) — unbounded for a dev environment
- `log.retention.bytes` default: -1 (no limit)

**Verification:** `docker compose config` shows retention settings in the kafka service definition.

### Anti-Patterns to Avoid

- **Editing `urlsmodel_gen.go`**: Never edit generated files. The `ClickCount` field in the `Urls` struct stays — it is generated code reflecting the actual DB schema. Only remove the custom `IncrementClickCount` method.
- **Removing `click_count` column from migrations**: The column exists in the DB, and the generated model still selects/inserts it. Removing the column requires a new migration + model regeneration — out of scope.
- **Trying to test `main()`**: The `consumer.go` main package cannot be unit tested. Accept 0% on that package; it is not part of the coverage criterion.
- **Using interface nil check incorrectly**: When changing `GeoDB` from `*geoip2.Reader` to `GeoIPReader` interface, ensure `NewServiceContext` returns nil (untyped nil) for GeoDB when not configured, not a typed nil pointer. Assigning `var geoDB *geoip2.Reader = nil` directly to the interface field sets it to a typed nil (which is non-nil interface). Use a conditional assignment instead.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| GeoDB mocking | Custom fake mmdb file in testdata/ | Simple `GeoIPReader` interface with struct mock | Interface is idiomatic Go; no binary test fixtures committed to git |
| Coverage measurement | Custom scripts | `go test -cover ./...` + `go tool cover -func` | Built into toolchain |
| Kafka topic creation with retention | Init container with `kafka-topics.sh --create --config retention.ms=...` | Broker-level `KAFKA_LOG_RETENTION_HOURS` env var | Simpler, fewer moving parts; auto-created topics inherit broker defaults |

**Key insight:** For DEBT-01 and DEBT-02, the existing project mock patterns (hand-rolled struct mocks with `XxxFunc` fields, as seen in `mock_urlsmodel.go` and `mock_clicksmodel.go`) should be followed exactly. Do not introduce new mocking frameworks.

## Common Pitfalls

### Pitfall 1: Broken Interface After Removing IncrementClickCount
**What goes wrong:** After removing `IncrementClickCount` from `UrlsModel` interface, `MockUrlsModel` no longer satisfies the interface if the mock method is removed correctly but the compile check `var _ UrlsModel = (*MockUrlsModel)(nil)` catches it.
**Why it happens:** The mock has the compile-time assertion. The fix is to remove the method from both the interface and the mock simultaneously.
**How to avoid:** Edit both `urlsmodel.go` (interface + implementation) and `mock_urlsmodel.go` (field + method) in the same step, then run `go build ./...`.
**Warning signs:** Compiler error "MockUrlsModel does not implement UrlsModel".

### Pitfall 2: Typed Nil Interface for GeoDB
**What goes wrong:** After introducing `GeoIPReader` interface, if `NewServiceContext` assigns a typed nil `*geoip2.Reader` to the interface field, `svcCtx.GeoDB == nil` in `resolveCountry` evaluates to `false` even when GeoDB is logically absent.
**Why it happens:** In Go, an interface value is nil only if both its type and value are nil. A `(*geoip2.Reader)(nil)` stored in a `GeoIPReader` has a non-nil type.
**How to avoid:** Use explicit conditional assignment in `NewServiceContext`:
```go
// CORRECT
var geoDB GeoIPReader
if c.GeoIPPath != "" {
    gdb, err := geoip2.Open(c.GeoIPPath)
    if err == nil {
        geoDB = gdb
    }
}
```
Not `geoDB = nil` (which assigns untyped nil to interface — fine) but also not `var geoDB *geoip2.Reader = nil` then `GeoDB: geoDB` (assigns typed nil).
**Warning signs:** `resolveCountry` panics or returns wrong result when GeoIP is not configured.

### Pitfall 3: Mobile UA Test Using Wrong User-Agent String
**What goes wrong:** The existing test for Android UA (`"Mozilla/5.0 (Linux; Android 10; SM-G973F)"`) expects "Desktop" because that string does NOT contain "Mobile". A test expecting "Mobile" must use a UA that includes the "Mobile" token.
**Why it happens:** The `mssola/useragent` library's `Mobile()` method checks for the presence of "Mobile" in the UA string.
**How to avoid:** Use a UA that includes "Mobile" in the string: `"Mozilla/5.0 (Linux; Android 10; SM-G973F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/87.0.4280.86 Mobile Safari/537.36"`.
**Warning signs:** New Mobile UA test expects "Mobile" but gets "Desktop".

### Pitfall 4: geoip2.Country Struct Field Access in Tests
**What goes wrong:** `geoip2.Country` has nested anonymous structs. You cannot use struct literal syntax like `geoip2.Country{Country: geoip2.CountryRecord{IsoCode: "US"}}` because the nested types are anonymous.
**Why it happens:** The `Country` struct in geoip2-golang v1.13.0 uses inline anonymous struct declarations (not named types).
**How to avoid:** Construct the struct then assign fields:
```go
c := &geoip2.Country{}
c.Country.IsoCode = "US"
```
**Warning signs:** Compile error "cannot use promoted field" or "unknown field".

## Code Examples

### Dead Code Removal Pattern (DEBT-01)

```go
// BEFORE: services/url-api/model/urlsmodel.go
type UrlsModel interface {
    urlsModel
    withSession(session sqlx.Session) UrlsModel
    ListWithPagination(ctx context.Context, page, pageSize int, search, sort, order string) ([]*Urls, int64, error)
    IncrementClickCount(ctx context.Context, shortCode string) error  // REMOVE
}

// AFTER:
type UrlsModel interface {
    urlsModel
    withSession(session sqlx.Session) UrlsModel
    ListWithPagination(ctx context.Context, page, pageSize int, search, sort, order string) ([]*Urls, int64, error)
    // IncrementClickCount removed — replaced by Kafka → analytics-consumer pipeline (Phase 9)
}
```

### GeoIPReader Interface (DEBT-02)

```go
// services/analytics-consumer/internal/svc/servicecontext.go

// GeoIPReader abstracts country lookup for testability.
// *geoip2.Reader naturally satisfies this interface.
type GeoIPReader interface {
    Country(ipAddress net.IP) (*geoip2.Country, error)
}

type ServiceContext struct {
    Config     config.Config
    ClickModel model.ClicksModel
    GeoDB      GeoIPReader  // changed from *geoip2.Reader
}

func NewServiceContext(c config.Config) *ServiceContext {
    // ... existing pool setup ...

    var geoDB GeoIPReader  // nil interface by default
    if c.GeoIPPath != "" {
        gdb, geoErr := geoip2.Open(c.GeoIPPath)
        if geoErr != nil {
            logx.Infof("GeoIP database not available at %s, falling back to Unknown", c.GeoIPPath)
        } else {
            geoDB = gdb  // *geoip2.Reader assigned to interface
        }
    }

    return &ServiceContext{
        Config:     c,
        ClickModel: model.NewClicksModel(conn),
        GeoDB:      geoDB,
    }
}
```

### Kafka Retention in docker-compose.yml (DEBT-03)

```yaml
kafka:
  image: apache/kafka:3.7.0
  # ... existing config ...
  environment:
    # ... existing vars ...
    KAFKA_LOG_RETENTION_HOURS: "24"
    KAFKA_LOG_RETENTION_BYTES: "1073741824"
    KAFKA_LOG_SEGMENT_BYTES: "104857600"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `IncrementClickCount` to update click_count in urls table | Kafka → analytics-consumer inserts into clicks table, analytics-rpc counts from clicks | Phase 9 | Dead code remains in codebase — this phase removes it |
| No Kafka retention (default 7 days, no size limit) | Explicit 24h/1GB retention | Phase 12 | Prevents unbounded growth in dev/prod |

## Open Questions

1. **Does the 80% coverage requirement apply to the mqs package specifically or the total analytics-consumer coverage?**
   - What we know: `go test -cover ./services/analytics-consumer/...` reports mqs at 76.4%; total (including untestable main and svc) is 47.7%.
   - What's unclear: The success criterion says "above 80% as reported by `go test -cover ./...`" — this command reports per-package, not a total.
   - Recommendation: Target the `mqs` package at >80% (the only package with testable business logic). The main and svc packages having 0% is standard Go practice for wiring/infrastructure code.

2. **Should `log.segment.bytes` be included in DEBT-03?**
   - What we know: Kafka only deletes full segments, so large segments delay effective retention. 100MB segments make retention more responsive.
   - What's unclear: The requirement only mentions "does not grow unboundedly" — time-based retention alone technically satisfies this.
   - Recommendation: Include `KAFKA_LOG_SEGMENT_BYTES: "104857600"` alongside retention settings for completeness. It improves retention effectiveness without risk.

## Sources

### Primary (HIGH confidence)
- Codebase inspection — direct `grep` and file reads confirming all IncrementClickCount call sites, current coverage at 76.4%, and geoip2.Reader struct type
- `go test -coverprofile` analysis — 55 total statements in mqs package, 42 covered, exact uncovered line ranges confirmed
- `/Users/baotoq/go/pkg/mod/github.com/oschwald/geoip2-golang@v1.13.0/reader.go` — confirmed `Country(ipAddress net.IP) (*Country, error)` signature and `Reader` is a struct
- Kafka environment variable naming convention verified from existing docker-compose.yml patterns (e.g., `KAFKA_NODE_ID` = `node.id`)

### Secondary (MEDIUM confidence)
- Apache Kafka 3.x topic config docs (https://kafka.apache.org/30/generated/topic_config.html) — confirmed `retention.ms`, `retention.bytes`, `segment.bytes` property names
- WebSearch on apache/kafka Docker KRaft mode — confirmed `KAFKA_LOG_RETENTION_HOURS` and `KAFKA_LOG_RETENTION_BYTES` env var convention

### Tertiary (LOW confidence)
- None

## Metadata

**Confidence breakdown:**
- DEBT-01 (dead code removal): HIGH — all call sites verified by grep, files confirmed
- DEBT-02 (test coverage): HIGH — exact statement counts verified, interface pattern is idiomatic Go
- DEBT-03 (Kafka retention): MEDIUM — env var names consistent with convention; recommend testing with `docker compose config` to confirm they're picked up by the apache/kafka image

**Research date:** 2026-02-22
**Valid until:** 2026-04-22 (stable domain — Go testing patterns and Kafka retention config are not fast-moving)
