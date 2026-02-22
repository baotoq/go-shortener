# Feature Research

**Domain:** Observability, Service Discovery, and Tech Debt for go-zero Microservices (v3.0 Production Readiness)
**Researched:** 2026-02-22
**Confidence:** HIGH

## Context: What Already Exists

This is a subsequent milestone. The URL shortener already ships v2.0 with three go-zero microservices (url-api, analytics-rpc, analytics-consumer), Kafka, PostgreSQL, Docker Compose, and CI. This research covers only the new v3.0 features: distributed tracing, Prometheus metrics, Grafana dashboards, Etcd service discovery, and tech debt cleanup.

**Critical prior finding:** go-zero already pulls OpenTelemetry and Prometheus as transitive dependencies — `go.mod` already contains `go.opentelemetry.io/otel`, `prometheus/client_golang`, and `go.etcd.io/etcd/client/v3`. The infrastructure is already in the dependency graph. These features need configuration and wiring, not new library additions.

## Feature Landscape

### Table Stakes (Users Expect These)

Features a production-ready observability stack must have. Missing these means the system cannot be monitored or debugged in production.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Telemetry config block in YAML per service** | go-zero's built-in OTel requires only YAML config changes — no code changes for HTTP and zRPC tracing | LOW | Add `Telemetry.Name`, `Telemetry.Endpoint`, `Telemetry.Sampler`, `Telemetry.Batcher` to each service YAML. go-zero auto-instruments HTTP handlers and gRPC calls when config present. |
| **Jaeger all-in-one in Docker Compose** | Trace visualization backend required; without it, spans are generated but never viewable | LOW | Use `jaegertracing/all-in-one` image with `COLLECTOR_OTLP_ENABLED=true`. Expose port 16686 (UI), 4317 (OTLP gRPC). Add to existing `docker-compose.yml`. |
| **Prometheus metrics endpoint already active** | go-zero auto-exposes `/metrics` on DevServer port (6470/6471/6472) — already running in current services | LOW | DevServer is already enabled in all three YAML configs. Prometheus just needs a scrape config pointing at those ports. |
| **Prometheus container in Docker Compose** | Metrics collection backend; without it, `/metrics` endpoints exist but are never scraped | LOW | Add `prom/prometheus` to Docker Compose with a `prometheus.yml` scrape config targeting all three DevServer ports. |
| **Grafana container in Docker Compose** | Dashboard visualization; Prometheus without Grafana is query-only | LOW | Add `grafana/grafana` to Docker Compose. Pre-configure Prometheus datasource via provisioning files. |
| **Trace context propagation through HTTP and zRPC** | Spans across url-api and analytics-rpc must be linked in Jaeger; otherwise each service shows isolated traces | LOW | go-zero automatically propagates W3C trace context in HTTP headers and gRPC metadata. No code changes needed when `Telemetry` config is set. |
| **Trace context propagation through Kafka boundary** | Without manual injection/extraction in Kafka message headers, the HTTP → Kafka → Consumer trace chain breaks | MEDIUM | go-queue does not auto-propagate OTel context. Requires manual: producer injects `otel.GetTextMapPropagator().Inject(ctx, kafkaHeaderCarrier)` into message headers; consumer extracts before processing. |
| **Etcd service discovery for zRPC** | Replace hardcoded `dns:///localhost:8081` in url-api config with etcd-based dynamic discovery for analytics-rpc | MEDIUM | Server: add `Etcd.Hosts` and `Etcd.Key` to analytics-rpc `zrpc.RpcServerConf` YAML. Client: replace `Target` with `Etcd` block in url-api `zrpc.RpcClientConf` YAML. Add etcd container to Docker Compose. |
| **Health check endpoints accessible** | go-zero auto-exposes `/healthz` on DevServer — already active; Docker Compose health checks can use them | LOW | Already present in all three YAML configs (`HealthPath: /healthz`). Verify Docker Compose `healthcheck` entries use these. |
| **Grafana dashboard for go-zero services** | Pre-built dashboard (ID 19909 on Grafana Labs) covers go-zero's built-in HTTP and RPC metric names | LOW | Import dashboard JSON. Uses go-zero's standard metric names: `http_server_requests_duration_ms`, `rpc_server_requests_duration_ms`, `rpc_client_requests_duration_ms`. |

### Differentiators (Competitive Advantage)

Features that make the observability stack genuinely useful rather than just present.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Custom business metrics via `core/metric`** | Count actual redirect events, Kafka publish failures, URL creation rates — not just HTTP status codes | MEDIUM | Use `metric.NewCounterVec` and `metric.NewHistogramVec` in logic layer. Demonstrates the distinction between framework auto-instrumentation and domain-level observability. |
| **End-to-end trace from HTTP to Kafka to DB** | Single Jaeger trace showing: redirect HTTP span → Kafka publish span → consumer enrichment span → PostgreSQL insert span | HIGH | Requires Kafka header propagation (table stakes above) plus adding manual span creation in consumer `Consume()` method. The OTel SDK is already in go.mod; spans use `otel.Tracer().Start(ctx, "operation")`. |
| **Grafana dashboard for Kafka consumer** | analytics-consumer uses `service.ServiceConf` (not `rest.RestConf`) so no HTTP metrics exist; a custom panel showing consumer throughput adds real value | MEDIUM | Consumer exposes DevServer at 6472 for Prometheus. Custom metrics for messages consumed/second and enrichment failures fill the gap. |
| **Prometheus alerting rules** | Defining alert thresholds (e.g., `rpc_server_requests_code_total{code!="OK"} > 5`) moves from "dashboard with colored lines" to "system that tells you when something is wrong" | MEDIUM | Add `prometheus.yml` alerting rules and route to a null receiver for local dev. Not a hard requirement but demonstrates production intent. |
| **Kafka topic retention configuration** | Currently defaults; configure `click-events` topic with explicit retention.ms and segment settings to prevent unbounded disk growth | LOW | Add Kafka topic config via environment variable in Docker Compose: `KAFKA_LOG_RETENTION_HOURS=24`. Known tech debt item from PROJECT.md. |
| **Automated E2E test** | Currently no automated test verifies the full flow (shorten → redirect → Kafka → consumer → analytics); adding one ensures observability additions don't break the critical path | HIGH | Use testcontainers-go to spin up all services and verify full click event pipeline. Known tech debt item from PROJECT.md. |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **OpenTelemetry Collector as middleware layer** | "Industry standard, decouples services from backends" | Adds another container, another config file, another failure mode — in a 3-service local dev setup where Jaeger can receive OTLP directly. The complexity is not justified. | Send OTLP directly from services to Jaeger's native OTLP receiver (port 4317). Skip the Collector until Kubernetes or multiple backends are needed. |
| **Zipkin instead of Jaeger** | "Lighter weight" | go-zero supports both, but Jaeger has native OTLP support (post v1.35) and is the CNCF standard. Using the Zipkin batcher (`Batcher: zipkin`) requires the legacy Thrift endpoint, not OTLP. | Use Jaeger with `Batcher: otlpgrpc`. Same effort, more future-proof. |
| **Sampling rate below 1.0 in development** | "Performance" | In local dev with low traffic, 100% sampling is correct. Reduced sampling hides traces and makes debugging impossible during development. | Set `Sampler: 1.0` for dev. Only reduce in production when trace volume is measured. |
| **Consul instead of etcd for service discovery** | "Consul has a better UI" | go-zero has native etcd integration built in (`discov.EtcdConf` is part of `zrpc.RpcServerConf`). Consul requires the third-party `go-zero/contrib` package. Using etcd avoids the extra dependency. | etcd. It's already in go.mod as a transitive dependency of go-zero. |
| **Redis for service discovery** | "We already have Redis available" | go-zero does not support Redis-based service discovery. It supports etcd, direct connection, and Kubernetes endpoints only. | etcd for dynamic discovery. Direct connection mode if discovery is not needed (acceptable for fixed-topology Docker Compose). |
| **Full distributed tracing for every Kafka message header** | "Capture all analytics consumer spans with baggage" | go-queue's `Consume(ctx, key, val)` already receives a context. The minimal implementation is extracting the trace context from message headers into that context, then passing it through. Baggage adds no value here. | Extract `traceparent` header into context, link consumer span to producer span. Done. No baggage. |
| **Remove IncrementClickCount via a big refactor** | "Clean up the dead code properly" | `IncrementClickCount` is a dead method in the url model — it was replaced by the Kafka consumer approach. Removing it is low-risk: delete the method from `urlsmodel.go`, verify no callers, done. The anti-feature is treating this as a big effort. | Delete dead method, run `go build ./...`, done. One file, 10 lines. |
| **Grafana datasource configured via UI** | "Just click through the UI after starting Docker Compose" | Manual UI setup is not reproducible, breaks after `docker-compose down -v`, and can't be committed to version control. | Use Grafana provisioning: `grafana/provisioning/datasources/prometheus.yml` and `grafana/provisioning/dashboards/` directory. These are mounted as volumes and auto-applied on container start. |

## Feature Dependencies

```
Distributed Tracing (HTTP + zRPC)
    └──requires──> Telemetry config in all 3 service YAMLs
    └──requires──> Jaeger container in Docker Compose
    └──independent from──> Kafka trace propagation

Kafka Trace Propagation
    └──requires──> Distributed Tracing (must be initialized first)
    └──requires──> Manual header inject in url-api redirect logic
    └──requires──> Manual header extract in analytics-consumer Consume()
    └──enhances──> Distributed Tracing (completes the trace chain)

Prometheus Metrics Collection
    └──requires──> Prometheus container in Docker Compose
    └──requires──> Prometheus scrape config targeting DevServer ports
    └──independent from──> Distributed Tracing (separate signals)
    └──DevServer already enabled──> in url-api (6470), analytics-rpc (6471), analytics-consumer (6472)

Grafana Dashboards
    └──requires──> Prometheus (data must be collected first)
    └──requires──> Grafana container in Docker Compose
    └──requires──> Prometheus datasource provisioning
    └──independent from──> Distributed Tracing

Custom Business Metrics
    └──requires──> Prometheus collection running
    └──enhances──> Grafana Dashboards (custom panels for business metrics)

Etcd Service Discovery
    └──requires──> Etcd container in Docker Compose
    └──requires──> analytics-rpc: Etcd block in YAML (for registration)
    └──requires──> url-api: Etcd block in RpcClientConf (replacing Target)
    └──independent from──> Distributed Tracing
    └──independent from──> Prometheus Metrics

Tech Debt: Dead Code Removal
    └──independent from──> All observability features
    └──requires──> Verify no callers before deletion

Tech Debt: Consumer Coverage
    └──independent from──> All observability features
    └──requires──> Additional test cases for resolveCountry, resolveDeviceType, resolveTrafficSource

Tech Debt: E2E Test
    └──requires──> All three services buildable
    └──enhances──> Confidence when adding observability features
    └──requires──> testcontainers-go orchestration across all services
```

### Dependency Notes

- **Tracing depends on YAML config, not code:** go-zero's `rest.RestConf` and `zrpc.RpcServerConf` both embed `trace.Config`. Adding the `Telemetry` block to YAML auto-enables tracing for HTTP handlers and gRPC calls. No code changes in handlers or logic.
- **Kafka propagation requires code:** go-queue does not auto-propagate OTel context. The producer side (url-api redirect logic) must inject headers; the consumer side must extract. This is the only feature requiring Go code changes for tracing.
- **Etcd is YAML-only for server and client:** `RpcServerConf.Etcd` accepts `Hosts` and `Key` for registration. `RpcClientConf.Etcd` accepts the same for discovery. Remove `Target: dns:///localhost:8081` when switching to etcd.
- **DevServer is already running:** All three YAML configs already have `DevServer.Enabled: true` and `DevServer.Port` set (6470, 6471, 6472). Prometheus just needs a scrape config; no service changes needed.
- **Grafana provisioning must precede dashboard import:** Mount `provisioning/` directory before adding dashboard JSON, or the datasource won't exist when the dashboard loads.
- **E2E test is the most complex tech debt item:** Spawning all three services with testcontainers-go requires network coordination and startup ordering. Should be a separate phase/plan from dead code removal.

## MVP Definition

### Launch With (v3.0 Core Observability)

The minimum that makes the system "production observable" — traces you can follow, metrics you can query, dashboards you can view.

- [ ] **Telemetry config in all 3 service YAMLs** — enables HTTP and zRPC auto-tracing with one config block per service
- [ ] **Jaeger in Docker Compose** — without this, traces are generated but invisible
- [ ] **Trace context propagated through Kafka** — without this, the most interesting trace (redirect → analytics) is broken
- [ ] **Prometheus in Docker Compose with scrape config** — without this, the existing `/metrics` endpoints serve no purpose
- [ ] **Grafana in Docker Compose with provisioning** — dashboard import via UI is not reproducible; provisioning makes it committed to version control
- [ ] **Grafana dashboard for go-zero (dashboard 19909)** — covers built-in HTTP/RPC metrics out of the box; avoid building custom panels from scratch
- [ ] **Etcd in Docker Compose + analytics-rpc registration + url-api discovery** — replaces hardcoded DNS, teaches dynamic service discovery pattern

### Add After Core Working (v3.0 Enhancements)

- [ ] **Custom business metrics** — add after built-in metrics are confirmed working; CounterVec for redirects and URL creation events
- [ ] **Kafka topic retention config** — low effort, add to Docker Compose KAFKA_ env vars
- [ ] **Dead code removal (IncrementClickCount)** — isolated change, low risk, minimal effort
- [ ] **Consumer coverage improvement** — add unit tests for the three enrichment helpers (resolveCountry, resolveDeviceType, resolveTrafficSource)

### Future Consideration (Post v3.0)

- [ ] **Automated E2E test** — high complexity, testcontainers-go across all three services; warrants its own research phase
- [ ] **Prometheus alerting rules** — adds operational value but not needed for a learning project milestone
- [ ] **Consumer-specific Grafana panels** — custom panels for Kafka throughput; go-zero dashboard 19909 covers HTTP/RPC but not consumer metrics

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Telemetry YAML config (HTTP + zRPC tracing) | HIGH | LOW | P1 |
| Jaeger Docker Compose | HIGH | LOW | P1 |
| Prometheus Docker Compose + scrape config | HIGH | LOW | P1 |
| Grafana Docker Compose + provisioning | HIGH | LOW | P1 |
| Grafana dashboard import (19909) | HIGH | LOW | P1 |
| Kafka trace context propagation | HIGH | MEDIUM | P1 |
| Etcd service discovery | HIGH | MEDIUM | P1 |
| Custom business metrics | MEDIUM | MEDIUM | P2 |
| Kafka topic retention | LOW | LOW | P2 |
| Dead code removal | LOW | LOW | P2 |
| Consumer test coverage | MEDIUM | LOW | P2 |
| Automated E2E test | HIGH | HIGH | P3 |
| Prometheus alerting rules | MEDIUM | MEDIUM | P3 |

**Priority key:**
- P1: Must have for milestone — core observability and service discovery
- P2: Should have — tech debt and metric quality
- P3: Nice to have — future milestone consideration

## go-zero Built-In Observability: What's Already Wired

Understanding what go-zero provides automatically versus what requires code is critical for scoping this milestone correctly.

### Auto-Instrumented (Config Only, No Code)

| Capability | Trigger | Metric / Span Names |
|-----------|---------|---------------------|
| HTTP request duration | `Telemetry` block in YAML | `http_server_requests_duration_ms{path=...}` |
| HTTP status code counter | `Telemetry` block in YAML | `http_server_requests_code_total{path=..., code=...}` |
| zRPC server duration | `Telemetry` block in YAML | `rpc_server_requests_duration_ms{method=...}` |
| zRPC server error counter | `Telemetry` block in YAML | `rpc_server_requests_code_total{method=..., code=...}` |
| zRPC client duration | `Telemetry` block in YAML | `rpc_client_requests_duration_ms{method=...}` |
| zRPC client error counter | `Telemetry` block in YAML | `rpc_client_requests_code_total{method=..., code=...}` |
| SQL query duration | automatic | `sql_client_requests_duration_ms` |
| SQL error counter | automatic | `sql_client_requests_error_total` |
| HTTP trace spans | `Telemetry` block in YAML | spans named by route |
| zRPC trace spans | `Telemetry` block in YAML | spans named by method |
| W3C context propagation (HTTP → zRPC) | `Telemetry` block in YAML | automatic header injection/extraction |

### Requires Manual Code

| Capability | What's Needed | Why Not Auto |
|-----------|--------------|--------------|
| Kafka trace context propagation | Inject `traceparent` header in producer; extract in consumer | go-queue does not instrument Kafka messages |
| Custom business metrics | `metric.NewCounterVec(...)` + `.Inc()` calls in logic layer | Domain-specific, framework cannot know what to count |
| Consumer span creation | `otel.Tracer().Start(ctx, "consume-click-event")` in `Consume()` | Consumer is not an HTTP handler or gRPC method |

### Telemetry YAML Block (All Three Services)

```yaml
Telemetry:
  Name: url-api          # service name in Jaeger
  Endpoint: jaeger:4317  # OTLP gRPC endpoint (Docker network name)
  Sampler: 1.0           # 100% sampling in dev
  Batcher: otlpgrpc      # OTLP over gRPC (Jaeger native)
```

### Etcd Configuration (analytics-rpc server)

```yaml
# analytics-rpc/etc/analytics-docker.yaml
Name: analytics-rpc
ListenOn: 0.0.0.0:8081
Etcd:
  Hosts:
    - etcd:2379
  Key: analytics-rpc
```

### Etcd Configuration (url-api client)

```yaml
# url-api/etc/url-docker.yaml
AnalyticsRpc:
  Etcd:
    Hosts:
      - etcd:2379
    Key: analytics-rpc
  NonBlock: true
  Timeout: 2000
  # Remove: Target: dns:///analytics-rpc:8081
```

## Sources

- [Monitoring | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/monitor/index) — DevServer metrics/health endpoints, built-in metric names (HIGH confidence)
- [OpenTelemetry Go-Zero monitoring | Uptrace](https://uptrace.dev/guides/opentelemetry-go-zero) — Telemetry YAML config fields, batcher options (MEDIUM confidence, verified against go-zero source)
- [Service Connection | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/grpc/client/conn) — zRPC client etcd vs direct connection modes (HIGH confidence)
- [Discovery | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/component/discovery/) — Etcd registration/discovery pattern (HIGH confidence)
- [Service Configuration | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/grpc/server/configuration) — RpcServerConf Etcd fields (HIGH confidence)
- [Go-Zero | Grafana Labs (Dashboard 19909)](https://grafana.com/grafana/dashboards/19909-go-zero/) — Pre-built go-zero Grafana dashboard (MEDIUM confidence)
- [Metric | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/component/metric/) — core/metric package: CounterVec, GaugeVec, HistogramVec (HIGH confidence)
- [How to Propagate Trace Context Across Kafka Producers and Consumers](https://oneuptime.com/blog/post/2026-02-06-propagate-trace-context-kafka-producers-consumers/view) — Kafka header injection/extraction pattern in Go (MEDIUM confidence)
- [Jaeger Getting Started](https://www.jaegertracing.io/docs/1.76/getting-started/) — all-in-one Docker image, OTLP ports (HIGH confidence)
- go.mod analysis — confirmed `go.opentelemetry.io/otel`, `prometheus/client_golang`, `go.etcd.io/etcd/client/v3` already in dependency graph (HIGH confidence, direct observation)

---
*Feature research for: v3.0 Production Readiness — Observability, Service Discovery, Tech Debt*
*Researched: 2026-02-22*
