# Project Research Summary

**Project:** go-shortener v3.0 — Production Readiness Milestone
**Domain:** Observability, Service Discovery, and Tech Debt for go-zero Microservices
**Researched:** 2026-02-22
**Confidence:** HIGH

## Executive Summary

This milestone adds production-readiness infrastructure to an existing, validated go-zero microservices system (url-api, analytics-rpc, analytics-consumer) running on PostgreSQL, Kafka, and Docker Compose. The critical finding across all four research areas is that go-zero already bundles OpenTelemetry, Prometheus client, and etcd client as transitive dependencies — no new Go packages are required. The entire observability stack is activated through YAML configuration changes, with the single exception of Kafka trace propagation which requires two code changes (inject in redirect logic, extract in consumer).

The recommended approach is a four-phase execution ordered by infrastructure dependencies: (1) tech debt cleanup as a standalone warmup, (2) etcd service discovery which requires only YAML and Docker Compose changes, (3) distributed tracing including the Kafka boundary which is the most complex code change, and (4) Prometheus + Grafana which depends on metrics being exported before dashboards can be built. Jaeger v2 (`jaegertracing/jaeger:2.15.0`) accepts OTLP natively — the `Batcher: otlpgrpc` config targets it directly, bypassing the deprecated Jaeger exporter. Prometheus 3.x (`prom/prometheus:v3.9.1`) and Grafana 12.3.3 are the current stable releases.

The primary risks are silent failures that look like working configurations: using `Batcher: jaeger` emits no error but produces no traces, scraping the wrong Prometheus ports results in empty dashboards with no errors, and missing the `Telemetry` block from analytics-consumer YAML leaves the most important observability gap (Kafka-to-database pipeline) invisible. Each pitfall has a clear verification step that must be part of phase acceptance criteria. Cardinality explosion from using `short_code` as a Prometheus label is the only risk with HIGH recovery cost — it must be prevented by design, not fixed after the fact.

## Key Findings

### Recommended Stack

The v3.0 stack adds four Docker Compose infrastructure services only. No new Go packages are needed. All Go libraries activate via YAML config — the OTel SDK, Prometheus client, and etcd client are already in the dependency graph as transitive dependencies of go-zero v1.10.0.

**Core technologies:**

- **Jaeger v2 (`jaegertracing/jaeger:2.15.0`):** Distributed trace backend — Jaeger v1 reached EOL December 31, 2025; v2 is built on OTel Collector core and natively accepts OTLP on port 4317 (gRPC) and 4318 (HTTP). Single container for local dev.
- **go-zero `Telemetry` YAML block (built-in, no new package):** Activates the OTel SDK already bundled as a transitive dep; auto-instruments HTTP handlers and gRPC calls with zero code changes. Use `Batcher: otlpgrpc`, never `Batcher: jaeger` (removed from OTel Go in 2023).
- **Prometheus 3.x (`prom/prometheus:v3.9.1`):** Metrics scraping and storage — v3.x is current stable (released 2026-01-07); v2.x is maintenance-only. Scrapes go-zero DevServer endpoints already running at ports 6470/6471/6472.
- **Grafana 12.3.3 (`grafana/grafana:12.3.3`):** Dashboard visualization — auto-provisioned from YAML/JSON files mounted at container start; no manual UI setup required. Dashboard 19909 on Grafana Labs covers go-zero's built-in HTTP/RPC metrics out of the box.
- **etcd (`gcr.io/etcd-development/etcd:v3.5.27`):** Service registry for zRPC — go-zero's `RpcServerConf` and `RpcClientConf` natively embed `EtcdConf`; switching from `Target: dns:///localhost:8081` to etcd-based discovery requires only YAML changes and no Go code changes. Use official `gcr.io/etcd-development/etcd` image, not `bitnami/etcd` (deprecated).
- **OTel propagation (built-in, already in go.mod):** W3C traceparent injection/extraction for Kafka boundary — packages are present as transitive deps; only explicit imports needed in redirect logic and consumer.

### Expected Features

**Must have (table stakes — v3.0 core):**

- Telemetry YAML block in all 3 service configs — enables HTTP and zRPC auto-tracing with one block per service; no code changes
- Jaeger in Docker Compose — without it, traces are generated but never viewable
- Kafka trace context propagation via `TraceContext` field in `ClickEvent` JSON — without it, the redirect → analytics pipeline is invisible in Jaeger as two disconnected traces
- Prometheus in Docker Compose with `prometheus.yml` scrape config targeting DevServer ports 6470/6471/6472 (not the main service ports 8080/8081)
- Grafana in Docker Compose with provisioned Prometheus datasource and pre-built go-zero dashboard (Grafana Labs ID 19909)
- etcd in Docker Compose + analytics-rpc registration + url-api discovery — replaces hardcoded `Target: dns:///localhost:8081`

**Should have (v3.0 enhancements, add after core is working):**

- Custom business metrics using `go-zero/core/metric` — CounterVec for redirects and URL creation events
- Kafka topic retention configuration — explicit `KAFKA_LOG_RETENTION_HOURS` env var to prevent unbounded disk growth (known tech debt)
- Dead code removal — delete `IncrementClickCount` from url model (~10 lines; replaced by Kafka consumer approach)
- Consumer test coverage — unit tests for `resolveCountry`, `resolveDeviceType`, `resolveTrafficSource` enrichment helpers

**Defer to future milestone (post v3.0):**

- Automated E2E test via testcontainers-go — high complexity, warrants its own research phase
- Prometheus alerting rules — operational value but not needed for a learning project milestone
- Consumer-specific Grafana panels — dashboard 19909 covers HTTP/RPC; Kafka consumer metrics need custom panels

**Anti-features to avoid:**

- OpenTelemetry Collector as middleware layer — adds a 5th container with complex config; Jaeger v2 accepts OTLP directly
- Consul for service discovery — requires third-party go-zero/contrib package; etcd is already in go.mod
- Sampling rate below 1.0 in development — hides traces, makes debugging impossible during dev
- Any Prometheus label using `short_code`, `url`, or IP — cardinality explosion with HIGH recovery cost (must DROP series, not just fix config)

### Architecture Approach

The v3.0 architecture is an additive overlay on the existing three-service system. The Handler → Logic → Model pattern, ServiceContext DI, and Docker Compose structure are unchanged. Four new Docker Compose services (etcd, Jaeger, Prometheus, Grafana) and a new `infra/` directory are the primary structural additions. The most architecturally significant decision is Kafka trace propagation: because `kq.Pusher` does not expose Kafka message headers for external manipulation, the W3C `traceparent` value is embedded as a `TraceContext string` field in the `ClickEvent` JSON payload. The consumer extracts this and creates a span link (not parent-child) to the producer span — span links correctly represent the async Kafka relationship.

**Major components and responsibilities:**

1. **Jaeger (OTLP receiver + UI)** — receives traces from all three services via OTLP gRPC (port 4317); visualizes trace chains including span links across the Kafka boundary; UI at port 16686; all-in-one image for local dev only
2. **Prometheus (metrics scraper + query engine)** — scrapes go-zero DevServer `/metrics` endpoints every 15s; stores time series for Grafana panels
3. **Grafana (dashboard + provisioning)** — auto-provisions Prometheus datasource and go-zero dashboard on container start; all panels must use `histogram_quantile()`, never `avg()`; config committed to `infra/grafana/`
4. **etcd (service registry)** — analytics-rpc registers on startup; url-api resolves analytics-rpc endpoint dynamically; replaces hardcoded DNS; use tmpfs (not named volume) to avoid permission issues
5. **`infra/` directory (new)** — `infra/prometheus/prometheus.yml` and `infra/grafana/provisioning/` with datasource + dashboard YAML; dashboard JSON committed to version control for reproducibility

**Files changed in existing services:**

| File | Change Type |
|------|-------------|
| `services/*/etc/*.yaml` | Config — add Telemetry block to all three; analytics-rpc and url-api add Etcd block |
| `common/events/events.go` | Code — add `TraceContext string` field to ClickEvent |
| `services/url-api/internal/logic/redirect/redirectlogic.go` | Code — inject traceparent into ClickEvent before Kafka push |
| `services/analytics-consumer/internal/mqs/clickeventconsumer.go` | Code — extract traceparent, create span link in Consume() |
| `docker-compose.yml` | New services — etcd, jaeger, prometheus, grafana; updated depends_on |
| `infra/` (new) | New files — prometheus.yml, Grafana provisioning YAML, dashboard JSON |

**No changes to:** Handler files, model files, generated code, ServiceContext struct.

### Critical Pitfalls

1. **`Batcher: jaeger` produces silent failure** — go-zero's `StartAgent()` swallows the "unknown exporter" error into `logx.Error()` (not fatal); no traces appear and no crash occurs. Use `Batcher: otlpgrpc` with `Endpoint: jaeger:4317`. Verify by checking Jaeger UI shows all three services listed.

2. **Kafka trace context breaks silently without code changes** — Adding `Telemetry` YAML alone does NOT propagate traces through Kafka. The correct approach is embedding `TraceContext` in `ClickEvent` struct and manually calling `otel.GetTextMapPropagator().Inject()/Extract()`. Verify by checking Jaeger shows a three-hop connected trace: HTTP redirect → Kafka publish → consumer insert.

3. **analytics-consumer YAML missing `Telemetry` block leaves pipeline invisible** — `service.ServiceConf.SetUp()` only initializes the OTel SDK when `Telemetry.Endpoint` is non-empty; if the block is absent from both `consumer.yaml` and `consumer-docker.yaml`, no consumer spans are emitted. Additionally, consumer spans must be created manually in `Consume()` — there is no auto-instrumentation for Kafka consumers.

4. **Etcd startup race breaks Docker Compose cold start** — Without `depends_on: etcd: condition: service_healthy`, analytics-rpc can attempt registration before etcd's port 2379 is ready. Add a proper etcd healthcheck (`etcdctl endpoint health`) and set etcd as a healthy-condition dependency for analytics-rpc from the start.

5. **Prometheus scraping wrong ports results in empty dashboards** — go-zero exposes metrics on DevServer ports (6470/6471/6472), not the main service port (8080/8081). Inside Docker Compose, `localhost` refers to the Prometheus container, not the host — use Docker service names (`url-api:6470`, not `localhost:6470`). Verify with Prometheus `Status > Targets` showing all three services as UP.

## Implications for Roadmap

Based on the dependency analysis from ARCHITECTURE.md and the pitfall-to-phase mapping from PITFALLS.md, the research supports a four-phase execution ordered by infrastructure prerequisites.

### Phase 1: Tech Debt Cleanup

**Rationale:** No dependencies on v3.0 infrastructure; provides a clean baseline before adding observability. Low-risk, isolated changes that build confidence and reduce debugging surface area.

**Delivers:** Dead code removed (`IncrementClickCount`), consumer enrichment helpers unit tested, Kafka topic retention configured.

**Addresses:** P2 features (dead code removal, consumer test coverage, Kafka retention config)

**Avoids:** No specific pitfalls, but a cleaner codebase reduces ambiguity when tracing is added in Phase 3.

**Research flag:** Standard patterns — skip `/gsd:research-phase`. Delete method, verify no callers (`go build ./...`), add test cases for three helpers, add Docker Compose env var.

### Phase 2: Infrastructure Setup and Etcd Service Discovery

**Rationale:** All new Docker Compose services must be in place before any service configuration changes. Etcd is pure YAML + Docker Compose with no Go code changes — lowest risk observability feature. Adding Jaeger and Prometheus containers here also enables Phase 3 verification without requiring a separate Docker Compose change.

**Delivers:** etcd running in Docker Compose with healthcheck and tmpfs volume; analytics-rpc registers on startup; url-api discovers analytics-rpc via etcd resolver; Jaeger, Prometheus, Grafana containers running (empty but ready); `infra/` directory structure created with placeholder configs.

**Addresses:** Etcd service discovery (P1), infrastructure foundation for all subsequent phases

**Avoids:**
- Etcd startup race — add healthcheck from the start (PITFALL 8)
- Etcd volume permission denied — use tmpfs, never named volume (PITFALL 9)
- Breaking graceful degradation — verify link detail still returns 0 clicks when analytics-rpc is stopped

**Research flag:** Standard patterns — skip `/gsd:research-phase`. go-zero etcd YAML config fields confirmed in STACK.md. Docker Compose healthcheck patterns are standard.

### Phase 3: Distributed Tracing

**Rationale:** Jaeger must be running (added in Phase 2) before Telemetry configs point to it. OTel SDK must be initialized (via Telemetry block) before Kafka manual propagation code can call `otel.GetTextMapPropagator()`. This is the only phase requiring non-trivial Go code changes — isolating it after infrastructure is stable reduces debugging ambiguity.

**Delivers:** All three services emit traces to Jaeger; full three-hop trace visible in Jaeger (HTTP redirect → Kafka → consumer insert); `ClickEvent` struct has `TraceContext` field; manual span creation in consumer `Consume()` using span links.

**Addresses:** Telemetry YAML config (P1), Kafka trace propagation (P1), end-to-end trace chain

**Avoids:**
- Wrong Batcher value — use `otlpgrpc` not `jaeger` (PITFALL 2)
- Kafka trace broken — embed `TraceContext` in ClickEvent JSON; verify with Jaeger three-hop trace (PITFALL 1)
- Consumer YAML missing Telemetry block — add to both `consumer.yaml` and `consumer-docker.yaml` (PITFALL 3)
- Consumer relying on auto-instrumentation — add explicit `tracer.Start()` in `Consume()`

**Research flag:** The Kafka propagation via `TraceContext` JSON field should be smoke-tested as acceptance criteria. PITFALLS.md notes that go-queue's consumer-side `startConsumers()` already extracts OTel from Kafka headers — if this is confirmed, the consumer may already receive valid span context in the `ctx` argument to `Consume()`, making JSON extraction redundant. Verify during implementation to choose the simpler approach.

### Phase 4: Prometheus Metrics and Grafana Dashboards

**Rationale:** Prometheus must scrape before Grafana dashboards have data to display. Building dashboards last ensures all metrics are visible and queries can be verified during dashboard construction. Custom business metrics are added here after built-in go-zero metrics are confirmed working.

**Delivers:** `prometheus.yml` scrape config targeting DevServer ports; all three services show UP in Prometheus Status > Targets; Grafana provisioned with Prometheus datasource; go-zero dashboard (Grafana Labs 19909) loaded automatically; all latency panels use `histogram_quantile(0.99)` not `avg()`.

**Addresses:** Prometheus Docker Compose (P1), Grafana provisioning (P1), go-zero dashboard (P1), custom business metrics (P2)

**Avoids:**
- Scraping wrong ports — use DevServer ports 6470/6471/6472 not 8080/8081 (PITFALL 10)
- Metrics disabled — verify `curl :6470/metrics | grep http_server` returns data before building dashboards (PITFALL 6)
- Cardinality explosion — never use `short_code`, `url`, or IP as Prometheus label; code-review all custom metric labels (PITFALL 5)
- Grafana using `avg()` for latency — all panels must use `histogram_quantile()`, RED method (PITFALL 7)
- Grafana datasource using `localhost:9090` instead of `prometheus:9090`

**Research flag:** Standard patterns — skip `/gsd:research-phase`. Prometheus scrape config and Grafana provisioning are well-documented. Verify that go-zero dashboard 19909 metric names match what go-zero v1.10.0 actually exports from DevServer.

### Phase Ordering Rationale

- **Tech debt first** — zero dependencies on new infrastructure; clean baseline before adding complexity.
- **Etcd second** — YAML-only changes, establishes Docker Compose expansion pattern, and the infrastructure containers added here (Jaeger, Prometheus) are prerequisites for Phase 3 verification without requiring a separate PR.
- **Tracing third** — requires Jaeger running (from Phase 2) and is the only phase with meaningful Go code changes; isolating it after infrastructure is stable reduces debugging ambiguity.
- **Metrics/dashboards last** — requires Prometheus scraping data before dashboards are meaningful; building dashboards before metrics are confirmed working wastes effort.

This ordering follows ARCHITECTURE.md's explicit build-order recommendation and maps each phase's deliverable to the next phase's prerequisite.

### Research Flags

Phases needing deeper research during planning:
- **Phase 3 (Distributed Tracing):** Verify whether go-queue's consumer-side `startConsumers()` already extracts OTel context from Kafka headers into the `ctx` argument passed to `Consume()`. If confirmed, the consumer-side code change may be simpler than expected (just `tracer.Start(ctx, ...)`, no manual extraction). Investigate during execution; if issues arise with the JSON field approach, fall back to using `segmentio/kafka-go` producer directly for header-level injection.

Phases with standard patterns (skip research-phase):
- **Phase 1 (Tech Debt):** Dead code deletion, test addition, env var — no research needed.
- **Phase 2 (Etcd):** go-zero etcd YAML config documented and confirmed against source. Docker Compose healthcheck patterns are standard.
- **Phase 4 (Prometheus + Grafana):** Scrape config and Grafana provisioning are well-documented. Standard RED method dashboard patterns.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | All technology choices verified against official sources and go.mod. Version numbers confirmed against release notes. Key insight (no new Go deps needed) verified by direct go.mod analysis. Bitnami etcd deprecation confirmed. Jaeger v1 EOL confirmed. |
| Features | HIGH | go-zero built-in metrics and instrumentation boundaries confirmed against go-zero v1.10.0 source. Feature prioritization is clear given the project context. go-zero built-in metric names verified. |
| Architecture | HIGH (HTTP/gRPC/etcd), MEDIUM (Kafka propagation) | go-zero Telemetry and etcd integration confirmed against source code. Kafka trace propagation via JSON field is inferred from `kq.Pusher` source inspection — JSON field is a reasonable workaround that should be verified with a smoke test during Phase 3. |
| Pitfalls | HIGH | Critical pitfalls verified against go-zero v1.10.0 and go-queue v1.2.2 source code. Recovery strategies and warning signs are specific and actionable. |

**Overall confidence:** HIGH

### Gaps to Address

- **Kafka propagation smoke test:** Before marking Phase 3 complete, verify end-to-end trace chain in Jaeger (HTTP redirect → Kafka → consumer) shows as a connected trace. If broken, debug with `Batcher: file` to log exported spans locally.

- **go-queue consumer-side OTel extraction:** PITFALLS.md source analysis notes that go-queue's `startConsumers()` already calls `otel.GetTextMapPropagator().Extract()` from Kafka headers (via `kq/internal/trace.go`). If this is correct, the consumer already receives valid span context in `ctx`. Verify during Phase 3 implementation — if confirmed, only the producer-side `TraceContext` injection may be needed, simplifying consumer changes.

- **Grafana dashboard 19909 metric name compatibility:** Dashboard 19909 on Grafana Labs was built for a specific version of go-zero's metric names. Verify that `http_server_requests_duration_ms` and related names match what go-zero v1.10.0 actually exports from DevServer before relying on this dashboard JSON. If names differ, edit the panel queries.

- **Jaeger span links UX:** Verify that Jaeger 2.15.0 UI displays span links in a discoverable way. If the async Kafka relationship is not clearly visible, add a note to the project README explaining how to navigate from HTTP redirect trace to consumer trace in Jaeger.

## Sources

### Primary (HIGH confidence)

- go-zero source code (v1.10.0): `core/trace/config.go`, `core/trace/agent.go`, `core/service/serviceconf.go`, `internal/devserver/server.go` — Telemetry config fields, Batcher options, OTel initialization lifecycle
- go-queue source code (v1.2.2): `kq/pusher.go`, `kq/queue.go`, `kq/internal/trace.go` — Push vs KPush behavior, consumer OTel extraction
- [go-zero monitoring docs](https://go-zero.dev/en/docs/tutorials/monitor/index) — DevServer metrics, built-in metric names
- [go-zero gRPC client configuration](https://go-zero.dev/en/docs/tutorials/grpc/client/conn) — EtcdConf fields
- [go-zero Service Configuration](https://go-zero.dev/en/docs/tutorials/grpc/server/configuration) — RpcServerConf Etcd fields
- [go-zero Discovery docs](https://zeromicro.github.io/go-zero.dev/en/docs/component/discovery/) — Etcd registration/discovery lifecycle
- [Jaeger Getting Started v1.76](https://www.jaegertracing.io/docs/1.76/getting-started/) — OTLP ports and all-in-one configuration
- [Jaeger v2 CNCF announcement](https://www.cncf.io/blog/2024/11/12/jaeger-v2-released-opentelemetry-in-the-core/) — v2 architecture based on OTel Collector
- [OTel Jaeger exporter deprecation](https://opentelemetry.io/blog/2023/jaeger-exporter-collector-migration/) — Jaeger exporter removed July 2023
- [Prometheus v3.9.1 release](https://github.com/prometheus/prometheus/releases/tag/v3.9.1) — current stable version
- [Grafana Provisioning docs](https://grafana.com/docs/grafana/latest/administration/provisioning/) — datasource and dashboard auto-provisioning
- [etcd releases](https://github.com/etcd-io/etcd/releases/) — v3.5.27 current 3.5.x
- Direct `go.mod` analysis — confirmed OTel, Prometheus client, and etcd client already in dependency graph
- [OTel propagation package](https://pkg.go.dev/go.opentelemetry.io/otel/propagation) — W3C TraceContext propagation API

### Secondary (MEDIUM confidence)

- [Uptrace go-zero OTel guide](https://uptrace.dev/guides/opentelemetry-go-zero) — Telemetry struct fields verification
- [go-zero zRPC etcd discovery blog](https://community.ops.io/kevwan/implementing-service-discovery-for-microservices-5bcb) — YAML config examples for server and client
- [How to Propagate Trace Context Across Kafka Producers and Consumers](https://oneuptime.com/blog/post/2026-02-06-propagate-trace-context-kafka-producers-consumers/view) — Kafka header propagation pattern
- [Kafka with OpenTelemetry | Last9](https://last9.io/blog/kafka-with-opentelemetry/) — span link pattern for async boundaries
- [Go-Zero | Grafana Labs (Dashboard 19909)](https://grafana.com/grafana/dashboards/19909-go-zero/) — pre-built dashboard availability
- [Grafana dashboard best practices](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/best-practices/) — RED method, histogram_quantile usage
- [Prometheus cardinality management | Grafana Labs](https://grafana.com/blog/how-to-manage-high-cardinality-metrics-in-prometheus-and-kubernetes/) — cardinality explosion prevention

### Tertiary (LOW confidence)

- [etcd data_dir permission denied | coreos/bugs](https://github.com/coreos/bugs/issues/2446) — tmpfs workaround for volume permissions (workaround confirmed, edge case)

---
*Research completed: 2026-02-22*
*Ready for roadmap: yes*
