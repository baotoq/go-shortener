# Stack Research

**Domain:** Observability, tracing, metrics, and service discovery additions to go-zero microservices
**Researched:** 2026-02-22
**Confidence:** HIGH

## Context: What Already Exists vs What Is New

This research covers **only the v3.0 additions**. The existing stack (go-zero v1.10.0, PostgreSQL, Kafka, zRPC) is validated and unchanged.

### Critical Discovery: go-zero Built-in Telemetry

**go-zero already ships all necessary OpenTelemetry and Prometheus instrumentation as indirect dependencies.** The `go.mod` already contains:

- `go.opentelemetry.io/otel v1.35.0` (indirect, via go-zero)
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.24.0` (indirect)
- `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.24.0` (indirect)
- `go.opentelemetry.io/otel/exporters/zipkin v1.24.0` (indirect)
- `github.com/prometheus/client_golang v1.23.2` (indirect, via go-zero)
- `go.etcd.io/etcd/client/v3 v3.5.15` (indirect, via go-zero)

**No new Go library imports are required for tracing, metrics, or Etcd.** All activation is via YAML configuration and Docker infrastructure.

---

## Recommended Stack: New Infrastructure Only

### Distributed Tracing: Jaeger v2

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Jaeger all-in-one | `jaegertracing/jaeger:2.15.0` | Trace visualization UI + OTLP receiver | Jaeger v2 is built on OpenTelemetry Collector core. Jaeger v1 reached EOL December 31, 2025. v2 natively accepts OTLP (ports 4317 gRPC, 4318 HTTP) and provides Jaeger UI on port 16686. Single container for local dev. |
| go-zero Telemetry config | Built-in (no new package) | OTel trace export to Jaeger | go-zero's `Telemetry` YAML block activates the built-in OTel SDK. Use `Batcher: otlpgrpc` targeting Jaeger's OTLP gRPC port. The `jaeger` batcher option exists in older docs but the OTel jaeger exporter was deprecated in July 2023. Use `otlpgrpc` instead. |

**go-zero Telemetry config struct** (source: `core/trace/config.go`, verified from source):

```go
type Config struct {
    Name           string            // service identifier in traces
    Endpoint       string            // OTLP collector address (e.g., "jaeger:4317" for gRPC)
    Sampler        float64           // default=1.0 (100% sampling)
    Batcher        string            // default=otlpgrpc, options=zipkin|otlpgrpc|otlphttp|file
    OtlpHeaders    map[string]string // optional custom headers
    OtlpHttpPath   string            // optional, for otlphttp (e.g., "/v1/traces")
    OtlpHttpSecure bool              // optional, TLS for otlphttp
    Disabled       bool              // optional, disable tracing
}
```

**Activation in each service YAML** (no code changes required for HTTP and zRPC):

```yaml
Telemetry:
  Name: url-api          # unique per service
  Endpoint: jaeger:4317  # OTLP gRPC endpoint (no http:// prefix for gRPC)
  Sampler: 1.0
  Batcher: otlpgrpc
```

**Kafka trace propagation** requires manual context inject/extract. go-zero does NOT auto-propagate trace context through Kafka messages. The producer must inject W3C `traceparent` headers into Kafka message headers, and the consumer must extract them. Uses `otel.GetTextMapPropagator()` from the already-present OTel SDK.

### Prometheus Metrics

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| prom/prometheus | `v3.9.1` | Metrics scraping and storage | Prometheus 3.x is the current stable series (v3.9.1 released 2026-01-07). Prometheus 2.x is no longer actively developed. |
| go-zero DevServer | Built-in (no new package) | Metrics endpoint exposure | go-zero's `DevServer` block auto-exposes `/metrics` on a configurable port. Already configured in the project on ports 6470, 6471, 6472. Collects `http_server_requests_duration_ms` and `http_server_requests_code_total` automatically. |

**DevServer already configured** in all three services. Prometheus only needs a `prometheus.yml` scrape config:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: url-api
    static_configs:
      - targets: ['url-api:6470']
  - job_name: analytics-rpc
    static_configs:
      - targets: ['analytics-rpc:6471']
  - job_name: analytics-consumer
    static_configs:
      - targets: ['analytics-consumer:6472']
```

### Grafana Dashboards

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| grafana/grafana | `12.3.3` | Dashboard visualization | Latest stable (released 2026-02-12). Supports auto-provisioning datasources and dashboards from YAML/JSON files mounted into the container. No manual UI setup needed. |

**Auto-provisioning pattern** (verified from Grafana official docs):

```
grafana/
  provisioning/
    datasources/
      prometheus.yaml   # auto-registers Prometheus as default datasource
    dashboards/
      dashboard.yaml    # points to dashboard JSON directory
  dashboards/
    url-shortener.json  # pre-built dashboard JSON
```

### Etcd Service Discovery

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| gcr.io/etcd-development/etcd | `v3.5.27` | Service registry for zRPC | go-zero ships `go.etcd.io/etcd/client/v3 v3.5.15` as an indirect dep. etcd 3.5.x is the stable series (v3.5.27 is current). Use the official `gcr.io/etcd-development/etcd` image — `bitnami/etcd` is deprecated (moved to `bitnamilegacy`). |
| go-zero EtcdConf | Built-in (no new package) | Client-side service discovery and server-side registration | go-zero's zRPC natively integrates with etcd. Server registers itself on startup; client resolves instances dynamically. Replaces hardcoded `dns:///localhost:8081`. |

**Server registration** (analytics-rpc YAML change):

```yaml
# services/analytics-rpc/etc/analytics.yaml
Name: analytics-rpc
ListenOn: 0.0.0.0:8081
Etcd:
  Hosts:
    - etcd:2379
  Key: analytics-rpc
```

**Client discovery** (url-api YAML change):

```yaml
# services/url-api/etc/url.yaml
AnalyticsRpc:
  Etcd:
    Hosts:
      - etcd:2379
    Key: analytics-rpc
  NonBlock: true
  Timeout: 2000
```

**No Go code changes needed** — go-zero's `zrpc.MustNewClient()` and `zrpc.MustNewServer()` detect the Etcd config and switch from direct connection to etcd-based discovery automatically.

---

## Supporting Libraries

No new Go dependencies required. All libraries are already present as indirect dependencies of go-zero.

| Library | Already Present | Role |
|---------|----------------|------|
| `go.opentelemetry.io/otel` | v1.35.0 indirect | OTel SDK core — activated by Telemetry YAML block |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.24.0 indirect | OTLP gRPC exporter — used when `Batcher: otlpgrpc` |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` | v1.24.0 indirect | OTLP HTTP exporter — used when `Batcher: otlphttp` |
| `github.com/prometheus/client_golang` | v1.23.2 indirect | Prometheus metrics — used by go-zero DevServer |
| `go.etcd.io/etcd/client/v3` | v3.5.15 indirect | Etcd client — used when EtcdConf is present in YAML |
| `go.opentelemetry.io/otel/propagation` | indirect (via otel) | W3C TraceContext propagation — needed for manual Kafka inject/extract |

**One exception:** Kafka trace propagation requires calling OTel propagation APIs directly in the producer/consumer code. The packages are already present in `go.mod` — no `go get` needed, but the imports must be added explicitly:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/propagation"
)
```

---

## Development Tools: New Docker Compose Services

| Service | Image | Ports | Purpose |
|---------|-------|-------|---------|
| Jaeger | `jaegertracing/jaeger:2.15.0` | 16686 (UI), 4317 (OTLP gRPC), 4318 (OTLP HTTP) | Trace visualization |
| Prometheus | `prom/prometheus:v3.9.1` | 9090 | Metrics scraping and query |
| Grafana | `grafana/grafana:12.3.3` | 3000 | Dashboards |
| Etcd | `gcr.io/etcd-development/etcd:v3.5.27` | 2379 | Service registry |

---

## Installation

No Go package changes needed. All Go libraries activate via YAML config. New infrastructure containers added to `docker-compose.yml`.

```bash
# Verify no new Go dependencies needed
# go.mod already contains all required packages as indirect deps
# of go-zero v1.10.0

# Infrastructure only — Docker Compose additions
# jaeger, prometheus, grafana, etcd added to docker-compose.yml
```

---

## Alternatives Considered

| Recommended | Alternative | Why Not |
|-------------|-------------|---------|
| `Batcher: otlpgrpc` targeting Jaeger | `Batcher: jaeger` (legacy) | The OTel Jaeger exporter was deprecated July 2023 and removed from OTel Go. go-zero's `jaeger` batcher option wraps the deprecated exporter. Jaeger v2 natively accepts OTLP — send OTLP directly. |
| Jaeger v2 (`jaegertracing/jaeger:2.15.0`) | Jaeger v1 (`jaegertracing/all-in-one`) | Jaeger v1 reached EOL December 31, 2025. v2 is built on OTel Collector core, natively accepts OTLP without translation overhead. |
| `prom/prometheus:v3.9.1` | `prom/prometheus:v2.55.x` | Prometheus 3.x is the current major release (since November 2024). 2.x is in maintenance-only mode. |
| `gcr.io/etcd-development/etcd:v3.5.27` | `bitnami/etcd` | Bitnami etcd image deprecated, moved to `bitnamilegacy/etcd`. Use official image. |
| Grafana auto-provisioning (YAML files) | Manual Grafana UI setup | Auto-provisioning survives container restarts with zero manual steps. Mount `provisioning/` directory into the container. |
| go-zero built-in OTel tracing | Custom OTel setup | go-zero activates full OTel SDK automatically via `Telemetry` YAML block. HTTP and zRPC traces are instrumented with zero code. Only Kafka requires manual propagation. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| `go.opentelemetry.io/otel/exporters/jaeger` package | Removed from OTel Go in 2023. Dead package. | `otlptracegrpc` exporter targeting Jaeger's OTLP port 4317 |
| `Batcher: jaeger` in go-zero Telemetry config | Uses the removed Jaeger exporter internally. May compile but targets Jaeger's deprecated Thrift HTTP collector (port 14268) which Jaeger v2 may drop. | `Batcher: otlpgrpc` with `Endpoint: jaeger:4317` |
| `bitnami/etcd` Docker image | Deprecated, moved to `bitnamilegacy` namespace | `gcr.io/etcd-development/etcd:v3.5.27` |
| Custom Prometheus middleware or metrics registration | go-zero DevServer already exposes `/metrics` on port 6470/6471/6472. Adding custom Prometheus setup creates duplicate metrics. | go-zero built-in DevServer (already configured in this project) |
| OpenTelemetry Collector as intermediary | Adds a 5th infrastructure component with complex config. Unnecessary for local dev where Jaeger v2 natively accepts OTLP. | Send OTLP directly from go-zero services to Jaeger |
| Zipkin as trace backend | No UI advantage over Jaeger for this use case; Jaeger is more widely adopted for Go microservices observability. | Jaeger |

---

## Stack Patterns by Feature

**Distributed tracing (HTTP and zRPC):**
- Add `Telemetry:` block to each service YAML
- No code changes — go-zero's REST and zRPC middleware auto-instruments spans
- go-zero's HTTP server creates a span per request; zRPC server and client propagate context via gRPC metadata

**Kafka trace propagation (manual):**
- Producer (url-api): Extract span from context, call `otel.GetTextMapPropagator().Inject(ctx, carrier)` where carrier writes to Kafka message headers
- Consumer (analytics-consumer): Read Kafka headers, call `otel.GetTextMapPropagator().Extract(ctx, carrier)` to recover span context
- Create child span from extracted context in consumer

**Etcd service discovery:**
- Replace `Target: dns:///localhost:8081` with `Etcd:` block in url-api's `AnalyticsRpc` config
- Add `Etcd:` block to analytics-rpc server config for registration
- No Go code changes — zRPC client/server detect etcd config and switch resolvers automatically

**Grafana dashboards:**
- Pre-build dashboard JSON targeting go-zero's standard metric names:
  - `http_server_requests_duration_ms_bucket` (latency histogram)
  - `http_server_requests_code_total` (status codes by path)
- Mount dashboard JSON into Grafana container via Docker volume at startup

---

## Version Compatibility

| Component | Version | Compatible With | Notes |
|-----------|---------|-----------------|-------|
| go-zero | v1.10.0 | OTel v1.35.0 (indirect dep) | Telemetry block activates OTel SDK already bundled |
| go-zero Etcd client | v3.5.15 (indirect) | etcd server v3.5.27 | Client and server minor version mismatch is safe in etcd 3.5.x |
| Jaeger | v2.15.0 | OTel OTLP | Jaeger v2 accepts OTLP gRPC on port 4317, OTLP HTTP on port 4318 |
| Prometheus | v3.9.1 | go-zero DevServer `/metrics` | Prometheus scrapes go-zero's standard Prometheus exposition format |
| Grafana | v12.3.3 | Prometheus v3.9.1 | Standard Prometheus datasource, fully compatible |

---

## Sources

- [go-zero core/trace/config.go](https://github.com/zeromicro/go-zero/blob/master/core/trace/config.go) — TelemetryConf struct fields and valid Batcher values (HIGH confidence, source verified)
- [go-zero monitoring docs](https://go-zero.dev/en/docs/tutorials/monitor/index) — DevServer auto-metrics, Telemetry YAML config (HIGH confidence)
- [go-zero gRPC client configuration](https://go-zero.dev/en/docs/tutorials/grpc/client/conn) — EtcdConf fields for service discovery (HIGH confidence)
- [Jaeger Getting Started v1.76](https://www.jaegertracing.io/docs/1.76/getting-started/) — Jaeger all-in-one ports and OTLP config (HIGH confidence)
- [Jaeger v2 CNCF announcement](https://www.cncf.io/blog/2024/11/12/jaeger-v2-released-opentelemetry-in-the-core/) — v2 architecture based on OTel Collector (HIGH confidence)
- [OTel Jaeger exporter deprecation](https://opentelemetry.io/blog/2023/jaeger-exporter-collector-migration/) — Jaeger exporter removed July 2023 (HIGH confidence)
- [Prometheus v3.9.1 release](https://github.com/prometheus/prometheus/releases/tag/v3.9.1) — v3.9.1 released 2026-01-07 (HIGH confidence)
- [Grafana What's New v12.3](https://grafana.com/docs/grafana/latest/whatsnew/whats-new-in-v12-3/) — v12.3.3 latest stable Feb 2026 (HIGH confidence)
- [Grafana Provisioning docs](https://grafana.com/docs/grafana/latest/administration/provisioning/) — auto-provisioning datasources and dashboards (HIGH confidence)
- [etcd releases](https://github.com/etcd-io/etcd/releases/) — v3.5.27 latest 3.5.x (HIGH confidence)
- [go-zero zRPC etcd discovery blog](https://community.ops.io/kevwan/implementing-service-discovery-for-microservices-5bcb) — YAML config examples for server and client (MEDIUM confidence)
- [Uptrace go-zero OTel guide](https://uptrace.dev/guides/opentelemetry-go-zero) — Telemetry struct fields verification (MEDIUM confidence)
- [OTel propagation package](https://pkg.go.dev/go.opentelemetry.io/otel/propagation) — W3C TraceContext propagation for Kafka (HIGH confidence)

---
*Stack research for: v3.0 observability + service discovery additions to go-zero microservices*
*Researched: 2026-02-22*
