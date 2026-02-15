---
phase: 07-framework-foundation
plan: 02
subsystem: url-api
tags: [go-zero, api-generation, rest, validation, error-handling]
dependencies:
  requires:
    - 07-01 (Framework Scaffold)
  provides:
    - URL API service with complete route definitions
    - Request validation via .api tags
    - RFC 7807 error handling
    - Stub endpoints for all operations
  affects:
    - services/url-api/* (complete service structure)
tech-stack:
  added:
    - go-zero REST framework
    - goctl API code generation
    - httpx error handling
  patterns:
    - .api specification as single source of truth
    - Generated handlers + logic separation
    - Context-aware error handlers
key-files:
  created:
    - services/url-api/url.api (API specification)
    - services/url-api/url.go (main entry point)
    - services/url-api/etc/url.yaml (service configuration)
    - services/url-api/internal/types/types.go (generated types)
    - services/url-api/internal/handler/* (5 generated handlers)
    - services/url-api/internal/logic/* (5 logic implementations)
  modified:
    - Makefile (added gen-url, run-url targets)
    - go.mod/go.sum (go-zero dependencies)
decisions:
  - choice: "Use httpx.SetErrorHandlerCtx for RFC 7807 errors"
    rationale: "Context-aware variant preferred over SetErrorHandler in go-zero"
    impact: "All errors return Problem Details format globally"
  - choice: "Stub logic returns mock data instead of errors"
    rationale: "Proves full request/response pipeline works end-to-end"
    impact: "Service is bootable and testable without database"
  - choice: "Redirect returns error in Phase 7, HTTP redirect in Phase 8"
    rationale: "HTTP redirect requires database lookup; error path proves routing works"
    impact: "Redirect handler will be customized in Phase 8 for actual redirects"
metrics:
  duration: 226s
  completed: 2026-02-16
  tasks: 2
  commits: 2
  files_created: 19
  files_modified: 10
---

# Phase 7 Plan 2: URL API Service Summary

**One-liner:** go-zero REST API with 5 routes, request validation, RFC 7807 error handling, and stub logic

## Objective Achieved

Created a fully bootable URL API service using go-zero framework with complete API specification, code generation, request validation, custom error handling, and stub implementations for all endpoints.

## What Was Built

### 1. API Specification (url.api)
- **5 routes across 3 groups:**
  - Shorten: `POST /api/v1/urls` (create short URL)
  - Redirect: `GET /:code` (redirect to original URL)
  - Links: `GET /api/v1/links` (list with pagination)
  - Links: `GET /api/v1/links/:code` (get link details)
  - Links: `DELETE /api/v1/links/:code` (delete link)

- **Request validation via tags:**
  - `range=[1:]` for page (must be >= 1)
  - `range=[1:100]` for per_page (1-100)
  - `options=created_at|original_url` for sort field
  - `options=asc|desc` for order direction
  - `default=1`, `default=20`, `default=created_at`, `default=desc`
  - `optional` for search parameter

### 2. Generated Code Structure
```
services/url-api/
├── url.api                    # API specification
├── url.go                     # Main entry point
├── etc/url.yaml               # Configuration
└── internal/
    ├── config/config.go       # Config struct
    ├── svc/servicecontext.go  # Service context
    ├── types/types.go         # Request/response types
    ├── handler/
    │   ├── shorten/shortenhandler.go
    │   ├── redirect/redirecthandler.go
    │   ├── links/listlinkshandler.go
    │   ├── links/getlinkdetailhandler.go
    │   ├── links/deletelinkhandler.go
    │   └── routes.go          # Route registration
    └── logic/
        ├── shorten/shortenlogic.go
        ├── redirect/redirectlogic.go
        └── links/
            ├── listlinkslogic.go
            ├── getlinkdetaillogic.go
            └── deletelinklogic.go
```

### 3. Custom Error Handling
- **RFC 7807 Problem Details:** All errors return JSON in Problem Details format
- **Validation errors:** go-zero validation mapped to 400 responses
- **Error handler:** `httpx.SetErrorHandlerCtx` provides context-aware error transformation
- **Example validation error:**
  ```json
  {
    "type": "https://api.example.com/problems/validation-error",
    "title": "Bad Request",
    "status": 400,
    "detail": "value \"invalid\" for field \"sort\" is not defined in options \"[created_at original_url]\""
  }
  ```

### 4. Configuration
**url.yaml:**
- Service: url-api on port 8080
- Timeout: 30000ms
- MaxConns: 10000
- Logging: console mode, info level
- BaseUrl: http://localhost:8080 (for constructing short URLs)

**config.go:**
- Embeds `rest.RestConf` for go-zero REST server config
- Adds `BaseUrl` field for short URL generation

### 5. Stub Logic Implementations
All endpoints return mock data to prove the framework skeleton works:

**Shorten:**
```go
return &types.ShortenResponse{
    ShortCode:   "stub0001",
    ShortUrl:    l.svcCtx.Config.BaseUrl + "/stub0001",
    OriginalUrl: req.OriginalUrl,
}, nil
```

**List Links:**
```go
return &types.LinkListResponse{
    Links: []types.LinkItem{
        {ShortCode: "stub0001", OriginalUrl: "https://example.com", CreatedAt: 1700000000},
        {ShortCode: "stub0002", OriginalUrl: "https://go-zero.dev", CreatedAt: 1700000001},
    },
    Page:       req.Page,
    PerPage:    req.PerPage,
    TotalPages: 1,
    TotalCount: 2,
}, nil
```

**Redirect:**
```go
return fmt.Errorf("short code '%s' not found (stub - DB not connected)", req.Code)
```
(Returns error to prove routing and error handling work; Phase 8 will add actual HTTP redirect)

**Get Link Detail:**
```go
return &types.LinkDetailResponse{
    ShortCode:   req.Code,
    OriginalUrl: "https://example.com/stub",
    CreatedAt:   1700000000,
    TotalClicks: 42,
}, nil
```

**Delete Link:**
```go
return nil  // Success stub
```

### 6. Built-in Request Logging
go-zero provides structured logging automatically:
```json
{
  "@timestamp": "2026-02-16T00:39:50.234+07:00",
  "caller": "handler/loghandler.go:167",
  "content": "[HTTP] 200 - POST /api/v1/urls - [::1]:53016 - curl/8.7.1",
  "duration": "0.3ms",
  "level": "info",
  "span": "f7784f833ee55fe8",
  "trace": "52c5a4d9ceb95cf3dcac3c6ddb15205e"
}
```

Each request logs:
- HTTP status code
- Method and path
- Client IP and user agent
- Duration
- Trace and span IDs (for distributed tracing)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Missing go-zero dependencies**
- **Found during:** Task 1 after running goctl
- **Issue:** go.sum missing entries for go-zero transitive dependencies (fatih/color, prometheus, opentelemetry, etc.)
- **Fix:** Ran `go mod tidy` to download and add all missing dependencies
- **Files modified:** go.mod, go.sum
- **Commit:** b9eac7d (included in Task 1)

No other deviations. Plan executed as written.

## Verification Results

All verification steps passed:

1. **Compilation:** `go build ./services/url-api/...` succeeds
2. **Service starts:** Boots on port 8080 with config loaded
3. **POST /api/v1/urls:** Returns stub ShortenResponse with short_code, short_url, original_url
4. **GET /api/v1/links:** Returns stub LinkListResponse with paginated links
5. **GET /api/v1/links?page=0:** Returns 400 validation error (range violation)
6. **GET /api/v1/links?sort=invalid:** Returns 400 validation error (options violation)
7. **GET /:code:** Returns 400 error in Problem Details format (stub error)
8. **GET /api/v1/links/:code:** Returns stub LinkDetailResponse
9. **DELETE /api/v1/links/:code:** Returns 200 success
10. **Request logging:** Console logs show method, path, status, duration, trace IDs

## Integration Points

### With 07-01 (Framework Scaffold)
- Uses `pkg/problemdetails` package created in 07-01
- Follows go-zero project structure established in 07-01
- Integrates with shared events (not used yet, but structure in place)

### For Future Plans
**Phase 8 (Database Migration) will add:**
- PostgreSQL connection in ServiceContext
- go-zero sqlx model for URL table
- Replace stub logic with actual database operations
- Customize redirect handler to do HTTP 302 redirect

**Phase 9 (Messaging Migration) will add:**
- Kafka producer in ServiceContext
- Publish ClickEvent after redirect
- Remove stub responses with real data

## Success Criteria Met

- [x] url.api defines 5 routes across 3 groups (shorten, redirect, links)
- [x] goctl generates all handlers, logic, types, routes without errors
- [x] Request validation works: page range, sort options, order options enforced
- [x] RFC 7807 Problem Details returned for all errors via httpx.SetErrorHandlerCtx
- [x] Config loaded from YAML (port 8080, BaseUrl, log settings)
- [x] ServiceContext initialized with config only (Phase 7 stub)
- [x] All logic functions return stub/TODO responses
- [x] Service boots and responds to curl requests on all endpoints
- [x] Built-in request logging enabled (console mode)

## Key Takeaways

1. **go-zero code generation is reliable:** goctl produces clean, organized code from .api specs
2. **Validation tags are powerful:** Built-in validation eliminates boilerplate validation code
3. **Generated code is customization-friendly:** Handlers are generated, logic is editable
4. **Error handling is global:** One SetErrorHandlerCtx call applies to all endpoints
5. **Stub logic proves the skeleton:** Mock responses validate the entire framework stack works

## Next Steps

**Immediate (07-03):** Create Analytics RPC service
**After Phase 7:** Implement database layer in Phase 8

## Self-Check: PASSED

**Created files:**
```bash
$ ls services/url-api/url.api
services/url-api/url.api

$ ls services/url-api/url.go
services/url-api/url.go

$ ls services/url-api/internal/types/types.go
services/url-api/internal/types/types.go

$ ls services/url-api/internal/handler/routes.go
services/url-api/internal/handler/routes.go

$ ls services/url-api/internal/logic/shorten/
shortenlogic.go

$ ls services/url-api/internal/logic/redirect/
redirectlogic.go

$ ls services/url-api/internal/logic/links/
deletelinklogic.go  getlinkdetaillogic.go  listlinkslogic.go
```

**Commits exist:**
```bash
$ git log --oneline --all | grep b9eac7d
b9eac7d feat(07-02): generate URL API service from .api spec

$ git log --oneline --all | grep cae5c84
cae5c84 feat(07-02): customize URL API with error handler and stub logic
```

All claimed files and commits verified successfully.
