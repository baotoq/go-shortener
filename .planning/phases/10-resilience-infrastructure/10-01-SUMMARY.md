---
phase: 10-resilience-infrastructure
plan: 01
subsystem: observability
tags: [prometheus, metrics, devserver, health-checks]
dependency_graph:
  requires: []
  provides: [prometheus-metrics, health-endpoints]
  affects: [url-api, analytics-rpc, analytics-consumer]
tech_stack:
  added: []
  patterns: [devserver, prometheus-exposition]
key_files:
  created: []
  modified:
    - services/url-api/etc/url.yaml
    - services/analytics-rpc/etc/analytics.yaml
    - services/analytics-consumer/etc/consumer.yaml
    - services/analytics-consumer/consumer.go
decisions:
  - key: devserver-port-allocation
    choice: url-api=6470, analytics-rpc=6471, analytics-consumer=6472
    rationale: Unique ports for each service to avoid conflicts during local development
  - key: consumer-devserver-initialization
    choice: Call c.MustSetUp() after config load
    rationale: Explicit setup required for non-HTTP services to initialize DevServer
  - key: analytics-rpc-timeout
    choice: 2000ms timeout on zRPC client
    rationale: Aligns with resilience requirements (RESL-03) for bounded request durations
metrics:
  duration_seconds: 312
  tasks_completed: 1
  files_modified: 4
  commits: 1
  completed_date: 2026-02-16
---

# Phase 10 Plan 01: Resilience & Observability Summary

Enabled Prometheus metrics via go-zero DevServer on all three services with unique ports and health check endpoints.

## What Was Done

### Task 1: Enable DevServer Prometheus metrics on all three services

**Objective**: Configure go-zero's built-in DevServer to expose Prometheus metrics and health endpoints on all three services.

**Implementation**:

1. **url-api** (DevServer on port 6470):
   - Added DevServer config block with Enabled=true, Port=6470
   - Configured /metrics for Prometheus scraping
   - Configured /healthz for health checks
   - Added Timeout=2000ms to AnalyticsRpc client config
   - Verified Timeout=30000ms and MaxConns=10000 already configured

2. **analytics-rpc** (DevServer on port 6471):
   - Added DevServer config block with Enabled=true, Port=6471
   - Configured /metrics and /healthz endpoints
   - zRPC server has built-in circuit breakers, load shedding, and tracing (enabled by default)

3. **analytics-consumer** (DevServer on port 6472):
   - Added DevServer config block with Enabled=true, Port=6472
   - Updated consumer.go to call c.MustSetUp() after config load
   - This initializes ServiceConf infrastructure (logging, metrics, devserver, tracing)

**Verification**:
- All services compile successfully
- url-api /metrics endpoint returns Prometheus metrics including:
  - `http_server_requests_duration_ms` histogram
  - `http_server_requests_code_total` counter
  - Standard Go runtime metrics (go_goroutines, go_memstats, etc.)
- analytics-rpc /metrics endpoint returns metrics
- analytics-consumer /metrics endpoint returns metrics
- All /healthz endpoints respond (url-api and analytics-rpc: OK, consumer: depends on Kafka)

**Files Modified**:
- `services/url-api/etc/url.yaml`: Added DevServer block, AnalyticsRpc.Timeout
- `services/analytics-rpc/etc/analytics.yaml`: Added DevServer block
- `services/analytics-consumer/etc/consumer.yaml`: Added DevServer block
- `services/analytics-consumer/consumer.go`: Added c.MustSetUp() call

**Commit**: `ee58c0f` - feat(10-01): enable DevServer Prometheus metrics on all services

## Deviations from Plan

None - plan executed exactly as written.

The plan anticipated a potential issue with DevServer auto-starting for non-HTTP services (analytics-consumer). This was resolved by adding an explicit `c.MustSetUp()` call after config load, which is the standard go-zero pattern for initializing ServiceConf infrastructure.

## Technical Details

### go-zero Built-in Resilience Features

The plan leverages go-zero's built-in resilience features (all enabled by default, no code changes needed):

1. **Circuit Breakers** (RESL-02): zRPC clients use Google SRE adaptive breaker automatically
2. **Load Shedding** (RESL-01): REST server has built-in adaptive load shedder based on CPU
3. **Request Timeouts** (RESL-03): Configured at REST server (30s) and zRPC client (2s) level
4. **Connection Limits** (RESL-04): MaxConns=10000 on url-api REST server
5. **Prometheus Metrics** (RESL-05): DevServer exposes metrics automatically via Prometheus client
6. **OpenTelemetry Tracing** (RESL-06): Tracing middleware enabled by default on REST and RPC servers

### DevServer Architecture

go-zero's DevServer is a separate HTTP server that runs on a different port from the main application server. It provides:

- **Metrics endpoint** (`/metrics`): Prometheus-compatible metrics exposition
- **Health endpoint** (`/healthz`): Health check for Docker/k8s readiness probes
- **Profiling** (optional): pprof endpoints when enabled

This separation ensures observability infrastructure doesn't interfere with application traffic.

### Port Allocation Strategy

Chose 6400-series ports for DevServer to avoid conflicts with main service ports (8000-series):

| Service              | Main Port | DevServer Port |
|----------------------|-----------|----------------|
| url-api              | 8080      | 6470           |
| analytics-rpc        | 8081      | 6471           |
| analytics-consumer   | N/A       | 6472           |

### ServiceConf.MustSetUp() Pattern

For non-HTTP services like analytics-consumer, go-zero doesn't automatically call `MustSetUp()` (unlike rest.Server and zrpc.Server which do this during Start()). Therefore, we explicitly call it after config load:

```go
var c config.Config
conf.MustLoad(*configFile, &c)
c.MustSetUp()  // Initialize logging, metrics, devserver, tracing
```

This pattern is documented in go-zero for standalone services and queue consumers.

## Testing Notes

**Verified Metrics**:

```bash
# url-api metrics (sample)
curl http://localhost:6470/metrics

# Includes HTTP server metrics:
http_server_requests_duration_ms_bucket{code="404",method="GET",path="/:code",le="50"} 1
http_server_requests_code_total{code="404",method="GET",path="/:code"} 1

# Includes Go runtime metrics:
go_goroutines 17
go_memstats_alloc_bytes 3.084192e+06
```

**Health Checks**:

```bash
# url-api and analytics-rpc
curl http://localhost:6470/healthz  # Returns: OK
curl http://localhost:6471/healthz  # Returns: OK

# analytics-consumer
curl http://localhost:6472/healthz  # Returns: Service Unavailable (when Kafka down)
                                     # Returns: OK (when Kafka up and consumer connected)
```

The consumer health check correctly reflects Kafka connectivity state.

## Requirements Satisfied

- **RESL-01**: Load shedding enabled by default on REST server (go-zero adaptive shedder)
- **RESL-02**: Circuit breakers active on zRPC clients (Google SRE breaker)
- **RESL-03**: Request timeouts configured (REST: 30s, zRPC: 2s)
- **RESL-04**: Connection limits configured (MaxConns=10000)
- **RESL-05**: Prometheus metrics exposed on all services via DevServer
- **RESL-06**: OpenTelemetry tracing enabled by default (middleware active)

All resilience and observability requirements are now satisfied through configuration only, with no custom middleware code required.

## Next Steps

Plan 10-02 will create multi-target Dockerfile and docker-compose.yaml for containerized deployment. The DevServer ports configured here will be used for Prometheus scraping in the Docker Compose setup.

Plan 10-03 will add unit tests with hand-written mocks for model layer.

## Self-Check

Verifying all claimed files and commits exist:

**Files Modified**:
- services/url-api/etc/url.yaml
- services/analytics-rpc/etc/analytics.yaml
- services/analytics-consumer/etc/consumer.yaml
- services/analytics-consumer/consumer.go

**Commits**:
- ee58c0f: feat(10-01): enable DevServer Prometheus metrics on all services
