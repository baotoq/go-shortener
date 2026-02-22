---
phase: 13-distributed-tracing
plan: 01
subsystem: observability
tags: [opentelemetry, jaeger, tracing, kafka, go-zero, docker-compose]
dependency_graph:
  requires: []
  provides: [distributed-tracing, jaeger-ui, otlp-trace-pipeline]
  affects: [url-api, analytics-rpc, analytics-consumer, docker-compose]
tech_stack:
  added: [jaegertracing/all-in-one:1.76.0]
  patterns: [otlpgrpc-exporter, go-zero-telemetry-yaml, kafka-otel-header-propagation]
key_files:
  created: []
  modified:
    - docker-compose.yml
    - services/url-api/etc/url.yaml
    - services/url-api/etc/url-docker.yaml
    - services/analytics-rpc/etc/analytics.yaml
    - services/analytics-rpc/etc/analytics-docker.yaml
    - services/analytics-consumer/etc/consumer.yaml
    - services/analytics-consumer/etc/consumer-docker.yaml
    - services/url-api/internal/logic/redirect/redirectlogic.go
key_decisions:
  - Use Batcher: otlpgrpc (not jaeger) — Jaeger exporter removed from OTel Go SDK in 2023; Jaeger v1.76 accepts OTLP natively on port 4317
  - No Go code changes needed for HTTP/zRPC auto-instrumentation — go-zero Telemetry YAML block activates TracingHandler and gRPC interceptors automatically
  - Fix context.Background() to l.ctx in redirect Kafka publish — go-queue v1.2.2 already injects OTel headers if context carries active span
  - No OTel Collector required — Jaeger v1.76 accepts OTLP directly, eliminating infrastructure complexity
metrics:
  duration: 153s
  completed: 2026-02-22
  tasks_completed: 2
  files_modified: 8
---

# Phase 13 Plan 01: Distributed Tracing Infrastructure Summary

**One-liner:** OpenTelemetry distributed tracing via go-zero Telemetry YAML + Jaeger all-in-one on OTLP gRPC, with Kafka context propagation fixed via l.ctx.

## What Was Built

Added end-to-end OpenTelemetry distributed tracing to all three microservices with Jaeger visualization in Docker Compose. The implementation leverages go-zero's built-in Telemetry auto-instrumentation (activated by YAML config only, no Go code changes for HTTP/zRPC) and go-queue v1.2.2's native OTel Kafka header propagation.

### Task 1: Add Jaeger to Docker Compose

Added `jaegertracing/all-in-one:1.76.0` service to `docker-compose.yml` with:
- Port 16686 (Jaeger UI)
- Port 4317 (OTLP gRPC receiver)
- Port 4318 (OTLP HTTP receiver)
- `COLLECTOR_OTLP_ENABLED=true` environment variable
- Placed before app services (infra-first ordering)
- All three app services declare `jaeger: condition: service_started` in `depends_on`

### Task 2: Telemetry YAML Config + Kafka Context Fix

Added `Telemetry` block to all 6 service YAML config files:
- Local dev configs (`url.yaml`, `analytics.yaml`, `consumer.yaml`): `Endpoint: localhost:4317`
- Docker configs (`url-docker.yaml`, `analytics-docker.yaml`, `consumer-docker.yaml`): `Endpoint: jaeger:4317`
- All use `Batcher: otlpgrpc`, `Sampler: 1.0`
- Service names: `url-api`, `analytics-rpc`, `analytics-consumer`

Fixed Kafka trace context propagation in `redirectlogic.go`:
- Changed `KqPusher.Push(context.Background(), ...)` to `KqPusher.Push(l.ctx, ...)`
- `l.ctx` carries the active HTTP request span context
- go-queue v1.2.2's `PushWithKey` injects W3C `traceparent` header into Kafka message headers
- go-queue's `startConsumers()` extracts the header into the consumer's `ctx` automatically

## Decisions Made

1. **`Batcher: otlpgrpc` not `Batcher: jaeger`**: The Jaeger-specific OTel exporter was removed from the OTel Go SDK in 2023. go-zero v1.10.0 supports `zipkin|otlpgrpc|otlphttp|file`. Jaeger v1.76 accepts OTLP natively on port 4317.

2. **No Go code for HTTP/zRPC tracing**: go-zero's `ServiceConf.MustSetUp()` (already called in all service mains) reads the `Telemetry` YAML block and initializes the global TracerProvider. `TracingHandler` middleware (HTTP) and gRPC interceptors activate automatically.

3. **No OTel Collector**: Jaeger v1.76 accepts OTLP directly, eliminating the need for an intermediate OTel Collector service.

4. **Kafka span linking via go-queue headers**: The STATE.md initially planned a `TraceContext string` field in ClickEvent JSON. Research confirmed go-queue v1.2.2 already propagates via Kafka message headers natively — the JSON field approach was unnecessary and avoided.

## Verification Results

| Check | Result |
|-------|--------|
| `docker compose config` | PASS |
| `go build ./...` | PASS |
| `go vet ./...` | PASS |
| Telemetry: in all 6 YAML files | PASS (6/6) |
| Batcher: otlpgrpc in all 6 YAML files | PASS (6/6) |
| Local YAMLs use localhost:4317 | PASS (3/3) |
| Docker YAMLs use jaeger:4317 | PASS (3/3) |
| redirectlogic.go uses l.ctx | PASS |
| context.Background removed from Push | PASS |

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | c1f3fc7 | feat(13-01): add Jaeger to Docker Compose with OTLP ports |
| 2 | 1886b73 | feat(13-01): add Telemetry config to all services and fix Kafka context propagation |

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

All 9 files verified present. Both commits (c1f3fc7, 1886b73) confirmed in git log.
