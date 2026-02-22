# Architecture Research: v3.0 Observability & Service Discovery

**Domain:** URL Shortener Microservices — Observability Stack Integration
**Researched:** 2026-02-22
**Confidence:** HIGH (go-zero built-in features), MEDIUM (Kafka trace propagation — manual instrumentation required)

---

## What This Document Covers

This document supersedes the v2.0 ARCHITECTURE.md for the purpose of v3.0 planning. It focuses exclusively on how OpenTelemetry tracing, Prometheus metrics, Grafana dashboards, and Etcd service discovery integrate into the existing 3-service go-zero architecture. The existing Handler → Logic → Model pattern, ServiceContext DI, and Docker Compose structure are taken as fixed context.

---

## Existing System Snapshot (v2.0 Baseline)

```
┌─────────────────────────────────────────────────────────────────────┐
│  Client                                                              │
│     │ HTTP :8080                                                     │
├─────▼────────────────────────────────────────────────────────────────┤
│  url-api (REST)           analytics-rpc (zRPC)  analytics-consumer   │
│  ┌─────────────────┐      ┌─────────────────┐   ┌──────────────────┐ │
│  │ handler/        │─gRPC→│ server/         │   │ mqs/             │ │
│  │ logic/          │      │ logic/          │   │ clickeventconsumer│ │
│  │ model/          │      │ model/          │   │ model/ (clicks)  │ │
│  │ svcCtx          │      │ svcCtx          │   │ svcCtx           │ │
│  └────┬────────────┘      └────────────────-┘   └────────▲─────────┘ │
│       │ Kafka push (fire-and-forget)                      │ consume  │
│       │                  Kafka topic: click-events         │          │
├───────┼──────────────────────────────────────────────────┼───────────┤
│  Infrastructure                                                       │
│  PostgreSQL :5433   Kafka :9092   [Etcd: not yet]                    │
└─────────────────────────────────────────────────────────────────────┘

Direct zRPC: url-api → analytics-rpc via dns:///localhost:8081
DevServer metrics endpoints already exist:
  url-api           :6470/metrics
  analytics-rpc     :6471/metrics
  analytics-consumer :6472/metrics
```

## Target System (v3.0)

```
┌────────────────────────────────────────────────────────────────────────────┐
│  Client                                                                     │
│     │ HTTP :8080  (traceparent header propagated)                           │
├─────▼────────────────────────────────────────────────────────────────────── │
│  url-api                   analytics-rpc           analytics-consumer        │
│  ┌───────────────────┐     ┌──────────────────┐    ┌─────────────────────┐  │
│  │ Telemetry → Jaeger│     │ Telemetry → Jaeger│    │ Telemetry → Jaeger  │  │
│  │ :6470/metrics     │     │ :6471/metrics     │    │ :6472/metrics       │  │
│  │                   │─gRPC│                   │    │ manual OTel spans   │  │
│  │ Etcd client       │     │ Etcd registration │    │ trace header in msg │  │
│  │ (resolver)        │     │ (auto via config) │    │                     │  │
│  └──────────────────-┘     └───────────────────┘    └─────────────────────┘  │
│         │                                                     ▲               │
│         │ Kafka push + traceparent in message header          │ extract ctx   │
│         └─────────────────────────────────────────────────────┘               │
├────────────────────────────────────────────────────────────────────────────── │
│  New Infrastructure                                                           │
│  Etcd :2379    Jaeger :16686/:4317    Prometheus :9090    Grafana :3000       │
│  (existing: PostgreSQL :5433, Kafka :9092)                                    │
└────────────────────────────────────────────────────────────────────────────── │
```

---

## Component Integration Details

### 1. OpenTelemetry Tracing via go-zero Telemetry Block

**How it works:** go-zero's `RestConf`, `RpcServerConf`, and `service.ServiceConf` all embed a `Telemetry trace.Config` field marked `json:",optional"`. When present in YAML, go-zero auto-initializes the OTel SDK and instruments HTTP handlers and gRPC calls automatically — no code changes required for those boundaries.

**Configuration (add to each service YAML):**

```yaml
# services/url-api/etc/url.yaml  (add to existing config)
Telemetry:
  Name: url-api
  Endpoint: jaeger:4317
  Sampler: 1.0
  Batcher: otlpgrpc
```

```yaml
# services/analytics-rpc/etc/analytics.yaml
Telemetry:
  Name: analytics-rpc
  Endpoint: jaeger:4317
  Sampler: 1.0
  Batcher: otlpgrpc
```

```yaml
# services/analytics-consumer/etc/consumer.yaml
Telemetry:
  Name: analytics-consumer
  Endpoint: jaeger:4317
  Sampler: 1.0
  Batcher: otlpgrpc
```

**What is automatically instrumented (no code changes):**
- All HTTP handlers in url-api — incoming request spans with method, path, status
- All zRPC client calls from url-api to analytics-rpc — outgoing gRPC spans
- All zRPC server handlers in analytics-rpc — incoming gRPC spans
- analytics-consumer's `c.MustSetUp()` activates the OTel SDK via ServiceConf

**What requires manual instrumentation:**
- Kafka publish: inject `traceparent` header into Kafka message headers when pushing from url-api redirect logic
- Kafka consume: extract `traceparent` from Kafka message headers in analytics-consumer `Consume()` before processing

### 2. Prometheus Metrics (Already Wired — Just Add Scraping)

**Critical finding:** go-zero's DevServer is already configured and running in all three services. The `/metrics` endpoints at ports 6470, 6471, and 6472 already expose Prometheus metrics. No code changes are needed.

**What go-zero exposes automatically:**

| Metric | Type | Labels |
|--------|------|--------|
| `http_server_requests_duration_ms` | Histogram | path, method |
| `http_server_requests_code_total` | Counter | path, method, code |
| Go runtime metrics (GC, goroutines, heap) | Gauge/Counter | — |
| gRPC request duration | Histogram | method |
| gRPC error count | Counter | method, code |

**Only work needed:** Add a `prometheus.yml` scrape config and wire it in Docker Compose. The services already expose the data.

### 3. Grafana Dashboards

Grafana needs a Prometheus data source and dashboard definitions. The dashboard queries will target the go-zero metric names listed above. Dashboards are provisioned via YAML files mounted into the Grafana container.

**New files to create:**
- `infra/grafana/provisioning/datasources/prometheus.yaml` — Prometheus data source
- `infra/grafana/provisioning/dashboards/dashboard.yaml` — dashboard loader config
- `infra/grafana/dashboards/url-shortener.json` — the actual dashboard JSON (latency, throughput, error rate panels)

### 4. Etcd Service Discovery

**How go-zero Etcd discovery works:**

Server side (analytics-rpc): add `Etcd` block under `RpcServerConf`. On startup, go-zero automatically registers the service address in Etcd under the given key and sends periodic keep-alives. On shutdown, it deregisters.

Client side (url-api): replace the `Target: dns:///localhost:8081` in `AnalyticsRpc` config with an `Etcd` block using the same key. go-zero's gRPC resolver queries Etcd for live nodes and applies round-robin load balancing automatically.

**Configuration changes:**

```yaml
# services/analytics-rpc/etc/analytics.yaml — add Etcd block
Etcd:
  Hosts:
    - etcd:2379
  Key: analytics-rpc
```

```yaml
# services/url-api/etc/url.yaml — replace Target with Etcd
AnalyticsRpc:
  # Remove: Target: dns:///localhost:8081
  Etcd:
    Hosts:
      - etcd:2379
    Key: analytics-rpc
  NonBlock: true
  Timeout: 2000
```

**No Go code changes required** — the config struct already accepts `zrpc.RpcClientConf` which has the `Etcd discov.EtcdConf` field built in. The existing `zrpc.MustNewClient(c.AnalyticsRpc)` call in url-api's ServiceContext reads the Etcd config automatically.

---

## Data Flows with Observability

### HTTP → gRPC Flow (Automatic)

```
Client HTTP Request (GET /{code})
    │ W3C traceparent header injected by client (or started fresh)
    ▼
url-api: RedirectHandler (go-zero auto-starts span: "HTTP GET /{code}")
    │ context.Context carries trace
    ▼
RedirectLogic.Redirect()
    │ ctx propagated through all logic calls
    ▼
GetLinkDetail path: AnalyticsRpc.GetClickCount(ctx, ...) via zRPC
    │ go-zero auto-injects gRPC metadata with trace context
    ▼
analytics-rpc: GetClickCountLogic (go-zero auto-starts child span)
    │ PostgreSQL query
    ▼
Return to url-api → HTTP 302 Response

All spans appear in Jaeger as one connected trace.
```

### HTTP → Kafka Flow (Manual Instrumentation in Redirect Logic)

```
url-api: RedirectLogic.Redirect() — span is active in ctx
    │
    ▼
threading.GoSafe(func() {
    // MANUAL: extract span context from ctx
    // MANUAL: inject traceparent into Kafka message headers
    // push ClickEvent JSON with headers to Kafka
})

analytics-consumer: Consume(ctx, key, val string)
    // MANUAL: extract traceparent from Kafka record headers
    // MANUAL: create new span with link to producer span
    // process click, enrich, insert into PostgreSQL
    // end span
```

The producer-consumer relationship uses a **span link** (not parent-child) because Kafka is async. The Jaeger UI shows the producer trace and consumer trace as separate traces with a visible link between them.

**Implementation approach for Kafka propagation:**

go-queue's `kq.Pusher` does not expose Kafka message headers. The workaround is to include the trace context inside the JSON payload of the `ClickEvent` struct by adding a `TraceContext string` field, or to switch to using the underlying `segmentio/kafka-go` producer directly for the Kafka push in redirect logic to get header access. The JSON field approach is simpler and avoids the `kq` wrapper limitation.

Simplest viable approach — add `TraceContext` to `ClickEvent`:
```go
// common/events/events.go
type ClickEvent struct {
    ShortCode    string `json:"short_code"`
    Timestamp    int64  `json:"timestamp"`
    IP           string `json:"ip"`
    UserAgent    string `json:"user_agent"`
    Referer      string `json:"referer"`
    TraceContext string `json:"trace_context,omitempty"` // W3C traceparent
}
```

Producer (redirect logic) serializes the trace context into this field. Consumer extracts it and reconstructs the span link. This keeps `kq.Pusher` intact.

---

## New Docker Compose Services

### Additions to `docker-compose.yml`

```yaml
  etcd:
    image: bitnami/etcd:3.5
    environment:
      ALLOW_NONE_AUTHENTICATION: "yes"
    ports:
      - "2379:2379"
    healthcheck:
      test: ["CMD", "etcdctl", "endpoint", "health"]
      interval: 5s
      timeout: 5s
      retries: 5

  jaeger:
    image: jaegertracing/all-in-one:1.57
    environment:
      COLLECTOR_OTLP_ENABLED: "true"
    ports:
      - "16686:16686"  # Jaeger UI
      - "4317:4317"    # OTLP gRPC receiver
    healthcheck:
      test: ["CMD-SHELL", "wget -q --spider http://localhost:14269/ || exit 1"]
      interval: 5s
      timeout: 5s
      retries: 5

  prometheus:
    image: prom/prometheus:v2.51.0
    volumes:
      - ./infra/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro
    ports:
      - "9090:9090"
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.enable-lifecycle'

  grafana:
    image: grafana/grafana:10.4.0
    volumes:
      - ./infra/grafana/provisioning:/etc/grafana/provisioning:ro
      - ./infra/grafana/dashboards:/var/lib/grafana/dashboards:ro
      - grafana-data:/var/lib/grafana
    ports:
      - "3000:3000"
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
      GF_USERS_ALLOW_SIGN_UP: "false"
    depends_on:
      - prometheus
```

### Updated `depends_on` for existing services

```yaml
  analytics-rpc:
    depends_on:
      postgres:
        condition: service_healthy
      etcd:                       # NEW: register with Etcd on start
        condition: service_healthy
      jaeger:                     # NEW: send traces to Jaeger
        condition: service_started

  url-api:
    depends_on:
      postgres:
        condition: service_healthy
      kafka:
        condition: service_healthy
      analytics-rpc:
        condition: service_started
      etcd:                       # NEW: discover analytics-rpc via Etcd
        condition: service_healthy
      jaeger:
        condition: service_started

  analytics-consumer:
    depends_on:
      postgres:
        condition: service_healthy
      kafka:
        condition: service_healthy
      jaeger:
        condition: service_started
```

### New `infra/` Directory Structure

```
infra/
├── prometheus/
│   └── prometheus.yml          # Scrape config for 3 DevServer /metrics endpoints
└── grafana/
    ├── provisioning/
    │   ├── datasources/
    │   │   └── prometheus.yaml  # Auto-configure Prometheus data source
    │   └── dashboards/
    │       └── dashboard.yaml   # Point Grafana at /var/lib/grafana/dashboards
    └── dashboards/
        └── url-shortener.json   # Dashboard: latency p95, req/s, error rate, RPC latency
```

### Prometheus Scrape Configuration

```yaml
# infra/prometheus/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: url-api
    static_configs:
      - targets: ['url-api:6470']
    metrics_path: /metrics

  - job_name: analytics-rpc
    static_configs:
      - targets: ['analytics-rpc:6471']
    metrics_path: /metrics

  - job_name: analytics-consumer
    static_configs:
      - targets: ['analytics-consumer:6472']
    metrics_path: /metrics
```

---

## Component Change Summary

| Component | Change Type | What Changes | Files Affected |
|-----------|-------------|--------------|----------------|
| url-api YAML | Config change | Add Telemetry block, change AnalyticsRpc to Etcd | `etc/url.yaml` |
| analytics-rpc YAML | Config change | Add Telemetry block, add Etcd registration | `etc/analytics.yaml` |
| analytics-consumer YAML | Config change | Add Telemetry block | `etc/consumer.yaml` |
| common/events | Code change | Add TraceContext field to ClickEvent | `common/events/events.go` |
| redirect logic | Code change | Inject trace context into ClickEvent before push | `services/url-api/internal/logic/redirect/redirectlogic.go` |
| consumer | Code change | Extract trace context, create span link | `services/analytics-consumer/internal/mqs/clickeventconsumer.go` |
| docker-compose.yml | New services | Add etcd, jaeger, prometheus, grafana | `docker-compose.yml` |
| infra/ | New files | prometheus.yml, grafana provisioning, dashboard JSON | `infra/` (new) |
| volumes | New volume | grafana-data for Grafana persistence | `docker-compose.yml` |

**No changes to:** Handler files, model files, generated code, ServiceContext struct (no new dependencies needed for tracing or metrics — go-zero handles them via config).

---

## Architectural Patterns

### Pattern 1: go-zero Telemetry Config (Zero-Code Tracing)

**What:** Adding a `Telemetry` YAML block to any go-zero service (RestConf, RpcServerConf, or ServiceConf) automatically enables OTel tracing with no code changes. The framework instruments HTTP and gRPC boundaries automatically.

**When to use:** All three services. This is the primary tracing integration point.

**Trade-offs:** Automatic for HTTP/gRPC; Kafka boundaries require manual span injection. OTel context is propagated through `context.Context` automatically once initialized.

**Example:**
```yaml
Telemetry:
  Name: url-api
  Endpoint: jaeger:4317   # Docker Compose service name
  Sampler: 1.0            # 100% sampling for dev; reduce in production
  Batcher: otlpgrpc       # Jaeger v1.35+ natively accepts OTLP
```

### Pattern 2: Span Link for Async Kafka Boundary

**What:** When a producer sends a message to Kafka and a consumer processes it asynchronously, the trace context is propagated via a field in the message payload (or headers). The consumer creates a new root span and adds a link to the producer span rather than making it a child.

**When to use:** The url-api → Kafka → analytics-consumer boundary.

**Trade-offs:** Span links are less intuitive than parent-child in Jaeger UI, but correctly represent the async relationship. Producer and consumer traces appear as separate traces connected by a link.

**Example:**
```go
// Producer (redirect logic) — inject W3C traceparent
prop := otel.GetTextMapPropagator()
carrier := propagation.MapCarrier{}
prop.Inject(ctx, carrier)
clickEvent.TraceContext = carrier["traceparent"]

// Consumer — extract and create link
carrier := propagation.MapCarrier{"traceparent": event.TraceContext}
remoteCtx := prop.Extract(context.Background(), carrier)
remoteSpan := trace.SpanFromContext(remoteCtx)
_, span := tracer.Start(context.Background(), "consume-click-event",
    trace.WithLinks(trace.Link{SpanContext: remoteSpan.SpanContext()}),
)
defer span.End()
```

### Pattern 3: Etcd Registration via go-zero Config (Zero-Code Discovery)

**What:** Adding an `Etcd` block to `RpcServerConf` causes go-zero to register the service at startup and deregister at shutdown. The client reads the same key from Etcd to get live endpoints with load balancing. No code changes required.

**When to use:** analytics-rpc server registration + url-api client discovery.

**Trade-offs:** Requires Etcd infrastructure. Etcd failure is handled gracefully — go-zero caches the last known node list and continues operating. For Docker Compose single-instance, this is equivalent to DNS target but enables future horizontal scaling.

**Example:**
```yaml
# Server: analytics-rpc/etc/analytics.yaml
Etcd:
  Hosts: [etcd:2379]
  Key: analytics-rpc        # Service registration key

# Client: url-api/etc/url.yaml
AnalyticsRpc:
  Etcd:
    Hosts: [etcd:2379]
    Key: analytics-rpc      # Must match server registration key
  NonBlock: true
  Timeout: 2000
```

### Pattern 4: Grafana Provisioning (Config-as-Code Dashboards)

**What:** Grafana supports automatic provisioning of data sources and dashboards from YAML/JSON files mounted at startup. No manual UI configuration needed.

**When to use:** Docker Compose environments where dashboard state must be reproducible.

**Trade-offs:** Dashboard JSON is verbose (100-500 lines). Use Grafana's UI to design the dashboard, then export JSON and commit it.

---

## Data Flow: Observability Paths

### Trace Collection Path

```
url-api, analytics-rpc, analytics-consumer
    │ OTLP gRPC (port 4317, inside Docker network)
    ▼
jaeger:4317 (collector)
    │ stored in memory (all-in-one dev mode)
    ▼
Jaeger UI http://localhost:16686
    - Search by service name
    - View full trace across HTTP → gRPC boundary
    - View span links to Kafka consumer traces
```

### Metrics Collection Path

```
url-api:6470/metrics
analytics-rpc:6471/metrics        ← Prometheus scrapes every 15s
analytics-consumer:6472/metrics
    │
    ▼
prometheus:9090 (storage + query)
    │
    ▼
grafana:3000 (visualization)
    - Dashboard: http_server_requests_duration_ms p95
    - Dashboard: http_server_requests_code_total (error rate)
    - Dashboard: gRPC request duration
```

### Service Discovery Path

```
analytics-rpc starts
    → registers "analytics-rpc" → etcd:2379/analytics-rpc → {host:port}

url-api starts
    → reads "analytics-rpc" from etcd:2379
    → gets [{analytics-rpc:8081}]
    → zRPC client dials analytics-rpc:8081 with round-robin LB

analytics-rpc scales to 2 instances
    → both register in etcd under same key
    → url-api resolver detects both, load-balances across them
```

---

## Build Order and Dependencies

Phase ordering is driven by these dependency constraints:

1. **Etcd before service discovery migration** — analytics-rpc must be able to register before url-api changes its connection target. Both YAML files change together, but Etcd must be in Docker Compose before either is tested.

2. **Jaeger before Telemetry config** — Services with `Endpoint: jaeger:4317` will log connection errors if Jaeger is not running. Add Jaeger to Docker Compose before adding Telemetry blocks to service YAMLs.

3. **Telemetry config before Kafka trace propagation** — The OTel SDK must be initialized (via Telemetry block) before manual `otel.GetTextMapPropagator()` calls work in redirect logic and consumer.

4. **Prometheus scrape config before Grafana dashboard** — Dashboard panels reference Prometheus metric names that only appear after Prometheus is scraping. Build dashboard last.

**Recommended phase ordering:**
```
Phase 1: Tech debt cleanup (no dependencies, can go first)
Phase 2: Etcd service discovery (infrastructure + config only, no code)
Phase 3: Distributed tracing (Jaeger + Telemetry config + Kafka manual instrumentation)
Phase 4: Prometheus + Grafana (scrape config, dashboard, new Docker services)
```

---

## Integration Points

### New External Services

| Service | Image | Integration Pattern | Notes |
|---------|-------|---------------------|-------|
| Jaeger | `jaegertracing/all-in-one:1.57` | OTLP gRPC receiver at :4317 | `COLLECTOR_OTLP_ENABLED=true` required; UI at :16686 |
| Prometheus | `prom/prometheus:v2.51.0` | Pull scrape from DevServer :6470/:6471/:6472 | Config file mounted at `/etc/prometheus/prometheus.yml` |
| Grafana | `grafana/grafana:10.4.0` | Provisioned from `infra/grafana/` | Auto-wires Prometheus data source |
| Etcd | `bitnami/etcd:3.5` | go-zero native via `discov.EtcdConf` | `ALLOW_NONE_AUTHENTICATION=yes` for dev |

### Modified Existing Boundaries

| Boundary | Current | After v3.0 | Change |
|----------|---------|------------|--------|
| url-api → analytics-rpc | `dns:///localhost:8081` | Etcd resolver, key `analytics-rpc` | YAML only |
| url-api HTTP span | No tracing | Auto-instrumented via Telemetry block | YAML only |
| url-api → analytics-rpc gRPC span | No tracing | Auto-instrumented via Telemetry block | YAML only |
| url-api → Kafka | No trace context | TraceContext field in ClickEvent JSON | Code change |
| analytics-consumer | No tracing | Span link from extracted TraceContext | Code change |
| All services → Prometheus | DevServer active but not scraped | Prometheus scrapes every 15s | New infra file |

### Internal Boundaries (Unchanged)

| Boundary | Communication | Notes |
|----------|---------------|-------|
| Handler ↔ Logic | Direct method call + context.Context | OTel context flows automatically via ctx |
| Logic ↔ Model | Interface call + context.Context | OTel context flows automatically via ctx |
| ServiceContext | DI container | No new dependencies added for tracing/metrics |

---

## Scaling Considerations

| Scale | Observability Adjustments |
|-------|--------------------------|
| Single-instance dev | All-in-one Jaeger (in-memory), Prometheus with local storage, Grafana local — sufficient |
| Multiple instances | Sampler: 0.1 (10% sampling), Jaeger with persistent backend (Cassandra/ES), Prometheus remote write |
| Production | Replace Jaeger all-in-one with Jaeger Collector + Query separate; add alerting rules to Prometheus; Grafana with LDAP auth |

### Scaling Priorities

1. **First adjustment:** Reduce Sampler from 1.0 to 0.1 before load testing — 100% sampling creates significant Jaeger write load.
2. **Second adjustment:** Jaeger all-in-one loses traces on restart — switch to Jaeger with persistent storage (Cassandra or Elasticsearch) before treating traces as persistent.

---

## Anti-Patterns

### Anti-Pattern 1: Tracing Without Context Propagation Through Kafka

**What people do:** Add `Telemetry` block to all three services, declare tracing "done," and ship it — not realizing Kafka is a gap.

**Why it's wrong:** The HTTP → Kafka → consumer path is the entire analytics pipeline. Without trace propagation through Kafka, the url-api trace ends at the Kafka push and the consumer trace is an orphan. You cannot trace a click event end-to-end.

**Do this instead:** Implement the TraceContext field in ClickEvent and span link pattern in the consumer. This connects the two traces in Jaeger.

### Anti-Pattern 2: All-in-One Jaeger in Production

**What people do:** Use `jaegertracing/all-in-one` in production.

**Why it's wrong:** All-in-one stores traces in memory. All traces are lost on container restart. Not suitable for any persistent observability.

**Do this instead:** For this project (local Docker Compose learning environment), all-in-one is correct. If the project were to be deployed, use Jaeger with a persistent storage backend. Flag this clearly in the Docker Compose file.

### Anti-Pattern 3: Changing zRPC Target Without Starting Etcd First

**What people do:** Change `url-api` to use Etcd resolver before Etcd is running.

**Why it's wrong:** `zrpc.MustNewClient` with `NonBlock: false` (the default) blocks until the first endpoint is resolved. With `NonBlock: true` (already set in url.yaml), it does not block — but connection will fail until Etcd has the analytics-rpc registration. Etcd must start and analytics-rpc must register before url-api starts making RPC calls.

**Do this instead:** Add `depends_on: etcd: condition: service_healthy` to url-api and analytics-rpc in Docker Compose before switching to Etcd discovery.

### Anti-Pattern 4: Custom Prometheus Instrumentation Before Checking DevServer

**What people do:** Add `prometheus/client_golang` metrics manually in handler middleware.

**Why it's wrong:** go-zero's DevServer already exposes `http_server_requests_duration_ms` and `http_server_requests_code_total` with per-path and per-method labels. Duplicating this creates redundant metrics with different names.

**Do this instead:** Use go-zero's built-in metrics from DevServer. Only add custom application metrics (e.g., `short_codes_created_total`, `redirects_total`) using `go-zero/core/metric` package if needed beyond what go-zero provides.

---

## Sources

- [OpenTelemetry Go-Zero monitoring | Uptrace](https://uptrace.dev/guides/opentelemetry-go-zero) — Telemetry YAML block format, automatic instrumentation behavior (HIGH confidence)
- [Service Configuration | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/grpc/server/configuration) — RpcServerConf Etcd fields (HIGH confidence)
- [Service Connection | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/grpc/client/conn) — RpcClientConf Etcd configuration (HIGH confidence)
- [Discovery | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/component/discovery/) — Etcd registration/discovery lifecycle (HIGH confidence)
- [Monitoring | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/monitor/index) — DevServer metrics, built-in Prometheus metrics names (HIGH confidence)
- [Kafka with OpenTelemetry | Last9](https://last9.io/blog/kafka-with-opentelemetry/) — Kafka trace propagation patterns, span link rationale (MEDIUM confidence)
- [Jaeger Deployment](https://www.jaegertracing.io/docs/1.76/deployment/) — OTLP ports 4317/4318, `COLLECTOR_OTLP_ENABLED` (HIGH confidence)
- [Prometheus with Docker Compose | Last9](https://last9.io/blog/prometheus-with-docker-compose/) — Scrape config patterns (HIGH confidence)
- [Grafana provisioning | Docker Docs](https://docs.docker.com/guides/go-prometheus-monitoring/compose/) — Dashboard and datasource provisioning patterns (HIGH confidence)
- Codebase analysis: `go.mod`, `docker-compose.yml`, all three service configs and entry points (authoritative)

---

*Architecture research for: v3.0 Observability & Service Discovery integration into go-zero URL shortener*
*Researched: 2026-02-22*
*Confidence: HIGH for go-zero native features (Telemetry, Etcd, DevServer metrics). MEDIUM for Kafka trace propagation — kq.Pusher header limitation confirmed via source inspection, JSON field workaround is standard practice.*
