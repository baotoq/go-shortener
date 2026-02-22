# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-22)

**Core value:** Shorten a long URL and reliably redirect anyone who visits the short link
**Current focus:** v3.0 Production Readiness — Phase 12: Tech Debt Cleanup

## Current Position

Phase: 12 of 15 (Tech Debt Cleanup)
Plan: 2 of 2 in current phase
Status: Phase complete
Last activity: 2026-02-22 — completed 12-01 (dead code removal + Kafka retention config)

Progress: [█░░░░░░░░░] 10% (v3.0 phases)

## Performance Metrics

**Velocity:**
- Total plans completed: 33 (v1.0: 18, v2.0: 15)
- Average duration: 218s (v2.0 phases 7-11)
- Total execution time: ~3330s (v2.0 actual)

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| v1.0 (1-6) | 18 | - | - |
| v2.0 (7-11) | 15 | ~3330s | ~222s |

*Updated after each plan completion*

## Accumulated Context

### Decisions

All decisions logged in PROJECT.md Key Decisions table.

Key v3.0 decisions from research:
- Use `Batcher: otlpgrpc` (not `jaeger`) — Jaeger exporter removed from OTel Go in 2023
- Kafka trace propagation via `TraceContext string` field in ClickEvent JSON — kq.Pusher does not expose Kafka headers
- Use span links (not parent-child) for async Kafka boundary in Jaeger
- Prometheus scrapes DevServer ports (6470/6471/6472), not main service ports (8080/8081)
- Grafana provisioned from `infra/grafana/provisioning/` — no manual UI setup
- No OTel Collector — Jaeger v2 accepts OTLP natively on port 4317

Key v3.0 decisions from 12-02:
- GeoIPReader interface in svc package (co-located with ServiceContext) — interface-typed local var in NewServiceContext avoids typed-nil pitfall with concrete *geoip2.Reader

Key v3.0 decisions from 12-01:
- Removed IncrementClickCount from interface + impl + mock + test atomically to keep compile-time assertion valid
- Kafka retention at broker level (24h time, 1GB size, 100MB segments) — applies to all auto-created topics including click-events

### Pending Todos

None.

### Blockers/Concerns

- Phase 13 (Tracing): Verify during implementation whether go-queue's consumer-side `startConsumers()` already extracts OTel context from Kafka headers into `ctx`. If confirmed, consumer changes are simpler (just `tracer.Start(ctx, ...)`, no manual JSON extraction).
- Phase 15 (Metrics): Verify that Grafana Labs dashboard 19909 metric names match what go-zero v1.10.0 actually exports before relying on it.

## Session Continuity

Last session: 2026-02-22
Stopped at: Completed 12-01-PLAN.md (dead code removal + Kafka retention)
Resume file: None
Next action: Phase 12 complete — begin Phase 13 (Tracing)

---
*Initialized: 2026-02-14*
*Last updated: 2026-02-22 after 12-01 (dead code removal + Kafka retention — Phase 12 complete)*
