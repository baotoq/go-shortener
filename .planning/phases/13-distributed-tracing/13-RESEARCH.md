# Phase 13: Distributed Tracing - Research

**Researched:** 2026-02-22
**Domain:** OpenTelemetry + go-zero Telemetry config + Jaeger + go-queue trace propagation
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| TRACE-01 | Services emit OpenTelemetry traces via Telemetry YAML config (HTTP + zRPC auto-instrumented) | go-zero `Telemetry` block in YAML is the only change needed — `RestConf`, `RpcServerConf`, and `ServiceConf` all embed `ServiceConf` which calls `trace.StartAgent(c.Telemetry)` in `MustSetUp()`. No Go code changes required for HTTP/zRPC auto-instrumentation. |
| TRACE-02 | Kafka click events propagate trace context across producer/consumer boundary | go-queue v1.2.2 already injects OTel W3C TraceContext headers in `PushWithKey` (producer) and extracts them in `startConsumers()` (consumer). The only fix needed is passing `l.ctx` instead of `context.Background()` in the redirect logic's `GoSafe` Kafka publish call. |
| TRACE-03 | Jaeger receives OTLP traces and provides trace visualization in Docker Compose | Add `jaegertracing/all-in-one:1.76.0` to docker-compose.yaml exposing ports 4317 (OTLP gRPC), 4318 (OTLP HTTP), 16686 (UI). OTLP is built-in — no extra env vars required. |
</phase_requirements>

---

## Summary

go-zero v1.10.0 has a first-class `Telemetry` YAML block (part of `service.ServiceConf`) that initializes a global OpenTelemetry TracerProvider on `MustSetUp()`. The `Batcher` option in go-zero v1.10.0 defaults to `otlpgrpc` (not `jaeger` — the Jaeger-specific OTel exporter was removed from the OTel Go SDK in 2023). Sending traces to Jaeger uses `Batcher: otlpgrpc` with `Endpoint: jaeger:4317`.

All three service types (REST via `rest.RestConf`, zRPC via `zrpc.RpcServerConf`, standalone via `service.ServiceConf`) embed `service.ServiceConf`, so adding a `Telemetry` block to the YAML config is the only change required for HTTP and zRPC auto-instrumentation.

For the Kafka async boundary (TRACE-02), go-queue v1.2.2 already implements full OTel trace propagation: `PushWithKey` injects W3C TraceContext headers into Kafka message headers, and `startConsumers()` extracts them into `ctx` before calling the `ConsumeHandler`. The one bug in the current codebase is that `redirectlogic.go` passes `context.Background()` (not `l.ctx`) to `KqPusher.Push()`, which breaks the trace chain. Fixing this single line completes TRACE-02.

**Primary recommendation:** Add `Telemetry` blocks to all six YAML config files (three service configs × two environments: local + docker), add Jaeger to docker-compose.yaml, and fix the `context.Background()` → `l.ctx` bug in `redirectlogic.go`. No new Go dependencies needed — OTel packages are already transitive dependencies of go-zero v1.10.0.

---

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go-zero` trace package | v1.10.0 (bundled) | Initializes global OTel TracerProvider via `Telemetry` YAML config | Built into go-zero — no code change required |
| `go-queue` kq | v1.2.2 (bundled) | Injects/extracts OTel context in Kafka message headers | Already in go.mod; already implements propagation natively |
| `jaegertracing/all-in-one` | 1.76.0 | Trace backend + UI; accepts OTLP natively on port 4317 | Standard dev trace backend; no OTel Collector needed |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | already indirect dep | OTLP gRPC exporter used by go-zero internally | Already present — no `go get` required |
| `go.opentelemetry.io/otel/propagation` | already indirect dep | W3C TraceContext propagator — set globally in `go-zero/core/trace/propagation.go` `init()` | No action needed — go-zero's `init()` sets the global propagator |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `Batcher: otlpgrpc` | `Batcher: jaeger` | `jaeger` batcher was removed from OTel Go SDK in 2023; `otlpgrpc` is the correct replacement — Jaeger v1.35+ accepts OTLP natively |
| `Batcher: otlpgrpc` | `Batcher: otlphttp` | HTTP works but gRPC is lower overhead and more common for internal observability pipelines |
| Direct Jaeger | OTel Collector → Jaeger | OTel Collector adds unnecessary complexity at this scale; Jaeger accepts OTLP natively on port 4317 (confirmed in REQUIREMENTS.md Out of Scope) |

**Installation:** No new packages. All OTel packages are already transitive dependencies of `github.com/zeromicro/go-zero v1.10.0` and `github.com/zeromicro/go-queue v1.2.2`.

---

## Architecture Patterns

### How go-zero Telemetry Auto-Instrumentation Works

All three service configuration types embed `service.ServiceConf`:

```go
// ServiceConf (go-zero/core/service/serviceconf.go) includes:
type ServiceConf struct {
    Name      string
    Log       logx.LogConf
    Telemetry trace.Config `json:",optional"`
    // ...
}

// MustSetUp calls trace.StartAgent(c.Telemetry)
// which initializes the global TracerProvider once.
```

- `rest.RestConf` embeds `ServiceConf` → `server.Start()` uses `TracingHandler` middleware automatically
- `zrpc.RpcServerConf` embeds `ServiceConf` → gRPC server uses tracing interceptors automatically
- `service.ServiceConf` (analytics-consumer) → `MustSetUp()` calls `trace.StartAgent` manually

For analytics-consumer, `c.MustSetUp()` is already called in `main()`. Adding `Telemetry` to consumer.yaml is sufficient.

### YAML Configuration Pattern (All Three Services)

```yaml
# Add to all three service YAML configs
Telemetry:
  Name: url-api          # service name in Jaeger UI
  Endpoint: localhost:4317   # local dev; use jaeger:4317 in docker variant
  Sampler: 1.0           # 100% sampling — appropriate for dev
  Batcher: otlpgrpc
```

For docker YAML variants, `Endpoint: jaeger:4317` (Docker service name).

### Kafka Trace Propagation — How go-queue Does It

**Producer side** (`PushWithKey` in go-queue v1.2.2):
```go
// Source: go-queue@v1.2.2/kq/pusher.go:128-143
func (p *Pusher) PushWithKey(ctx context.Context, key, v string) error {
    msg := kafka.Message{Key: []byte(key), Value: []byte(v)}
    mc := internal.NewMessageCarrier(internal.NewMessage(&msg))
    // Injects W3C traceparent header into Kafka message headers
    otel.GetTextMapPropagator().Inject(ctx, mc)
    // ...
}
```

**Consumer side** (`startConsumers` in go-queue v1.2.2):
```go
// Source: go-queue@v1.2.2/kq/queue.go:219-246
func (q *kafkaQueue) startConsumers() {
    for i := 0; i < q.c.Processors; i++ {
        q.consumerRoutines.Run(func() {
            for msg := range q.channel {
                mc := internal.NewMessageCarrier(internal.NewMessage(&msg))
                // Extracts trace context from Kafka headers into ctx
                ctx := otel.GetTextMapPropagator().Extract(context.Background(), mc)
                ctx = contextx.ValueOnlyFrom(ctx)
                // ctx now carries the extracted trace context
                if err := q.consumeOne(ctx, string(msg.Key), string(msg.Value)); err != nil {
                    // ...
                }
            }
        })
    }
}
```

**Global propagator** is set in go-zero's `trace/propagation.go` `init()`:
```go
// Source: go-zero@v1.10.0/core/trace/propagation.go
func init() {
    otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
        propagation.TraceContext{}, propagation.Baggage{}))
}
```

This `init()` runs when any go-zero package is imported. The global propagator is set before any service starts. go-queue's `PushWithKey` and `startConsumers()` use `otel.GetTextMapPropagator()` which retrieves this global propagator.

### The One Required Code Fix: Context in redirect Kafka Publish

Current code passes `context.Background()` to `Push()`, breaking the trace chain:

```go
// CURRENT (broken trace chain):
// go-queue@v1.2.2/kq/pusher.go Push() calls PushWithKey() with the ctx
// but we pass context.Background() — no span in this context
if pushErr := l.svcCtx.KqPusher.Push(context.Background(), string(payload)); pushErr != nil {
```

Fix: pass `l.ctx` which carries the active HTTP span:

```go
// FIXED (trace chain preserved):
if pushErr := l.svcCtx.KqPusher.Push(l.ctx, string(payload)); pushErr != nil {
```

Note: `threading.GoSafe` runs the closure in a goroutine. `l.ctx` is captured by the closure at the time `GoSafe` is called (while the HTTP handler is still active), so the span is valid when `Push()` executes. The span may have ended by the time the goroutine runs if the handler returns first, but OTel handles this gracefully — the propagated trace ID is still valid for linking the consumer span.

### Span Link vs Parent-Child for Kafka Boundary

The Kafka producer → consumer relationship is **asynchronous**, so the consumer span should NOT be a child span of the HTTP span (that would imply blocking). Instead, go-queue's consumer creates a **new root span** from the extracted context (via `context.Background()` as base), making the consumer span a **linked span** in Jaeger rather than a child.

The STATE.md decision "Use span links (not parent-child) for async Kafka boundary in Jaeger" is automatically satisfied by go-queue's implementation — the consumer extracts context from headers into a fresh `context.Background()`, which means Jaeger shows the spans as linked (same trace ID, different span hierarchy).

### Docker Compose: Adding Jaeger

```yaml
# Add to docker-compose.yaml services section
jaeger:
  image: jaegertracing/all-in-one:1.76.0
  ports:
    - "16686:16686"   # Jaeger UI
    - "4317:4317"     # OTLP gRPC
    - "4318:4318"     # OTLP HTTP (optional, for reference)
  environment:
    - COLLECTOR_OTLP_ENABLED=true   # safe to include; built-in on 1.76.0
```

Services (url-api, analytics-rpc, analytics-consumer) should declare `depends_on: jaeger: condition: service_started`.

### Anti-Patterns to Avoid

- **Using `Batcher: jaeger`**: The Jaeger-specific OTel exporter is not supported in go-zero v1.10.0. The valid options are `zipkin|otlpgrpc|otlphttp|file`. Use `otlpgrpc`.
- **Propagating via JSON `TraceContext` field in ClickEvent**: The STATE.md had this as a planned approach but it is NOT needed. go-queue v1.2.2 already propagates via Kafka headers natively. Adding a `TraceContext string` field to `ClickEvent` and extracting it manually would be redundant and incorrect.
- **Passing `context.Background()` to `KqPusher.Push()`**: The current code does this. It must be fixed to `l.ctx`.
- **Hand-rolling span creation in HTTP handlers**: go-zero's `TracingHandler` middleware creates spans automatically for all HTTP routes. No manual `tracer.Start()` calls needed in handlers or logic.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| HTTP request span creation | Manual `tracer.Start(ctx, "redirect")` in handlers | go-zero `Telemetry` YAML config | `TracingHandler` middleware does this for all routes automatically |
| zRPC span propagation | Manual gRPC interceptors | go-zero `Telemetry` YAML config | go-zero's zrpc interceptors handle client and server spans automatically |
| Kafka header injection/extraction | `TraceContext string` in ClickEvent JSON body | go-queue v1.2.2 `PushWithKey` / `startConsumers()` | Already implemented in go-queue — uses OTel `TextMapPropagator` on Kafka message headers |
| OTel global propagator setup | `otel.SetTextMapPropagator(...)` in main() | go-zero's `trace/propagation.go` init() | go-zero sets the W3C TraceContext propagator globally at package init — no action required |
| Consumer span creation | Manual `tracer.Start(ctx, "consume")` in `ClickEventConsumer.Consume()` | Kafka header extraction in go-queue | The `ctx` passed to `Consume()` already has the extracted trace context; any spans started from this ctx will be automatically linked |

**Key insight:** The entire distributed trace chain (HTTP → zRPC → Kafka headers → consumer) is handled by the combination of go-zero's `Telemetry` YAML config and go-queue v1.2.2's built-in OTel support. The total code change required is: one bug fix (1 line), six YAML file edits, and one Docker Compose addition.

---

## Common Pitfalls

### Pitfall 1: Wrong Batcher Value
**What goes wrong:** Setting `Batcher: jaeger` in YAML causes a runtime error "unknown exporter: jaeger" because go-zero v1.10.0 removed the Jaeger-specific exporter.
**Why it happens:** go-zero docs and examples online were updated when OTel dropped the Jaeger exporter in 2023, but older examples still show `Batcher: jaeger`.
**How to avoid:** Always use `Batcher: otlpgrpc` with `Endpoint: <jaeger-host>:4317`.
**Warning signs:** Service startup log shows "[otel] error: unknown exporter: jaeger".

### Pitfall 2: `context.Background()` in GoSafe Kafka Publish
**What goes wrong:** The producer injects an empty context into Kafka headers — no `traceparent` header. The consumer starts a fresh root trace with no connection to the HTTP span. Jaeger shows the redirect HTTP trace and the consumer DB insert trace as completely unrelated.
**Why it happens:** The existing `redirectlogic.go` line 72 uses `context.Background()` instead of `l.ctx`.
**How to avoid:** Pass `l.ctx` to `l.svcCtx.KqPusher.Push(l.ctx, ...)`.
**Warning signs:** Jaeger shows click events as isolated traces with no connection to redirect spans.

### Pitfall 3: analytics-consumer Missing Telemetry Initialization
**What goes wrong:** If `Telemetry` is added to `consumer.yaml` but `c.MustSetUp()` is not called, the TracerProvider is never initialized.
**Why it happens:** analytics-consumer uses `service.ServiceConf` directly — `MustSetUp()` must be called explicitly.
**How to avoid:** The existing `analytics-consumer.go` already calls `c.MustSetUp()`. Adding `Telemetry` to the YAML is sufficient.
**Warning signs:** No spans from analytics-consumer appear in Jaeger.

### Pitfall 4: Local Dev Endpoint vs Docker Endpoint
**What goes wrong:** Using `Endpoint: jaeger:4317` in local dev YAML files (where `jaeger` DNS name doesn't resolve) causes silent trace loss with "[otel] error: dial failed" messages.
**Why it happens:** Docker service names only resolve inside the Docker network.
**How to avoid:** Local YAML files use `Endpoint: localhost:4317`; docker YAML files use `Endpoint: jaeger:4317`.
**Warning signs:** Local services start without error but no traces appear in Jaeger.

### Pitfall 5: COLLECTOR_OTLP_ENABLED Not Needed on Jaeger 1.76
**What goes wrong:** Including `COLLECTOR_OTLP_ENABLED=true` is harmless but may suggest OTLP is disabled by default.
**Why it happens:** This env var was required on older Jaeger images (< 1.35). On 1.76, OTLP is built-in on ports 4317/4318.
**How to avoid:** Either include it (safe) or omit it — no functional difference on 1.76.
**Warning signs:** None — this is a non-issue.

---

## Code Examples

### YAML Telemetry Block (local dev)

```yaml
# Source: go-zero v1.10.0 core/trace/config.go + verified against go-zero docs
Telemetry:
  Name: url-api
  Endpoint: localhost:4317
  Sampler: 1.0
  Batcher: otlpgrpc
```

### YAML Telemetry Block (docker)

```yaml
# In *-docker.yaml variants — same but Endpoint uses Docker service name
Telemetry:
  Name: url-api
  Endpoint: jaeger:4317
  Sampler: 1.0
  Batcher: otlpgrpc
```

### Fixed Kafka Publish in redirectlogic.go

```go
// Source: go-queue@v1.2.2/kq/pusher.go — Push() calls PushWithKey(ctx, ...)
// which injects otel.GetTextMapPropagator().Inject(ctx, messageCarrier)
// BEFORE this fix: context.Background() — no trace context injected
// AFTER this fix: l.ctx — active HTTP span context injected into Kafka headers
if pushErr := l.svcCtx.KqPusher.Push(l.ctx, string(payload)); pushErr != nil {
```

### Docker Compose Jaeger Service

```yaml
# Source: jaegertracing/all-in-one:1.76.0 — OTLP built-in on 4317/4318
jaeger:
  image: jaegertracing/all-in-one:1.76.0
  ports:
    - "16686:16686"   # Jaeger UI
    - "4317:4317"     # OTLP gRPC receiver
    - "4318:4318"     # OTLP HTTP receiver
  environment:
    - COLLECTOR_OTLP_ENABLED=true
```

### How go-queue Propagates Trace Context (for reference — no code change needed)

```go
// Source: go-queue@v1.2.2/kq/pusher.go:128-143 — producer injects context
func (p *Pusher) PushWithKey(ctx context.Context, key, v string) error {
    msg := kafka.Message{...}
    mc := internal.NewMessageCarrier(internal.NewMessage(&msg))
    otel.GetTextMapPropagator().Inject(ctx, mc)  // injects traceparent header
    // ...
}

// Source: go-queue@v1.2.2/kq/queue.go:219-246 — consumer extracts context
// In startConsumers():
mc := internal.NewMessageCarrier(internal.NewMessage(&msg))
ctx := otel.GetTextMapPropagator().Extract(context.Background(), mc)  // extracts traceparent
ctx = contextx.ValueOnlyFrom(ctx)
q.consumeOne(ctx, ...)  // ctx carries extracted trace context → passed to Consume()
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `Batcher: jaeger` | `Batcher: otlpgrpc` | 2023 (OTel Go SDK removed Jaeger exporter) | Jaeger v1.35+ accepts OTLP natively; no separate Jaeger exporter needed |
| Manual `TraceContext` field in Kafka message body | Kafka message headers via `TextMapPropagator` | go-queue v1.2.2 (2023) | Trace context travels in Kafka headers, not JSON body — consumer extracts automatically |
| OTel Collector → Jaeger pipeline | Direct OTLP to Jaeger | Jaeger v1.35 (2022) | Eliminates OTel Collector as a required infrastructure component for dev |

**Deprecated/outdated:**
- `go.opentelemetry.io/otel/exporters/jaeger` package: removed from OTel Go SDK; replaced by OTLP exporter
- `COLLECTOR_ZIPKIN_HOST_PORT` / `COLLECTOR_OTLP_ENABLED` env vars: legacy configuration for Jaeger < 1.35

---

## Open Questions

1. **Should analytics-consumer create explicit child spans for the DB insert?**
   - What we know: The `ctx` passed to `ClickEventConsumer.Consume()` carries the extracted OTel context. Any `tracer.Start(ctx, "insert-click")` call would create a linked child span visible in Jaeger.
   - What's unclear: Whether the success criteria ("consumer insert" hop visible in Jaeger) requires explicit span creation or whether the extracted trace context appearing in Jaeger is sufficient for the success criteria.
   - Recommendation: The extracted trace context from the Kafka headers creates the trace link in Jaeger automatically (same trace ID). An explicit span is optional but would make the "insert click" hop more clearly visible. Given the success criteria says "consumer insert" must appear as a hop, a minimal explicit span in `Consume()` is advisable.

2. **Does the `threading.GoSafe` goroutine lifetime cause span issues?**
   - What we know: `l.ctx` is captured at closure creation time. The HTTP handler returns before the goroutine runs in many cases. The span associated with `l.ctx` may be marked as ended by the time `Push()` runs.
   - What's unclear: Whether an ended span's context is still valid for OTel header injection (the `traceparent` header only needs the trace ID + span ID, not an active span).
   - Recommendation: OTel propagation reads span context from `ctx` (trace ID + span ID), not span lifecycle state. An ended span still has a valid trace context for propagation. This is not a problem.

---

## Sources

### Primary (HIGH confidence)

- `go-zero@v1.10.0/core/trace/config.go` — verified `Batcher` options: `zipkin|otlpgrpc|otlphttp|file` (default `otlpgrpc`)
- `go-zero@v1.10.0/core/trace/agent.go` — verified `createExporter()` switch: no `jaeger` case exists
- `go-zero@v1.10.0/core/trace/propagation.go` — verified global propagator set to W3C TraceContext in `init()`
- `go-queue@v1.2.2/kq/pusher.go` — verified `PushWithKey()` injects OTel context into Kafka headers
- `go-queue@v1.2.2/kq/queue.go` — verified `startConsumers()` extracts OTel context from Kafka headers into `ctx`
- `go-queue@v1.2.2/kq/internal/trace.go` — verified `MessageCarrier` implements `propagation.TextMapCarrier`
- go-zero docs (via Context7 `/websites/go-zero_dev`) — confirmed `Telemetry` YAML block and `ServiceConf` structure

### Secondary (MEDIUM confidence)

- Jaeger docs https://www.jaegertracing.io/docs/1.76/getting-started/ — confirmed ports 4317 (OTLP gRPC), 16686 (UI) for `jaegertracing/all-in-one:1.76.0`
- uptrace.dev go-zero guide — confirmed `Batcher: otlpgrpc` with `Endpoint: localhost:4317` pattern

### Tertiary (LOW confidence)

- None — all critical claims verified from source code or official docs

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — verified from go-zero + go-queue source code directly
- Architecture: HIGH — verified `Telemetry` auto-instrumentation path from source; verified go-queue inject/extract from source
- Pitfalls: HIGH — pitfalls derived from source code analysis (wrong batcher, wrong context), not speculation

**Research date:** 2026-02-22
**Valid until:** 2026-08-22 (go-zero and go-queue versions locked in go.mod; Jaeger API stable)

---

## Key Discoveries Summary

1. **go-queue v1.2.2 already handles Kafka OTel propagation natively** — the STATE.md concern "Verify whether go-queue's consumer-side startConsumers() already extracts OTel context" is CONFIRMED YES. No manual JSON TraceContext field needed.

2. **The only code change needed is one line in redirectlogic.go** — change `context.Background()` to `l.ctx` on the `KqPusher.Push()` call.

3. **`Batcher: otlpgrpc` is the correct value** (not `jaeger`) — confirmed from go-zero v1.10.0 source. The `jaeger` batcher was removed from the OTel Go SDK in 2023.

4. **Telemetry block in YAML is the ONLY change for HTTP + zRPC auto-instrumentation** — no Go code changes, no new imports, no new packages.

5. **All OTel packages are already in go.mod** as indirect dependencies — no `go get` commands needed.
