# Phase 7: Framework Foundation - Research

**Researched:** 2026-02-16
**Domain:** go-zero microservices framework migration
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Project layout
- Replace in-place: rewrite `internal/urlservice` and `internal/analytics` directly into go-zero generated structure. Old Chi/Dapr code is deleted.
- Services start with stubs and respond to requests — the skeleton must be bootable, not just compilable.

#### API spec design
- **Strict validation**: Validate all fields, reject unknown fields, enforce formats. Fail early on bad input.

#### Migration approach
- **OK to break old code**: Phase 7 replaces the structure. Old Chi/Dapr code can be deleted.
- **Remove old dependencies immediately**: Chi, Dapr SDK, go-sqlite3 removed from go.mod in Phase 7. Clean break.
- **Services must start with stubs**: Routes respond with TODO/stub data proving the skeleton works end-to-end.
- **Discard old tests**: Remove v1.0 test files. New tests written from scratch aligned with go-zero patterns in Phase 10.

#### Error & logging style
- **Per-field validation errors**: Validation failures return array of `{field, message}` pairs, not a single combined message.
- **Enable built-in request logging**: go-zero logs every request with method, path, status, latency.

### Claude's Discretion

The following areas require research and recommendation:

1. **Module strategy**: Single vs multi-module decision based on go-zero conventions and project size
2. **Shared code location**: pkg/, shared/, common/ based on go-zero community patterns
3. **Generated output directory structure**: internal/ vs services/ vs other conventional layouts
4. **Route grouping strategy**: Based on go-zero patterns and existing API surface
5. **API path style**: Whether to keep same paths or clean up (no external consumers)
6. **Proto scope for Analytics**: Minimal vs full RPCs based on service boundary needs
7. **Error response format**: Keep RFC 7807 Problem Details or simplify based on go-zero error handling
8. **Log format strategy**: JSON vs plain text per environment based on go-zero defaults

### Specific Project Philosophy

- This is a learning project — go-zero patterns should be idiomatic and conventional, not clever
- Clean break philosophy: remove old code and dependencies immediately, don't carry debt forward
- Services must be bootable with stubs so each subsequent phase (DB, messaging) can plug in incrementally

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope

</user_constraints>

## Summary

go-zero is a production-grade cloud-native Go microservices framework with built-in code generation (goctl), API specification language (.api files), and RPC support (zRPC/gRPC). The framework enforces clean separation via Handler/Logic/ServiceContext pattern and provides batteries-included features like request validation, middleware, configuration management, and structured logging.

**Migration strategy:** Single module monorepo with services in `services/` directory. Use goctl to generate conventional structure from .api and .proto specs. Preserve RFC 7807 Problem Details via custom error handler. Services boot with stub logic to prove framework integration before adding database/messaging in later phases.

**Primary recommendation:** Use goctl's default conventions with minimal customization. Single module for simplicity, services/ directory for clarity, group routes by domain in .api files, and enable go-zero's built-in validation/logging features unchanged.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| zeromicro/go-zero | v1.7.3+ | Web & RPC framework | Official framework, comprehensive microservice support |
| goctl | latest | Code generator | go-zero's official CLI, generates handlers/logic/types from specs |
| google.golang.org/protobuf | v1.36+ | Protocol buffers | Required for zRPC service definitions |
| google.golang.org/grpc | v1.74+ | gRPC runtime | Required for zRPC (go-zero wraps gRPC) |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| go-zero/core/logx | (built-in) | Structured logging | Default logger, replaces Zap |
| go-zero/rest/httpx | (built-in) | HTTP helpers | Custom error handlers, response writers |
| go-zero/core/conf | (built-in) | YAML config loading | Configuration management |
| go-zero/core/mapping | (built-in) | Request unmarshaling | Automatic request validation |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-zero .api | OpenAPI/Swagger | go-zero .api is simpler, tighter integration with goctl |
| zRPC | Connect/gRPC-Gateway | zRPC provides go-zero conventions and built-in service discovery |
| logx | Zap/Zerolog | logx integrates seamlessly with rest.Server, minimal config |
| httpx error handler | Custom middleware | httpx.SetErrorHandler is go-zero's idiomatic pattern |

**Installation:**

```bash
# Install go-zero framework
go get -u github.com/zeromicro/go-zero@latest

# Install goctl CLI (required for code generation)
go install github.com/zeromicro/go-zero/tools/goctl@latest

# Verify installation
goctl --version

# Install protoc plugins for zRPC (if not already installed)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Architecture Patterns

### Recommended Project Structure (Single Module Monorepo)

```
go-shortener/                    # Single module root
├── go.mod                       # Single go.mod for entire project
├── go.sum
├── services/                    # All services under services/
│   ├── url-api/                # URL API service
│   │   ├── url.api             # API specification
│   │   ├── url.go              # Main entry point
│   │   ├── etc/
│   │   │   └── url-api.yaml    # Service configuration
│   │   └── internal/           # Generated + manual code
│   │       ├── config/
│   │       │   └── config.go   # Config struct (generated)
│   │       ├── handler/        # HTTP handlers (generated)
│   │       │   ├── routes.go
│   │       │   ├── shortenhandler.go
│   │       │   └── redirecthandler.go
│   │       ├── logic/          # Business logic (generated stubs, manual impl)
│   │       │   ├── shortenlogic.go
│   │       │   └── redirectlogic.go
│   │       ├── svc/
│   │       │   └── servicecontext.go  # Dependency injection
│   │       └── types/
│   │           └── types.go    # Request/response types (generated)
│   │
│   ├── analytics-rpc/          # Analytics RPC service
│   │   ├── analytics.proto     # Proto specification
│   │   ├── analytics.go        # Main entry point
│   │   ├── etc/
│   │   │   └── analytics-rpc.yaml
│   │   ├── analytics/          # Generated proto code
│   │   │   ├── analytics.pb.go
│   │   │   └── analytics_grpc.pb.go
│   │   ├── client/             # Generated RPC client
│   │   │   └── analyticsclient/
│   │   │       └── analyticsclient.go
│   │   └── internal/
│   │       ├── config/
│   │       │   └── config.go
│   │       ├── logic/          # RPC logic implementation
│   │       │   └── getclickcountlogic.go
│   │       ├── server/         # gRPC server registration
│   │       │   └── analyticsserver.go
│   │       └── svc/
│   │           └── servicecontext.go
│   │
│   └── analytics-consumer/     # Analytics Kafka consumer (Phase 9)
│       └── (deferred to Phase 9)
│
├── pkg/                        # Shared library code (project-specific)
│   └── problemdetails/         # RFC 7807 helpers (preserve existing)
│       └── problemdetails.go
│
└── common/                     # Shared types/events (cross-service)
    └── events/                 # Event definitions (migrate from internal/shared)
        └── events.go
```

**Rationale:**
- **Single module**: Go community best practice for small-to-medium projects. Simplifies testing, dependency management, and versioning.
- **services/ directory**: Clear separation from pkg/ (libraries) and common/ (shared types).
- **internal/ per service**: go-zero convention. Generated code stays isolated per service.
- **pkg/ for utilities**: RFC 7807 problem details is project-specific library code, not a service.
- **common/ for shared domain**: Event types cross service boundaries but aren't utilities.

### Pattern 1: API Service Specification (.api file)

**What:** go-zero's DSL for defining HTTP APIs with request/response types, routes, validation rules, and service groups.

**When to use:** All HTTP/REST services. Replaces manual handler wiring and request validation.

**Example (URL Service .api):**

```goapi
// Source: go-zero documentation + existing API surface
syntax = "v1"

info (
    title: "URL Shortener API"
    desc: "URL shortening and redirect service"
    author: "go-shortener"
    version: "v2.0"
)

// ========== Types ==========

type ShortenRequest {
    OriginalUrl string `json:"original_url"` // Required field
}

type ShortenResponse {
    ShortCode   string `json:"short_code"`
    ShortUrl    string `json:"short_url"`
    OriginalUrl string `json:"original_url"`
}

type LinkListRequest {
    Page      int    `form:"page,default=1,range=[1:]"`           // Min 1
    PerPage   int    `form:"per_page,default=20,range=[1:100]"`  // 1-100
    Sort      string `form:"sort,default=created_at,options=created_at|original_url"`
    Order     string `form:"order,default=desc,options=asc|desc"`
    Search    string `form:"search,optional"`                     // Optional
}

type LinkListResponse {
    Links      []LinkItem `json:"links"`
    Page       int        `json:"page"`
    PerPage    int        `json:"per_page"`
    TotalPages int        `json:"total_pages"`
    TotalCount int64      `json:"total_count"`
}

type LinkItem {
    ShortCode   string `json:"short_code"`
    OriginalUrl string `json:"original_url"`
    CreatedAt   int64  `json:"created_at"`
}

type ErrorResponse {
    Type   string `json:"type"`
    Title  string `json:"title"`
    Status int    `json:"status"`
    Detail string `json:"detail"`
}

// ========== Service Groups ==========

@server (
    prefix: /api/v1
    group: shorten
)
service url {
    @doc "Create short URL"
    @handler Shorten
    post /urls (ShortenRequest) returns (ShortenResponse)
}

@server (
    group: redirect
)
service url {
    @doc "Redirect to original URL"
    @handler Redirect
    get /:code
}

@server (
    prefix: /api/v1
    group: links
)
service url {
    @doc "List all links with pagination"
    @handler ListLinks
    get /links (LinkListRequest) returns (LinkListResponse)

    @doc "Get link details"
    @handler GetLinkDetail
    get /links/:code returns (LinkItem)

    @doc "Delete link"
    @handler DeleteLink
    delete /links/:code
}
```

**Code generation:**

```bash
cd services/url-api
goctl api go -api url.api -dir . -style gozero
```

**Key validation tags:**
- `optional` — field not required
- `default=X` — default value if not provided
- `range=[min:max]` — numeric range (inclusive)
- `options=a|b|c` — enum values
- `form` vs `json` vs `path` — parameter location

### Pattern 2: RPC Service Specification (.proto file)

**What:** Protocol Buffers definition for zRPC service. go-zero generates server/client stubs.

**When to use:** Service-to-service communication. Analytics exposes GetClickCount RPC for URL service to call.

**Example (Analytics Service .proto):**

```protobuf
// Source: go-zero zRPC documentation
syntax = "proto3";

package analytics;

option go_package = "./analytics";

// ========== Messages ==========

message GetClickCountRequest {
  string short_code = 1;
}

message GetClickCountResponse {
  string short_code = 1;
  int64 total_clicks = 2;
}

// ========== Service ==========

service Analytics {
  rpc GetClickCount(GetClickCountRequest) returns (GetClickCountResponse);
}
```

**Code generation:**

```bash
cd services/analytics-rpc
goctl rpc protoc analytics.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style gozero
```

**Minimal vs Full scope decision:** Start minimal (GetClickCount only). Add GetAnalyticsSummary RPC in Phase 9 when Kafka consumer is wired. Rationale: Phase 7 proves framework, Phase 9 proves messaging.

### Pattern 3: ServiceContext Dependency Injection

**What:** ServiceContext holds all dependencies (DB, RPC clients, config). Initialized once at startup, passed to all logic constructors.

**When to use:** Every service. Replaces manual dependency passing.

**Example (URL Service ServiceContext with stubs):**

```go
// Source: go-zero documentation + phase requirements
package svc

import (
    "services/url-api/internal/config"
)

type ServiceContext struct {
    Config config.Config
    // Phase 7: Stub fields (no DB, no RPC, no Kafka yet)
    // Phase 8 will add: UrlModel model.UrlModel
    // Phase 9 will add: AnalyticsRpc analyticsclient.Analytics
    // Phase 9 will add: KafkaProducer *kq.Pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
    return &ServiceContext{
        Config: c,
        // Phase 7: Empty struct, services boot successfully
    }
}
```

**Phase 8 evolution (for planning reference):**

```go
type ServiceContext struct {
    Config      config.Config
    UrlModel    model.UrlModel  // Added in Phase 8
}

func NewServiceContext(c config.Config) *ServiceContext {
    conn := sqlx.NewPostgres(c.Database.DataSource)
    return &ServiceContext{
        Config:   c,
        UrlModel: model.NewUrlModel(conn),
    }
}
```

### Pattern 4: Handler/Logic Separation

**What:** Handlers parse requests/write responses. Logic contains business rules. Clean separation enforced by goctl.

**When to use:** Always. Never put business logic in handlers.

**Example (Shorten endpoint):**

**Handler (generated, minimal edits):**

```go
// services/url-api/internal/handler/shorten/shortenhandler.go
package shorten

import (
    "net/http"
    "services/url-api/internal/logic/shorten"
    "services/url-api/internal/svc"
    "services/url-api/internal/types"
    "github.com/zeromicro/go-zero/rest/httpx"
)

func ShortenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.ShortenRequest
        if err := httpx.Parse(r, &req); err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        l := shorten.NewShortenLogic(r.Context(), svcCtx)
        resp, err := l.Shorten(&req)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
        } else {
            httpx.OkJsonCtx(r.Context(), w, resp)
        }
    }
}
```

**Logic (generated stub, manual implementation):**

```go
// services/url-api/internal/logic/shorten/shortenlogic.go
package shorten

import (
    "context"
    "services/url-api/internal/svc"
    "services/url-api/internal/types"
    "github.com/zeromicro/go-zero/core/logx"
)

type ShortenLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewShortenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ShortenLogic {
    return &ShortenLogic{
        Logger: logx.WithContext(ctx),
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

// Phase 7: Stub implementation
func (l *ShortenLogic) Shorten(req *types.ShortenRequest) (resp *types.ShortenResponse, err error) {
    // TODO: Phase 8 will add database logic
    return &types.ShortenResponse{
        ShortCode:   "stub123",
        ShortUrl:    "http://localhost:8080/stub123",
        OriginalUrl: req.OriginalUrl,
    }, nil
}
```

### Pattern 5: Custom Error Handler (RFC 7807 Preservation)

**What:** Override go-zero's default error handler to return RFC 7807 Problem Details format.

**When to use:** Main function setup, before server.Start().

**Example:**

```go
// services/url-api/url.go
package main

import (
    "flag"
    "fmt"
    "net/http"

    "services/url-api/internal/config"
    "services/url-api/internal/handler"
    "services/url-api/internal/svc"
    "pkg/problemdetails"

    "github.com/zeromicro/go-zero/core/conf"
    "github.com/zeromicro/go-zero/rest"
    "github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/url-api.yaml", "the config file")

func main() {
    flag.Parse()

    var c config.Config
    conf.MustLoad(*configFile, &c)

    server := rest.MustNewServer(c.RestConf)
    defer server.Stop()

    ctx := svc.NewServiceContext(c)
    handler.RegisterHandlers(server, ctx)

    // Custom error handler for RFC 7807 Problem Details
    httpx.SetErrorHandler(func(err error) (int, interface{}) {
        // Map go-zero errors to Problem Details
        problem := problemdetails.New(
            http.StatusInternalServerError,
            problemdetails.TypeInternalError,
            "Internal Server Error",
            err.Error(),
        )
        return problem.Status, problem
    })

    fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
    server.Start()
}
```

**Note:** Logic layer should return custom error types that the handler can detect and map to specific problem types. Phase 7 uses simple errors, Phase 8+ adds domain errors.

### Pattern 6: Configuration Loading (YAML)

**What:** go-zero loads config from YAML via conf.MustLoad(). Config struct embedded in main config.

**When to use:** Every service. Phase 7 minimal config, Phase 8+ adds DB, Phase 9+ adds Kafka.

**Example (Phase 7 URL Service config):**

```yaml
# services/url-api/etc/url-api.yaml
Name: url-api
Host: 0.0.0.0
Port: 8080

# Built-in features
Timeout: 30000      # Request timeout in ms
MaxConns: 10000     # Max concurrent connections

Log:
  ServiceName: url-api
  Mode: console     # console | file | volume
  Level: info       # info | error | severe
  KeepDays: 7

# Phase 7: Minimal config
BaseUrl: http://localhost:8080

# Phase 8 will add:
# Database:
#   DataSource: postgres://...
```

**Config struct:**

```go
// services/url-api/internal/config/config.go
package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
    rest.RestConf  // Embeds Name, Host, Port, Timeout, etc.
    BaseUrl string
    // Phase 8 adds: Database struct { DataSource string }
}
```

### Anti-Patterns to Avoid

- **Manual route registration:** Never use http.HandleFunc or chi.Router. Let goctl generate routes.go.
- **Business logic in handlers:** Handlers only parse/validate/respond. Logic layer owns business rules.
- **Editing types.go manually:** types.go is regenerated from .api file. Always edit .api, then regenerate.
- **Skipping ServiceContext:** Don't pass dependencies to logic constructors manually. Use ServiceContext.
- **Mixing validation locations:** Use .api tag validation, not manual checks in logic layer.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Request validation | Manual field checks in handlers | .api tag rules (`range`, `options`, `optional`) | go-zero validates before handler, returns structured errors |
| Error response format | Custom error middleware | httpx.SetErrorHandler + custom error types | Idiomatic go-zero pattern, integrates with httpx.ErrorCtx |
| Configuration loading | Manual YAML parsing | conf.MustLoad + embedded RestConf | Type-safe config with validation |
| Route registration | Manual http.HandleFunc or router setup | goctl-generated routes.go | Auto-wired from .api, prevents mistakes |
| Request/response types | Manual JSON struct tags | .api type definitions | Single source of truth, regenerates consistently |
| RPC client creation | Manual gRPC dial + client setup | goctl-generated client package | Handles connection, retries, service discovery |
| Structured logging | Custom logger wrappers | logx.WithContext | Integrates with request tracing, automatic context propagation |

**Key insight:** go-zero's value proposition is "convention over configuration." Fighting the conventions (manual routes, custom validation) loses framework benefits and creates maintenance debt.

## Common Pitfalls

### Pitfall 1: Editing Generated Code Incorrectly

**What goes wrong:** Developers edit types.go, routes.go, or handler signatures, then regenerate from .api. Changes are overwritten.

**Why it happens:** Generated files look like normal Go code. No clear "DO NOT EDIT" warning in go-zero templates.

**How to avoid:**
- **Never edit types.go** — it's fully regenerated. Edit .api instead.
- **Handlers can be edited** but signatures must match generated. Add logic in logic layer, not handlers.
- **routes.go is regenerated** — don't add custom routes there. Use middleware registration in main.go.

**Warning signs:** "My changes disappeared after running goctl" — you edited generated code.

### Pitfall 2: .api Validation Not Matching Logic Assumptions

**What goes wrong:** .api says field is `optional`, but logic layer assumes it's always present. Nil pointer panics.

**Why it happens:** Validation happens in unmarshaling layer. Logic doesn't re-check.

**How to avoid:**
- Make .api validation match business rules exactly.
- Required fields: omit `optional` tag.
- Enums: use `options=a|b|c` to restrict values.
- Ranges: use `range=[min:max]` for numbers.

**Warning signs:** Panic in logic layer on valid requests. Check if .api validation is too permissive.

### Pitfall 3: Confusing Service Groups and Route Prefixes

**What goes wrong:** Routes don't match expected paths. Handler generated in wrong directory.

**Why it happens:** go-zero has two grouping concepts:
- `@server(prefix: /api/v1)` — adds path prefix to all routes in block
- `@server(group: shorten)` — creates subdirectory `handler/shorten/`

**How to avoid:**
- Use `prefix` for API versioning (/api/v1, /api/v2).
- Use `group` for logical separation (shorten, redirect, links).
- Group name affects directory structure, not URL path.

**Example:**

```goapi
@server (
    prefix: /api/v1
    group: links
)
service url {
    @handler ListLinks
    get /links (ListLinkRequest) returns (ListLinkResponse)
}
```

- URL: `GET /api/v1/links`
- Handler file: `internal/handler/links/listlinkshandler.go`

### Pitfall 4: ServiceContext Initialization Order

**What goes wrong:** ServiceContext tries to connect to DB/Kafka in Phase 7 when dependencies aren't ready. Service fails to start.

**Why it happens:** Copying examples from docs that assume full stack is running.

**How to avoid (Phase 7 specific):**
- ServiceContext should only store config in Phase 7.
- Don't initialize DB connections, RPC clients, Kafka producers yet.
- Services must boot successfully with empty logic stubs.

**Warning signs:** "connection refused" errors on startup in Phase 7. Check if ServiceContext is trying to connect too early.

### Pitfall 5: Single Module Import Paths

**What goes wrong:** Imports use wrong paths. Goctl generates imports like `"url-api/internal/config"` instead of `"go-shortener/services/url-api/internal/config"`.

**Why it happens:** goctl infers module name from go.mod in current directory. If you run goctl from services/url-api/ without parent go.mod, it creates isolated module.

**How to avoid:**
- Single go.mod at repository root.
- Run goctl from service directory, but ensure parent go.mod exists.
- Check generated imports. If they don't include full path from module root, regenerate.

**Correct setup:**

```bash
# From repository root
cd services/url-api
goctl api go -api url.api -dir . -style gozero
# Generated imports should be: "go-shortener/services/url-api/internal/..."
```

### Pitfall 6: Error Handler Doesn't Receive Domain Errors

**What goes wrong:** Custom httpx.SetErrorHandler receives generic errors, not rich domain errors from logic.

**Why it happens:** Logic returns plain `fmt.Errorf()` or `errors.New()`. Error handler can't distinguish error types.

**How to avoid (Phase 8+):**
- Define custom error types implementing error interface.
- Logic returns typed errors (e.g., `ErrURLNotFound`, `ErrInvalidURL`).
- Error handler uses `errors.Is()` or type assertion to map to Problem Details.

**Phase 7 note:** Stub logic returns nil errors. Phase 8 adds domain errors.

## Code Examples

Verified patterns from official go-zero documentation:

### Complete .api Service with Multiple Groups

```goapi
// Source: go-zero route grouping documentation
syntax = "v1"

info (
    title: "URL Shortener API"
    desc: "URL shortening service"
    author: "go-shortener"
    version: "v2.0"
)

type ShortenRequest {
    OriginalUrl string `json:"original_url"`
}

type ShortenResponse {
    ShortCode   string `json:"short_code"`
    ShortUrl    string `json:"short_url"`
    OriginalUrl string `json:"original_url"`
}

// Group 1: Shorten operations
@server (
    prefix: /api/v1
    group: shorten
)
service url {
    @handler Shorten
    post /urls (ShortenRequest) returns (ShortenResponse)
}

// Group 2: Redirect (no prefix, root-level route)
@server (
    group: redirect
)
service url {
    @handler Redirect
    get /:code
}

// Group 3: Link management
@server (
    prefix: /api/v1
    group: links
)
service url {
    @handler ListLinks
    get /links

    @handler DeleteLink
    delete /links/:code
}
```

### Complete zRPC Service Setup

```bash
# Source: go-zero zRPC basic usage
# 1. Create proto file
cat > analytics.proto <<EOF
syntax = "proto3";

package analytics;
option go_package = "./analytics";

message GetClickCountRequest {
  string short_code = 1;
}

message GetClickCountResponse {
  string short_code = 1;
  int64 total_clicks = 2;
}

service Analytics {
  rpc GetClickCount(GetClickCountRequest) returns (GetClickCountResponse);
}
EOF

# 2. Generate code
goctl rpc protoc analytics.proto --go_out=. --go-grpc_out=. --zrpc_out=. --style gozero

# Generated structure:
# analytics/
#   analytics.pb.go
#   analytics_grpc.pb.go
# client/
#   analyticsclient/
#     analyticsclient.go
# internal/
#   config/config.go
#   logic/getclickcountlogic.go
#   server/analyticsserver.go
#   svc/servicecontext.go
# etc/analytics-rpc.yaml
# analytics.go (main)
```

### Request Validation Tags Reference

```goapi
// Source: go-zero parameter validation documentation

type ExampleRequest {
    // Required field (no optional tag)
    Name string `json:"name"`

    // Optional field
    Alias string `json:"alias,optional"`

    // Default value
    Age int `json:"age,default=20"`

    // Numeric range: (12:100] means > 12 and <= 100
    Score int `json:"score,range=(12:100]"`

    // Enum values
    Gender string `json:"gender,options=male|female|other"`

    // Combined: optional with default
    Avatar string `json:"avatar,optional,default=default.png"`

    // Form parameter (query string or form-data)
    Page int `form:"page,default=1,range=[1:]"`

    // Path parameter
    Code string `path:"code"`

    // Header parameter
    Authorization string `header:"Authorization"`
}
```

### Structured Logging with logx

```go
// Source: go-zero logx documentation

// In logic layer
func (l *ShortenLogic) Shorten(req *types.ShortenRequest) (*types.ShortenResponse, error) {
    // Context-aware logging (includes request ID, trace ID if enabled)
    logx.WithContext(l.ctx).Infow("shortening URL",
        logx.Field("original_url", req.OriginalUrl),
    )

    // Error logging
    if err != nil {
        logx.WithContext(l.ctx).Errorw("failed to shorten URL",
            logx.Field("original_url", req.OriginalUrl),
            logx.Field("error", err.Error()),
        )
        return nil, err
    }

    return resp, nil
}

// Configuration (in etc/url-api.yaml)
Log:
  ServiceName: url-api
  Mode: console        # console | file | volume
  Level: info          # info | error | severe
  Compress: false      # gzip compress for file mode
  KeepDays: 7         # retention for file mode
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual route registration (Chi) | .api specification + goctl | go-zero v1.0 (2020) | Type-safe, single source of truth |
| Manual validation in handlers | .api tag validation | go-zero v1.0 | Declarative, auto-generated |
| Custom error middleware | httpx.SetErrorHandler | go-zero v1.2+ | Idiomatic pattern, less code |
| Manual gRPC setup | goctl rpc protoc | go-zero v1.0 | Auto-generates client/server/logic |
| Zap/Zerolog | logx with context | go-zero v1.0 | Built-in request tracing |
| Manual config parsing | conf.MustLoad + RestConf | go-zero v1.0 | Type-safe, validated |

**Deprecated/outdated:**
- **goctl api new**: Still works but generates minimal example. For production, write .api spec first, then `goctl api go`.
- **Multi-module repos for go-zero**: Old docs suggest per-service go.mod. Current best practice: single module unless services are independently versioned.
- **@server(jwt: Auth)**: JWT middleware exists but deferred to v2.1 (auth out of scope).

## Recommendations (Claude's Discretion)

### Module Strategy: Single Module

**Decision:** Use single module at repository root.

**Rationale:**
- Project has 2 services (URL API, Analytics RPC). Go recommends single module for projects this size.
- Simplifies testing: `go test ./...` covers all services.
- No dependency versioning needed between services (they deploy together).
- go-zero community examples use single module for similar scales.

**Alternative rejected:** Multi-module with go workspaces adds complexity without benefit for 2 tightly coupled services.

### Shared Code Location

**Decision:**
- `pkg/` for project-specific libraries (problemdetails)
- `common/` for cross-service domain types (events)

**Rationale:**
- `pkg/` signals "library code, not a service." Aligns with Go community convention.
- `common/` avoids confusion with `internal/` (which means service-private in go-zero).
- Event definitions aren't utilities (not pkg/) and aren't service-specific (not internal/).

**Alternative rejected:** Single `shared/` directory mixes utilities and domain types.

### Generated Output Directory Structure

**Decision:** Use `services/` directory for all services.

**Rationale:**
- Clearer separation from `pkg/`, `common/`, `cmd/`.
- go-zero examples use top-level service directories, but `services/` adds structure.
- Matches user's Phase 1-6 convention (internal/ for service code).

**Alternative rejected:** Flat top-level (url-api/, analytics-rpc/ at root) clutters repository.

### Route Grouping Strategy

**Decision:** Group by domain (shorten, redirect, links) not by entity.

**Rationale:**
- Existing API already has logical grouping:
  - Shorten: POST /api/v1/urls
  - Redirect: GET /:code
  - Links: GET /api/v1/links, DELETE /api/v1/links/:code
- Grouping by domain matches user's Phase 4 "Link Management" feature split.

**Implementation:**

```goapi
@server(prefix: /api/v1, group: shorten)
service url {
    @handler Shorten
    post /urls (ShortenRequest) returns (ShortenResponse)
}

@server(group: redirect)
service url {
    @handler Redirect
    get /:code
}

@server(prefix: /api/v1, group: links)
service url {
    @handler ListLinks
    get /links (LinkListRequest) returns (LinkListResponse)

    @handler GetLinkDetail
    get /links/:code (LinkDetailRequest) returns (LinkDetailResponse)

    @handler DeleteLink
    delete /links/:code (DeleteLinkRequest)
}
```

### API Path Style

**Decision:** Keep existing paths, no cleanup needed.

**Rationale:**
- Current paths are RESTful and consistent:
  - POST /api/v1/urls (create)
  - GET /:code (redirect)
  - GET /api/v1/links (list)
  - GET /api/v1/links/:code (detail)
  - DELETE /api/v1/links/:code (delete)
- No external consumers to break, but existing paths are already clean.

**No changes needed.**

### Proto Scope for Analytics

**Decision:** Minimal scope in Phase 7 — single GetClickCount RPC.

**Rationale:**
- Phase 7 goal: prove framework integration, not feature completeness.
- GetClickCount is sufficient to test zRPC client/server pattern.
- Phase 9 will add GetAnalyticsSummary when Kafka consumer is wired.

**Proto definition (Phase 7):**

```protobuf
service Analytics {
  rpc GetClickCount(GetClickCountRequest) returns (GetClickCountResponse);
}
```

**Deferred to Phase 9:**
- GetAnalyticsSummary
- GetClickDetails
- Kafka consumer RPC (if needed)

### Error Response Format

**Decision:** Preserve RFC 7807 Problem Details via custom httpx.SetErrorHandler.

**Rationale:**
- User already has `pkg/problemdetails` package.
- RFC 7807 is industry standard, well-documented, client-friendly.
- go-zero's httpx.SetErrorHandler supports custom error formats cleanly.
- Simplifying to `{error: "message"}` loses structure (type, status, detail separation).

**Implementation:**

```go
// In main.go
httpx.SetErrorHandler(func(err error) (int, interface{}) {
    // Phase 7: Simple mapping
    problem := problemdetails.New(
        http.StatusInternalServerError,
        problemdetails.TypeInternalError,
        "Internal Server Error",
        err.Error(),
    )
    return problem.Status, problem

    // Phase 8+: Map domain errors to specific problem types
})
```

**Per-field validation errors:**

When .api validation fails, go-zero returns errors like `"field 'age' must be in range (12:100]"`. Custom error handler should parse these into:

```json
{
  "type": "https://api.example.com/problems/validation-error",
  "title": "Validation Failed",
  "status": 400,
  "detail": "Request validation failed",
  "errors": [
    {"field": "age", "message": "must be in range (12:100]"}
  ]
}
```

Phase 7 can return simple string. Phase 8 adds structured parsing.

### Log Format Strategy

**Decision:**
- **console mode for development** (plain text, human-readable)
- **file mode for production** (JSON, structured, parseable)

**Rationale:**
- go-zero's `Mode: console` outputs plain text with color. Good for local dev.
- `Mode: file` outputs JSON. Production logs should be JSON for log aggregation (ELK, Loki).
- Configuration is per-environment via YAML. No code changes needed.

**Configuration:**

```yaml
# etc/url-api.yaml (development)
Log:
  Mode: console
  Level: info

# etc/url-api-prod.yaml (production)
Log:
  Mode: file
  Level: error
  Path: /var/log/url-api
  Compress: true
  KeepDays: 30
```

**Built-in request logging:** Enabled by default. go-zero's rest.Server logs every request with method, path, status, latency. No middleware needed.

## Open Questions

1. **Migration path for existing data**
   - What we know: Phase 7 is framework skeleton. No database yet.
   - What's unclear: Phase 8 will migrate SQLite → PostgreSQL. Do we need data migration scripts?
   - Recommendation: Phase 8 planning should include data export/import strategy. Phase 7 ignores database.

2. **Health check endpoints in go-zero**
   - What we know: Existing /healthz and /readyz check DB connectivity.
   - What's unclear: Does go-zero have built-in health check middleware, or add manually?
   - Recommendation: Add health check routes to .api manually. go-zero doesn't generate them. Reuse existing healthz/readyz logic in Phase 7 stubs.

3. **Prometheus metrics endpoint**
   - What we know: go-zero auto-exposes /metrics if configured (Phase 10 requirement).
   - What's unclear: Does it work out-of-box in Phase 7, or requires middleware setup?
   - Recommendation: Phase 7 can test by adding `Prometheus: ListenOn: 9091` to config. Not critical for Phase 7 success.

## Sources

### Primary (HIGH confidence)

- [go-zero documentation (zero-doc)](https://github.com/zeromicro/zero-doc) - API syntax, zRPC setup, configuration
- [go-zero official site](https://go-zero.dev/en/) - Project structure, validation tags, service groups
- [go-zero GitHub repository](https://github.com/zeromicro/go-zero) - Source code reference for logx, httpx, conf packages
- goctl CLI --help output - Code generation flags and options

### Secondary (MEDIUM confidence)

- [Parameter Rules | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/api/parameter) - Validation tag syntax
- [Request Parameters | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/http/server/request-body) - Request deserialization
- [Building a Monorepo in Golang - Earthly Blog](https://earthly.dev/blog/golang-monorepo/) - Single vs multi-module decision
- [Go Modules- A Guide for monorepos (Part 1)](https://engineering.grab.com/go-module-a-guide-for-monorepos-part-1) - Monorepo best practices

### Tertiary (LOW confidence - general ecosystem)

- [Framework Overview | go-zero Documentation](https://go-zero.dev/en/docs/concepts/overview) - General feature overview
- [Effective Error Handling in Go: Best Practices and Common Pitfalls](https://codezup.com/effective-error-handling-in-go-best-practices-and-pitfalls/) - Error handling patterns (not go-zero specific)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official go-zero packages and goctl verified via Context7 and CLI
- Architecture: HIGH - Patterns extracted from official documentation and verified examples
- Pitfalls: MEDIUM - Derived from documentation caveats and community patterns, not exhaustive field experience
- Recommendations: HIGH - Based on Go community best practices and go-zero conventions

**Research date:** 2026-02-16
**Valid until:** 2026-03-16 (30 days - go-zero is stable, major patterns unlikely to change)
