# Requirements: Go URL Shortener

**Defined:** 2026-02-22
**Core Value:** Shorten a long URL and reliably redirect anyone who visits the short link

## v3.0 Requirements

Requirements for production readiness milestone. Each maps to roadmap phases.

### Tech Debt

- [ ] **DEBT-01**: Remove dead IncrementClickCount method and unused click_count column logic
- [ ] **DEBT-02**: Improve analytics-consumer test coverage above 80%
- [ ] **DEBT-03**: Configure Kafka click-events topic with explicit retention settings

### Tracing

- [ ] **TRACE-01**: Services emit OpenTelemetry traces via Telemetry YAML config (HTTP + zRPC auto-instrumented)
- [ ] **TRACE-02**: Kafka click events propagate trace context across producer/consumer boundary
- [ ] **TRACE-03**: Jaeger receives OTLP traces and provides trace visualization in Docker Compose

### Logging

- [ ] **LOG-01**: Services emit structured JSON logs via go-zero logx
- [ ] **LOG-02**: Loki aggregates logs from all three services in Docker Compose
- [ ] **LOG-03**: Log shipper (Alloy/Promtail) collects container logs and forwards to Loki

### Metrics & Dashboards

- [ ] **METRICS-01**: Prometheus scrapes built-in metrics from all service DevServer endpoints
- [ ] **METRICS-02**: Grafana auto-provisions datasources (Jaeger, Loki, Prometheus) on startup
- [ ] **METRICS-03**: Grafana includes pre-built dashboard with RED metrics (rate, errors, duration) per service

## Future Requirements

### Service Discovery

- **DISC-01**: Etcd service discovery for zRPC replacing hardcoded DNS
- **DISC-02**: Dynamic service registration and health-based deregistration

### Advanced Observability

- **OBS-01**: Custom business metrics (redirects/sec, shortens/sec, enrichment latency)
- **OBS-02**: Alerting rules for error rate spikes and latency degradation
- **OBS-03**: Trace-to-log and trace-to-metric correlation in Grafana

## Out of Scope

| Feature | Reason |
|---------|--------|
| Etcd service discovery | Deferred — hardcoded DNS sufficient for current scale |
| OTel Collector | Unnecessary at this scale — Jaeger receives OTLP natively |
| Tempo | Using Jaeger for traces — Grafana has native Jaeger datasource |
| Custom business metrics | Deferred to future milestone — built-in metrics sufficient for v3.0 |
| Alerting rules | Deferred — need baseline metrics data before defining thresholds |
| Kubernetes/Helm | Docker Compose only per project constraints |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| DEBT-01 | TBD | Pending |
| DEBT-02 | TBD | Pending |
| DEBT-03 | TBD | Pending |
| TRACE-01 | TBD | Pending |
| TRACE-02 | TBD | Pending |
| TRACE-03 | TBD | Pending |
| LOG-01 | TBD | Pending |
| LOG-02 | TBD | Pending |
| LOG-03 | TBD | Pending |
| METRICS-01 | TBD | Pending |
| METRICS-02 | TBD | Pending |
| METRICS-03 | TBD | Pending |

**Coverage:**
- v3.0 requirements: 12 total
- Mapped to phases: 0
- Unmapped: 12 ⚠️

---
*Requirements defined: 2026-02-22*
*Last updated: 2026-02-22 after initial definition*
