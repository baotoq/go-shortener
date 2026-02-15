# Phase 7: Framework Foundation - Context

**Gathered:** 2026-02-15
**Status:** Ready for planning

<domain>
## Phase Boundary

Establish go-zero code generation patterns and service structure. Define `.api` and `.proto` specs, generate go-zero scaffolding (Handler/Logic/Types), set up ServiceContext, request validation, error handling, config loading, and structured logging. No real database or messaging — services start with stub responses.

</domain>

<decisions>
## Implementation Decisions

### Project layout
- Replace in-place: rewrite `internal/urlservice` and `internal/analytics` directly into go-zero generated structure. Old Chi/Dapr code is deleted.
- Services start with stubs and respond to requests — the skeleton must be bootable, not just compilable.

### Module strategy (Claude's Discretion)
- Single vs multi-module: Claude decides based on go-zero conventions and project size.
- Shared code location (pkg/, shared/, common/): Claude decides based on go-zero community patterns.
- Generated output directory structure: Claude decides the most conventional go-zero layout.

### API spec design
- Route grouping strategy: Claude decides based on go-zero patterns and existing API surface.
- API path style: Claude decides whether to keep same paths or clean up (no external consumers to break).
- Proto scope for Analytics: Claude decides minimal vs full RPCs based on service boundary needs.
- **Strict validation**: Validate all fields, reject unknown fields, enforce formats. Fail early on bad input.

### Migration approach
- **OK to break old code**: Phase 7 replaces the structure. Old Chi/Dapr code can be deleted.
- **Remove old dependencies immediately**: Chi, Dapr SDK, go-sqlite3 removed from go.mod in Phase 7. Clean break.
- **Services must start with stubs**: Routes respond with TODO/stub data proving the skeleton works end-to-end.
- **Discard old tests**: Remove v1.0 test files. New tests written from scratch aligned with go-zero patterns in Phase 10.

### Error & logging style
- Error format: Claude decides whether to keep RFC 7807 Problem Details or simplify, based on what's practical with go-zero's error handling.
- **Per-field validation errors**: Validation failures return array of `{field, message}` pairs, not a single combined message.
- Log format: Claude decides (JSON vs plain text per environment) based on go-zero defaults and dev experience.
- **Enable built-in request logging**: go-zero logs every request with method, path, status, latency.

### Claude's Discretion
- Single module vs multi-module decision
- Shared code directory naming and location
- Generated output directory placement (internal/ vs services/ vs other)
- Route grouping and path naming in .api file
- Proto RPC scope (minimal vs full analytics)
- Error response format (RFC 7807 vs simplified)
- Log format strategy (JSON/plain text per environment)

</decisions>

<specifics>
## Specific Ideas

- This is a learning project — go-zero patterns should be idiomatic and conventional, not clever
- Clean break philosophy: remove old code and dependencies immediately, don't carry debt forward
- Services must be bootable with stubs so each subsequent phase (DB, messaging) can plug in incrementally

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 07-framework-foundation*
*Context gathered: 2026-02-15*
