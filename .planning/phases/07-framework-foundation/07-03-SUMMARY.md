---
phase: 07-framework-foundation
plan: 03
subsystem: analytics-rpc
tags:
  - rpc
  - grpc
  - code-generation
  - go-zero
  - protobuf
dependencies:
  requires:
    - 07-01-SUMMARY.md (goctl installation, framework patterns)
  provides:
    - Analytics RPC service skeleton
    - GetClickCount RPC stub (returns 42)
    - Generated client package for cross-service calls
  affects:
    - Makefile (added run-analytics, gen-analytics, run-all targets)
tech_stack:
  added:
    - go-zero zRPC server (gRPC wrapper)
    - protoc-gen-go (protobuf Go plugin)
    - protoc-gen-go-grpc (gRPC Go plugin)
    - google.golang.org/grpc (gRPC library)
  patterns:
    - Protobuf-first RPC design
    - goctl rpc code generation
    - Direct connection mode (no Etcd in Phase 7)
    - Console logging for development
key_files:
  created:
    - services/analytics-rpc/analytics.proto (RPC spec)
    - services/analytics-rpc/analytics.go (main entry point)
    - services/analytics-rpc/analytics/analytics.pb.go (protobuf stubs)
    - services/analytics-rpc/analytics/analytics_grpc.pb.go (gRPC interfaces)
    - services/analytics-rpc/analyticsclient/analytics.go (RPC client)
    - services/analytics-rpc/internal/logic/getclickcountlogic.go (business logic stub)
    - services/analytics-rpc/internal/server/analyticsserver.go (gRPC server impl)
    - services/analytics-rpc/internal/svc/servicecontext.go (service context)
    - services/analytics-rpc/internal/config/config.go (config struct)
    - services/analytics-rpc/etc/analytics.yaml (service config)
  modified:
    - Makefile (added Analytics RPC targets)
decisions:
  - decision: "Use port 8081 for Analytics RPC (8080 used by URL API)"
    rationale: "Clear service separation in development"
  - decision: "Remove Etcd configuration for Phase 7"
    rationale: "Direct connection mode sufficient for initial framework validation; service discovery added in Phase 10"
  - decision: "Stub GetClickCount returns total_clicks: 42"
    rationale: "Proves RPC framework works; real PostgreSQL queries added in Phase 8"
  - decision: "Add phase comments to ServiceContext"
    rationale: "Document future integration points (Phase 8: DB model, Phase 9: Kafka consumer)"
metrics:
  duration_seconds: 231
  tasks_completed: 2
  files_created: 10
  files_modified: 4
  commits: 2
  completed_at: "2026-02-16T00:40:39Z"
---

# Phase 7 Plan 3: Analytics RPC Service

**One-liner:** Analytics RPC service with GetClickCount stub using go-zero zRPC and protobuf code generation.

## What Was Built

Created the Analytics Service as a functional go-zero zRPC (gRPC) service:

1. **Protobuf Specification:** Defined `analytics.proto` with GetClickCount RPC accepting short_code and returning total_clicks
2. **Code Generation:** Used `goctl rpc protoc` to generate server skeleton, client package, logic stubs, config, and proto bindings
3. **Stub Logic:** Implemented GetClickCount to return `total_clicks: 42` with structured logging
4. **Configuration:** Customized service to run on port 8081 with console logging, removed Etcd dependency
5. **Build Tooling:** Added Makefile targets for running and regenerating the service

The service boots cleanly, listens on port 8081, and provides a generated client package ready for cross-service calls in Phase 9.

## Technical Implementation

### Service Structure

```
services/analytics-rpc/
├── analytics.proto              # Protobuf service definition
├── analytics.go                 # Main entry point (zrpc.MustNewServer)
├── etc/analytics.yaml           # Service config (port 8081, console logging)
├── analytics/                   # Generated protobuf code
│   ├── analytics.pb.go         # Message types
│   └── analytics_grpc.pb.go    # gRPC client/server interfaces
├── analyticsclient/             # Generated RPC client
│   └── analytics.go            # Client wrapper for URL service to import
└── internal/
    ├── config/config.go         # Config struct (zrpc.RpcServerConf)
    ├── logic/
    │   └── getclickcountlogic.go  # Stub returning total_clicks: 42
    ├── server/analyticsserver.go   # gRPC server implementation
    └── svc/servicecontext.go       # Service context (Phase 7: config only)
```

### Generated Code Flow

1. **analytics.proto → goctl rpc protoc → 10 generated files**
2. **Import paths automatically use full module path:** `go-shortener/services/analytics-rpc/...`
3. **zRPC wraps gRPC with go-zero conventions:** logging, middleware, service group support
4. **Direct connection mode:** No Etcd configured, clients connect to ListenOn address directly

### Stub Logic (Phase 7)

```go
func (l *GetClickCountLogic) GetClickCount(in *analytics.GetClickCountRequest) (*analytics.GetClickCountResponse, error) {
    logx.WithContext(l.ctx).Infow("get click count",
        logx.Field("short_code", in.ShortCode),
    )

    // Phase 7 stub: return fake click count
    // Phase 8: Query PostgreSQL for actual click count
    return &analytics.GetClickCountResponse{
        ShortCode:   in.ShortCode,
        TotalClicks: 42,
    }, nil
}
```

### Configuration (analytics.yaml)

```yaml
Name: analytics-rpc
ListenOn: 0.0.0.0:8081

Log:
  ServiceName: analytics-rpc
  Mode: console
  Level: info

# Phase 7: No etcd, no database, no Kafka
# go-zero zRPC defaults to direct connection mode when Etcd is not configured
```

### Makefile Targets

```makefile
run-analytics:    # Start Analytics RPC service
gen-analytics:    # Regenerate from analytics.proto
run-all:          # Start both URL API and Analytics RPC
```

## Deviations from Plan

None - plan executed exactly as written.

## What's Next

**Phase 7 Plan 3 completes the Analytics RPC skeleton.**

**Next immediate step:**
- If Phase 7 has more plans: Execute next plan in wave
- If Phase 7 complete: Phase 8 will add PostgreSQL database, migrate from SQLite, wire real click count queries

**Future integration points:**
- **Phase 8:** Add ClickModel to ServiceContext, query PostgreSQL in GetClickCount
- **Phase 9:** Add Kafka consumer to ServiceContext, consume click events, insert into database
- **Phase 10:** Add Etcd for service discovery, enable RPC calls from URL API to Analytics RPC

## Verification Summary

All verification criteria met:

- ✅ analytics.proto defines GetClickCount RPC with request/response messages
- ✅ goctl rpc protoc generates server, client, logic, config without errors
- ✅ Analytics RPC service starts on port 8081 and accepts gRPC connections
- ✅ GetClickCount returns stub response (short_code echoed, total_clicks: 42)
- ✅ Generated client package is importable by other services
- ✅ Configuration loaded from YAML with console logging enabled
- ✅ ServiceContext has config only (Phase 7 stub)
- ✅ Makefile targets work: run-analytics, gen-analytics, run-all

## Self-Check

Verifying all claimed artifacts exist and commits are real.

**Files created:**
- ✓ services/analytics-rpc/analytics.proto
- ✓ services/analytics-rpc/analytics.go
- ✓ services/analytics-rpc/analytics/analytics.pb.go
- ✓ services/analytics-rpc/analytics/analytics_grpc.pb.go
- ✓ services/analytics-rpc/analyticsclient/analytics.go
- ✓ services/analytics-rpc/internal/logic/getclickcountlogic.go
- ✓ services/analytics-rpc/internal/server/analyticsserver.go
- ✓ services/analytics-rpc/internal/svc/servicecontext.go
- ✓ services/analytics-rpc/internal/config/config.go
- ✓ services/analytics-rpc/etc/analytics.yaml

**Commits:**
- ✓ eb94785 (Task 1: Generate Analytics RPC service from proto spec)
- ✓ dc9248f (Task 2: Customize Analytics RPC config and stub logic)

## Self-Check: PASSED

All claimed files exist and commits are verified.
