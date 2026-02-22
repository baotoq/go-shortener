# Pitfalls Research

**Domain:** Adding observability (OpenTelemetry, Prometheus, Grafana) and Etcd service discovery to existing go-zero microservices
**Researched:** 2026-02-22
**Confidence:** HIGH (critical sections verified against go-zero v1.10.0 source and go-queue v1.2.2 source)

---

## Critical Pitfalls

### Pitfall 1: Using `KPush()` Instead of `Push()` Loses Trace Context in Kafka

**What goes wrong:**
Trace context is silently dropped at the Kafka producer boundary. The consumer starts a fresh trace with no connection to the HTTP request that triggered the publish. Jaeger shows two disconnected traces: one for the HTTP redirect, one orphaned consumer span.

**Why it happens:**
go-queue v1.2.2 has two publish methods with very different behavior. `PushWithKey()` (called by `Push()`) injects OTel context into Kafka message headers via `otel.GetTextMapPropagator().Inject(ctx, mc)`. But `KPush()` creates a plain `kafka.Message` with no header injection. The current url-api redirect handler likely uses `Push()`, which is correct — but this is a trap for anyone adding Kafka publishing elsewhere.

More critically, the ChunkExecutor (used when `syncPush` is NOT set, which is the default) uses `context.Background()` when flushing batches, not the original request context. This means the OTel context injected into the message header is preserved in the headers correctly (it's injected before Add()), but the executor flushes asynchronously. Verify that the headers are intact in the flushed message by reading the source at `pusher.go:139`.

**How to avoid:**
- Always use `Push()` or `PushWithKey()`, never `KPush()` — `KPush()` exists for explicit key control but skips OTel injection
- Verify trace headers arrive in Kafka by temporarily using `Batcher: file` in Telemetry config and logging message headers in consumer
- Add a debug log in `ClickEventConsumer.Consume()` to print the trace context extracted from `ctx` (check `trace.SpanFromContext(ctx).SpanContext().TraceID()`)
- The consumer side (go-queue's `startConsumers()`) correctly extracts OTel context from headers via `otel.GetTextMapPropagator().Extract()` — the consumer side is already correct

**Warning signs:**
- Jaeger shows click-events consumer spans as root spans (no parent), not children of the HTTP redirect span
- No `traceparent` header visible in Kafka message headers when debugging
- Traces for redirect and click processing appear completely disconnected in Jaeger

**Phase to address:**
Distributed Tracing phase — verify trace linkage end-to-end as part of tracing acceptance criteria. Test by sending a redirect request and checking that the resulting consumer span has the HTTP span as an ancestor in Jaeger.

---

### Pitfall 2: `Batcher: jaeger` Is Not a Valid go-zero Telemetry Option

**What goes wrong:**
Service starts silently. No traces appear in Jaeger. No error in logs because go-zero's `StartAgent()` swallows the "unknown exporter" error into `logx.Error()` rather than panicking. Developer spends hours debugging Jaeger configuration when the problem is a single YAML field.

**Why it happens:**
go-zero v1.10.0's `trace.Config` only supports four `Batcher` values: `zipkin`, `otlpgrpc`, `otlphttp`, `file`. The config tag enforces this: `json:",default=otlpgrpc,options=zipkin|otlpgrpc|otlphttp|file"`. Many blog posts and tutorials reference `Batcher: jaeger` (from an older native Jaeger exporter that was deprecated and removed from the OpenTelemetry Go SDK). Modern Jaeger (v1.35+) accepts OTLP natively on port 4317 (gRPC) and 4318 (HTTP).

**How to avoid:**
- Use `Batcher: otlpgrpc` pointing to Jaeger's OTLP gRPC port (default: 4317), not the Jaeger-specific port (14268)
- Use `Batcher: otlphttp` for HTTP-based export to Jaeger's port 4318
- Docker Compose Jaeger: use `jaegertracing/all-in-one` with OTLP enabled (it is by default in v1.35+)
- Set `Endpoint: jaeger:4317` for gRPC or `jaeger:4318` for HTTP in Docker Compose

Correct config:
```yaml
Telemetry:
  Name: url-api
  Endpoint: jaeger:4317
  Sampler: 1.0
  Batcher: otlpgrpc
```

**Warning signs:**
- Telemetry config has `Batcher: jaeger`
- No traces in Jaeger despite service running
- go-zero logs show `[otel] error: unknown exporter: jaeger` (check carefully — it's at `logx.Error` level, not fatal)
- Jaeger UI shows no services listed

**Phase to address:**
Distributed Tracing phase — set correct Batcher and verify traces appear before marking tracing complete.

---

### Pitfall 3: analytics-consumer Telemetry Not Initialized Because `Telemetry` Block Is Missing from YAML

**What goes wrong:**
url-api and analytics-rpc traces appear in Jaeger. analytics-consumer traces are completely absent. The Kafka-to-database pipeline is invisible.

**Why it happens:**
analytics-consumer's config struct embeds `service.ServiceConf` (not `rest.RestConf`), which does call `trace.StartAgent(sc.Telemetry)` during `SetUp()`. However, if the `Telemetry` block is absent from `consumer.yaml` and `consumer-docker.yaml`, `trace.StartAgent` is called with an empty `Config{}` where `Endpoint` is empty. Looking at `agent.go:115`: `if len(c.Endpoint) > 0` — when endpoint is empty, no exporter is created, no traces are emitted. The tracer provider is still initialized (so no panics), but exports to nowhere.

Additionally, analytics-consumer does not use `rest.RestConf` or `zrpc.RpcServerConf`, so automatic HTTP/gRPC instrumentation does NOT apply. Consumer spans must be created manually in `ClickEventConsumer.Consume()` using `tracer.Start(ctx, "consume-click-event")`.

**How to avoid:**
- Add `Telemetry` block to BOTH `consumer.yaml` AND `consumer-docker.yaml`
- Create explicit spans in `ClickEventConsumer.Consume()` — the context passed in already has the extracted parent span from Kafka headers; create a child span immediately
- Pattern:
  ```go
  tracer := otel.Tracer("analytics-consumer")
  ctx, span := tracer.Start(ctx, "consume-click-event")
  defer span.End()
  ```
- Verify by sending a redirect and checking Jaeger for a three-hop trace: HTTP redirect → Kafka publish → consumer insert

**Warning signs:**
- url-api and analytics-rpc traces in Jaeger, no analytics-consumer traces
- Consumer logs show normal processing with no trace IDs in log output
- `Telemetry` block exists in url-docker.yaml but not consumer-docker.yaml

**Phase to address:**
Distributed Tracing phase — add manual spans to consumer and add Telemetry config to consumer YAML files as explicit acceptance criteria.

---

### Pitfall 4: Etcd Migration Breaks analytics-rpc at Startup if Etcd Unavailable

**What goes wrong:**
After migrating from `Target: dns:///analytics-rpc:8081` to Etcd-based discovery, url-api fails to start if Etcd is not running. The current setup has `NonBlock: true` which prevents blocking on connection, but with Etcd, if the service key doesn't exist yet (analytics-rpc hasn't registered), url-api may get an empty endpoint list and all RPC calls fail immediately instead of degrading gracefully.

**Why it happens:**
go-zero's Etcd resolver loads endpoints at client construction time via the `load` method. If analytics-rpc hasn't registered yet (startup race), the endpoint list is empty. `NonBlock: true` allows the client to construct without waiting, but the first RPC call to an empty endpoint list returns `no available connection` rather than the graceful degradation path. This differs from the `dns:///` behavior where the hostname resolves to an IP even if gRPC isn't accepting connections yet.

**How to avoid:**
- Keep `NonBlock: true` in the RPC client config (already set — preserves graceful degradation)
- Add `depends_on` for Etcd in Docker Compose for both url-api and analytics-rpc containers
- analytics-rpc must register in Etcd before url-api starts making calls — Docker Compose `service_started` condition for analytics-rpc is sufficient
- The existing graceful degradation in url-api's link detail logic (returning 0 clicks on RPC error) already handles this — verify it catches `no available connection` errors, not just timeout errors
- Test degradation by starting url-api before analytics-rpc with Etcd-based discovery

**Warning signs:**
- url-api logs `no available connection` during startup before analytics-rpc is ready
- Link detail endpoint returns errors instead of 0 clicks during startup
- zRPC calls that succeeded with `dns:///` now fail with Etcd-based discovery
- `depends_on` not updated to include Etcd healthcheck

**Phase to address:**
Etcd Service Discovery phase — test startup ordering explicitly, update Docker Compose depends_on, verify graceful degradation still works.

---

### Pitfall 5: Prometheus Cardinality Explosion from High-Cardinality Labels

**What goes wrong:**
Custom Prometheus metrics are added with `short_code` or `url` as a label. Each unique short code (potentially millions) creates a separate time series. Prometheus memory usage grows unboundedly. Scrape timeouts begin. Grafana dashboards slow to a crawl. Prometheus OOM kills.

**Why it happens:**
Prometheus stores each unique label combination as a separate time series. URL shorteners naturally have high-cardinality identifiers (short_code, original_url, user_ip). Developers instrumenting click metrics think "I want to see clicks per short code" and add `short_code` as a label — this is correct for counters in application code but catastrophic for Prometheus labels.

go-zero's built-in metrics (HTTP request duration, gRPC call duration, database query duration) use low-cardinality labels (`method`, `path`, `statusCode`). Custom metrics are where cardinality explosions happen.

**How to avoid:**
- NEVER use `short_code`, `url`, `ip`, or any user-provided data as a Prometheus label
- High-cardinality analytics (clicks per short_code) belong in PostgreSQL, not Prometheus
- Prometheus is for system health (request rate, latency percentiles, error rate) — not business analytics
- Safe labels for this project: `service_name`, `method`, `status` (2xx/4xx/5xx), `handler` (fixed set of routes), `topic` (fixed set of Kafka topics)
- Use Prometheus Recording Rules to pre-aggregate if needed, rather than high-cardinality raw metrics
- Check Prometheus cardinality with: `sort_desc(count by (__name__)({__name__=~".+"}))` to find runaway series

**Warning signs:**
- Prometheus scrape duration increasing over time (>1s per scrape)
- Memory usage growing proportionally to number of unique short codes
- `TSDB head series` metric increasing without bound
- `prometheus_tsdb_head_series` exceeds 100k
- Grafana dashboard queries taking >5s

**Phase to address:**
Prometheus Metrics phase — define metric taxonomy upfront, code review all custom metric label choices before implementing dashboards.

---

### Pitfall 6: go-zero Built-in Metrics Disabled Because `prometheus.Enable()` Not Called

**What goes wrong:**
DevServer `/metrics` endpoint returns an empty or minimal metric set. go-zero's built-in HTTP request duration, gRPC call metrics, and database query metrics are absent. Only default Go runtime metrics appear. Grafana dashboards based on go-zero metrics show no data.

**Why it happens:**
go-zero uses a global Prometheus enable switch (`prometheus.Enable()`). This is called by `devserver.StartAgent()` only when `DevServer.EnableMetrics: true` (the default). However, if `DevServer.Enabled: false` is set anywhere (or the DevServer port conflicts), the Prometheus switch is never flipped. go-zero's internal metrics are gated behind this switch — they are registered but not reported until enabled.

The existing configs all have `DevServer.Enabled: true` with ports 6470, 6471, 6472 — this is correct. The pitfall is if someone sets `Enabled: false` to "simplify" the Docker Compose setup or if port conflicts cause the DevServer to fail silently.

**How to avoid:**
- Verify DevServer is running by hitting `/metrics` on each service's DevServer port before building dashboards
- Keep `EnableMetrics: true` (it's the default — don't override to false)
- Each service MUST have a unique DevServer port (already: 6470, 6471, 6472 — preserve these)
- Add a smoke test: `curl -s http://localhost:6470/metrics | grep go_zero` to verify go-zero metrics present
- For analytics-consumer: it uses `service.ServiceConf` and `devserver.StartAgent()` IS called — verify port 6472 returns metrics

**Warning signs:**
- `/metrics` endpoint returns only Go runtime metrics (`go_goroutines`, `process_cpu_seconds_total`) but no `rpc_client_*` or `http_server_*` metrics
- `DevServer.Enabled: false` in any service config
- Port conflicts on DevServer ports (6470/6471/6472) in Docker Compose

**Phase to address:**
Prometheus Metrics phase — verify metrics endpoint content as acceptance criteria before starting Grafana dashboard work.

---

### Pitfall 7: Grafana Dashboard Using `avg()` for Latency Hides Tail Latency

**What goes wrong:**
Grafana dashboard looks healthy — average response time is 45ms. P99 is actually 2.3 seconds for redirect requests. Users with slow connections or cache misses experience multi-second delays that never appear on the dashboard. SLO violations go undetected.

**Why it happens:**
`avg()` in PromQL hides bimodal distributions. URL shorteners have two response populations: cache hits (very fast) and cache misses / DB queries (slower). Average blends them. Prometheus histograms enable accurate quantile calculation via `histogram_quantile()`, but developers default to `avg()` because it's simpler and the result "looks clean."

Additionally, go-zero's built-in HTTP metrics use histograms with default buckets. The default Prometheus histogram buckets (`[.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10]` seconds) may not align with the redirect endpoint's actual latency distribution, making `histogram_quantile` interpolation inaccurate.

**How to avoid:**
- Use `histogram_quantile(0.99, rate(http_server_request_duration_ms_bucket[5m]))` for P99, not `avg()`
- Use `histogram_quantile(0.50, ...)` for median (P50), not `avg()`
- Add SLO thresholds as horizontal lines on latency panels (e.g., 200ms for redirect P99)
- If using custom histograms, define buckets that bracket your SLO: if SLO is "redirects under 100ms", include buckets at 50ms, 100ms, 200ms
- Use the RED method: Rate (requests/sec), Errors (error rate %), Duration (P50/P95/P99 latency) — not Average

**Warning signs:**
- Latency panels use `avg()` or `sum()` without percentile functions
- No SLO threshold lines on dashboard panels
- Dashboard shows "all green" but user complaints about slowness
- Histogram panel shows `_bucket` or `_count` metrics directly instead of `histogram_quantile()`

**Phase to address:**
Grafana Dashboards phase — establish RED method dashboard template before creating individual panels.

---

### Pitfall 8: Docker Compose Startup Order Fails with Etcd and Jaeger Dependencies

**What goes wrong:**
`docker-compose up` brings up services in the wrong order. analytics-rpc starts before Etcd is ready, fails to register, never appears in Etcd. url-api's zRPC client gets an empty endpoint list permanently. Or Jaeger starts after services, so the first traces are lost during the startup gap. Services exit and restart several times before stabilizing, creating confusing log noise.

**Why it happens:**
The current `docker-compose.yml` has healthchecks for PostgreSQL and Kafka but NOT for Etcd or Jaeger. Adding Etcd and Jaeger as new infrastructure services without healthchecks means `depends_on: condition: service_started` only checks if the container process started, not if Etcd's gRPC port is accepting connections or if Jaeger's OTLP port is ready.

Etcd startup is particularly slow on first run (generating TLS certificates, initializing the cluster). analytics-rpc's Etcd registration attempt can fail if analytics-rpc starts before Etcd's port 2379 is ready.

**How to avoid:**
- Add healthchecks for Etcd in Docker Compose:
  ```yaml
  healthcheck:
    test: ["CMD-SHELL", "etcdctl endpoint health --endpoints=localhost:2379"]
    interval: 5s
    timeout: 5s
    retries: 10
    start_period: 10s
  ```
- Add analytics-rpc `depends_on` for Etcd with `condition: service_healthy`
- Jaeger: traces lost during startup are acceptable (Sampler: 1.0 means all traces are attempted, but early failures are non-critical). Use `service_started` not `service_healthy` for Jaeger dependency
- Add `restart: unless-stopped` to all services (already present — keep this)
- Use `ALLOW_NONE_AUTHENTICATION: "yes"` and `ETCD_ADVERTISE_CLIENT_URLS` environment variables for single-node Etcd in development

**Warning signs:**
- analytics-rpc logs show Etcd connection refused during startup
- `docker-compose up` requires multiple `docker-compose restart analytics-rpc` to stabilize
- Etcd container has no healthcheck defined
- No `depends_on` relationship between analytics-rpc and Etcd

**Phase to address:**
Infrastructure Setup phase (first phase of v3.0) — add all new infrastructure services with proper healthchecks before writing application code.

---

### Pitfall 9: Etcd Volume Permission Denied with Non-Root Container Images

**What goes wrong:**
Etcd container fails to start with `etcdmain: cannot access data directory: permission denied`. This blocks all service discovery. The error only occurs when Etcd has a named volume mount, because Docker named volumes can be created with root ownership that conflicts with Etcd's non-root user.

**Why it happens:**
Official `etcd` or `bitnami/etcd` images run as non-root users. When a named Docker volume is first created, it's owned by root. The Etcd process (running as non-root) cannot write to the data directory. This is a first-run problem — subsequent runs work fine because the data directory already exists. The issue commonly appears in CI or after `docker-compose down -v`.

**How to avoid:**
- For development, use `tmpfs` instead of named volumes for Etcd data (data loss on restart is acceptable):
  ```yaml
  etcd:
    tmpfs:
      - /etcd-data
  ```
- OR use the `bitnami/etcd` image which handles permissions automatically
- OR add `user: root` to the Etcd service in Docker Compose (acceptable for local dev)
- NEVER use named volumes for Etcd in development — ephemeral Etcd is correct for local dev
- Ensure `ETCD_DATA_DIR` points to a directory the process can write

**Warning signs:**
- `docker-compose up` works first time but fails after `docker-compose down -v`
- Etcd container exits immediately with error code 1
- `docker logs etcd` shows `cannot access data directory`
- CI pipeline fails on Etcd startup inconsistently

**Phase to address:**
Infrastructure Setup phase — use tmpfs for Etcd data in development from the start.

---

### Pitfall 10: Prometheus Scrape Config Missing DevServer Ports

**What goes wrong:**
Prometheus starts, Grafana connects to it, but no go-zero metrics appear. Prometheus `Status > Targets` shows all targets as `DOWN`. The services are running and `/metrics` endpoints are healthy, but Prometheus can't reach them because the scrape config uses wrong ports (e.g., scraping port 8080 instead of DevServer port 6470 for url-api).

**Why it happens:**
go-zero exposes metrics on the DevServer port (6470/6471/6472), NOT on the main service port (8080/8081). Developers new to go-zero assume metrics are on the main port like many other frameworks. Additionally, when running in Docker Compose, `localhost` in Prometheus scrape config refers to the Prometheus container, not the host machine — service names must be used instead.

**How to avoid:**
- Prometheus scrape targets must use Docker Compose service names and DevServer ports:
  ```yaml
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
- Verify scrape config before adding Prometheus to Docker Compose: test with `curl http://service:6470/metrics` from within the Docker network
- Check `Status > Targets` in Prometheus UI after startup — all targets should show `UP`

**Warning signs:**
- Prometheus `Status > Targets` shows services as `DOWN`
- Using port 8080/8081 in Prometheus scrape config instead of 6470/6471/6472
- Using `localhost` instead of Docker service names in Prometheus `prometheus.yml`
- `scrape_configs` uses IP addresses instead of service names

**Phase to address:**
Prometheus Metrics phase — create `prometheus.yml` with correct service names and DevServer ports as the first step.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| `Sampler: 1.0` in production | See all traces | Storage/cost explosion at scale | Local dev and staging only — use 0.1 or lower in production |
| Skip manual spans in consumer | Less code | Kafka-to-DB path invisible in traces | Never — consumer spans are essential for end-to-end tracing |
| Use `avg()` for latency dashboards | Simpler queries | Hides tail latency, SLO violations undetected | Never — always use `histogram_quantile` |
| Single Etcd node | Simple setup | No HA, single point of failure | Acceptable for local dev, not production |
| Hardcode Prometheus target IPs | Quick setup | Breaks when containers restart/reschedule | Never — use service names |
| Add `short_code` as Prometheus label | Per-code visibility | Cardinality explosion, Prometheus OOM | Never — use PostgreSQL for per-code analytics |
| Skip Jaeger healthcheck in compose | Faster startup | First traces lost, race conditions | Acceptable (traces lost at startup are non-critical) |
| Skip Etcd healthcheck in compose | Faster startup | Services register before Etcd ready, silent failures | Never — Etcd must be healthy before services register |
| Keep `NonBlock: true` with Etcd | No startup blocking | First few RPC calls may fail | Acceptable — existing graceful degradation handles this |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| go-zero + Jaeger | `Batcher: jaeger` in Telemetry config | `Batcher: otlpgrpc` with `Endpoint: jaeger:4317` |
| go-queue + OTel | Using `KPush()` to publish events | Use `Push()` or `PushWithKey()` — they inject OTel headers |
| analytics-consumer + tracing | Relying on auto-instrumentation | Create manual spans in `Consume()` — no auto-instrumentation for Kafka consumers |
| Prometheus + Docker Compose | Scraping `localhost:6470` | Scrape `url-api:6470` using Docker service names |
| Prometheus + go-zero | Scraping main port 8080 | Scrape DevServer port 6470/6471/6472 |
| Etcd + Docker Compose | Named volume for Etcd data | Use `tmpfs` for local dev to avoid permission issues |
| Grafana + Prometheus | Connecting to `localhost:9090` | Use Docker service name `prometheus:9090` |
| Etcd + zRPC client | Keeping `NonBlock: false` | Use `NonBlock: true` to preserve graceful degradation |
| analytics-rpc + Etcd | Registering before Etcd ready | Add `depends_on: etcd: condition: service_healthy` |
| OTel + go-zero | Initializing OTel manually | Let `ServiceConf.SetUp()` call `trace.StartAgent()` via `Telemetry` config |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| Sampler: 1.0 in production | High trace storage costs, slow OTLP export | Use 0.1 (10%) in production, 1.0 in dev | >100 req/sec generates unsustainable trace volume |
| Custom high-cardinality Prometheus labels | Prometheus memory growth, slow scrapes | Limit labels to known low-cardinality values | >10k unique label values causes noticeable degradation |
| Synchronous OTel export blocking requests | Request latency increase | go-zero uses batch exporter by default — don't switch to synchronous | Any volume — synchronous export adds network RTT to request |
| Grafana querying raw `_bucket` metrics | Slow dashboard load | Use `histogram_quantile()` with `rate()` for summaries | >10 time series with recording rules needed |
| Etcd watch goroutines per connection | Memory growth with many clients | go-zero uses a single shared resolver — not a problem at this scale | N/A for 3-service system |
| Jaeger all-in-one storing all traces in memory | Jaeger OOM after extended testing | Configure `SPAN_STORAGE_TYPE=badger` with volume | After ~100k spans in memory |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Etcd with no authentication | Any process can register fake services | Use `ETCD_AUTH_TOKEN` or mTLS for non-dev environments |
| Prometheus metrics endpoint exposed publicly | Operational data leak | Bind DevServer to internal network only, not 0.0.0.0 in production |
| Jaeger UI exposed publicly | Trace data (URLs, parameters) visible to anyone | Add authentication proxy in front of Jaeger in production |
| Etcd credentials in docker-compose.yml | Credential leak in git | Use environment variable substitution or secrets |
| OTel headers containing sensitive data | Short codes, user IPs in trace attributes | Avoid adding request body or PII as span attributes |

## "Looks Done But Isn't" Checklist

- [ ] **Trace propagation**: Traces appear in Jaeger for url-api — verify the redirect → Kafka → consumer chain is a single connected trace (not three separate root spans)
- [ ] **Consumer spans**: analytics-consumer appears as a traced service in Jaeger — verify it creates child spans from the Kafka message context
- [ ] **Batcher config**: `Batcher: otlpgrpc` in all service YAML files, not `Batcher: jaeger` — verify by checking Jaeger for traces
- [ ] **DevServer metrics**: All three services export go-zero built-in metrics — verify `curl http://localhost:6470/metrics | grep http_server` returns data
- [ ] **Prometheus scrape**: All three services show `UP` in Prometheus `Status > Targets` — not just "running" in Docker Compose
- [ ] **Grafana data source**: Prometheus URL in Grafana uses service name `prometheus:9090`, not `localhost:9090`
- [ ] **Cardinality check**: No custom Prometheus metrics use `short_code`, `url`, or IP as a label value
- [ ] **Etcd registration**: analytics-rpc appears in Etcd after startup — verify with `etcdctl get --prefix /analytics-rpc`
- [ ] **Graceful degradation preserved**: With Etcd-based discovery, link detail still returns 0 clicks (not an error) when analytics-rpc is down
- [ ] **Etcd volume**: No named volume for Etcd data in Docker Compose — use tmpfs for local dev
- [ ] **Histogram buckets**: Dashboard latency panels use `histogram_quantile(0.99, rate(...))`, not `avg()` — verify in Grafana panel query
- [ ] **Telemetry in consumer**: `Telemetry` block present in both `consumer.yaml` and `consumer-docker.yaml`
- [ ] **Jaeger port**: Telemetry `Endpoint` uses OTLP port (4317 for gRPC, 4318 for HTTP), not Jaeger-native port (14268 or 14250)

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Broken trace propagation | MEDIUM | Enable `Batcher: file` to debug, verify OTel headers in Kafka messages, check `Push()` vs `KPush()` usage |
| Wrong Batcher value | LOW | Change `Batcher: jaeger` to `Batcher: otlpgrpc`, restart service |
| No consumer traces | LOW | Add `Telemetry` block to consumer YAML, add manual `tracer.Start()` in `Consume()` |
| Etcd permission denied | LOW | Remove named volume, add tmpfs, `docker-compose down -v && docker-compose up` |
| Prometheus scraping wrong port | LOW | Update `prometheus.yml` scrape targets, reload Prometheus config (`curl -X POST /-/reload`) |
| Cardinality explosion | HIGH | Drop offending metric series with `prometheus_tsdb_delete_series`, fix label, restart Prometheus; history is lost |
| Grafana shows no data | LOW | Check Prometheus target health, verify data source URL, inspect panel query |
| Etcd startup race | LOW | Add healthcheck to Etcd in Docker Compose, add `depends_on` for analytics-rpc, redeploy |
| Lost graceful degradation with Etcd | MEDIUM | Verify error handling in link detail logic catches `no available connection`; add explicit error type check |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Wrong Batcher value (jaeger vs otlpgrpc) | Infrastructure Setup | Jaeger UI shows traces from all three services |
| Kafka trace context broken (KPush vs Push) | Distributed Tracing | Single connected trace in Jaeger: redirect → Kafka → consumer |
| Consumer no auto-instrumentation | Distributed Tracing | analytics-consumer service visible in Jaeger service list |
| Consumer YAML missing Telemetry block | Distributed Tracing | consumer-docker.yaml and consumer.yaml both have Telemetry block |
| Prometheus scraping wrong ports | Prometheus Metrics | All targets show UP in Prometheus Status > Targets |
| Cardinality explosion from bad labels | Prometheus Metrics | Code review all custom metric label values; `prometheus_tsdb_head_series` < 10k |
| Metrics disabled (DevServer misconfigured) | Prometheus Metrics | `curl :6470/metrics` returns go-zero built-in HTTP/RPC metrics |
| Grafana avg() latency dashboards | Grafana Dashboards | All latency panels use histogram_quantile, not avg() |
| Etcd startup race in Docker Compose | Etcd Service Discovery | `docker-compose up` cold start succeeds without manual restarts |
| Etcd volume permission denied | Etcd Service Discovery | `docker-compose down -v && docker-compose up` succeeds |
| Etcd migration breaks graceful degradation | Etcd Service Discovery | Link detail returns 0 clicks when analytics-rpc is stopped |
| Prometheus target IPs instead of service names | Infrastructure Setup | prometheus.yml uses Docker service names, not IPs |

## Sources

**go-zero source code (verified against v1.10.0):**
- `core/trace/config.go` — Batcher options: `zipkin|otlpgrpc|otlphttp|file` only, no `jaeger`
- `core/trace/agent.go` — `StartAgent()` uses `sync.Once`, empty endpoint skips exporter creation
- `core/service/serviceconf.go` — `SetUp()` calls `trace.StartAgent()` and `devserver.StartAgent()`
- `internal/devserver/server.go` — `prometheus.Enable()` gated on `EnableMetrics: true`

**go-queue source code (verified against v1.2.2):**
- `kq/pusher.go` — `PushWithKey()` injects OTel headers; `KPush()` does NOT inject headers
- `kq/queue.go:226` — `startConsumers()` extracts OTel context from Kafka headers correctly
- `kq/internal/trace.go` — `MessageCarrier` implements `propagation.TextMapCarrier` for header injection/extraction

**OpenTelemetry Go and Kafka:**
- [Kafka with OpenTelemetry: Distributed Tracing Guide | Last9](https://last9.io/blog/kafka-with-opentelemetry/)
- [Complete Guide to tracing Kafka clients with OpenTelemetry in Go | SigNoz](https://signoz.io/blog/opentelemetry-kafka/)
- [How to Propagate Trace Context Across Kafka Producers and Consumers](https://oneuptime.com/blog/post/2026-02-06-propagate-trace-context-kafka-producers-consumers/view)

**go-zero Telemetry documentation:**
- [OpenTelemetry Go-Zero monitoring | Uptrace](https://uptrace.dev/guides/opentelemetry-go-zero)
- [Prometheus Configuration | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/go-zero/configuration/prometheus)

**Prometheus cardinality:**
- [How to manage high cardinality metrics in Prometheus | Grafana Labs](https://grafana.com/blog/how-to-manage-high-cardinality-metrics-in-prometheus-and-kubernetes/)
- [Cardinality is key | Robust Perception](https://www.robustperception.io/cardinality-is-key/)
- [Prometheus Label cardinality explosion - Stack Diagnosis | DrDroid](https://drdroid.io/stack-diagnosis/prometheus-label-cardinality-explosion)

**Grafana dashboard best practices:**
- [Grafana dashboard best practices](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/best-practices/)
- [The RED Method: How to Instrument Your Services | Grafana Labs](https://grafana.com/blog/the-red-method-how-to-instrument-your-services/)

**go-zero Etcd service discovery:**
- [Discovery | go-zero](https://zeromicro.github.io/go-zero.dev/en/docs/component/discovery/)
- [Service Connection | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/grpc/client/conn)

**Docker Compose and healthchecks:**
- [Control startup order - Docker Compose](https://docs.docker.com/compose/how-tos/startup-order/)
- [Docker Compose Health Checks | Last9](https://last9.io/blog/docker-compose-health-checks/)
- [Etcd data_dir permission denied | coreos/bugs GitHub](https://github.com/coreos/bugs/issues/2446)

---
*Pitfalls research for: Adding observability and service discovery to go-zero microservices (v3.0 Production Readiness)*
*Researched: 2026-02-22*
