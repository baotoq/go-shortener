# Milestones

## v1.0 MVP (Shipped: 2026-02-15)

**Phases completed:** 6 phases, 18 plans
**Go LOC:** 9,401 across 56 files
**Timeline:** 2026-02-14 → 2026-02-15
**Git range:** 24e03bf..7170c9e

**Key accomplishments:**
- URL shortening API with NanoID 8-char codes, validation, rate limiting, and 302 redirect
- Event-driven click analytics via Dapr pub/sub (fire-and-forget, never blocks redirect)
- Rich analytics enrichment: GeoIP country detection, device/browser parsing, referer classification
- Link management with cursor pagination, filtering, search, and cross-service click count enrichment
- Production infrastructure: Docker Compose with Dapr sidecars, GitHub Actions CI/CD, golangci-lint
- Test coverage hardened to 90.7% (handler package), Dapr integration tests with real Docker containers

**Known tech debt:**
- SQLite single-writer limitation (SetMaxOpenConns(1))
- In-memory pub/sub (dev only, production needs Redis/RabbitMQ)
- No container registry (Docker images built but not pushed)

**Archive:** `.planning/milestones/v1.0-ROADMAP.md`, `.planning/milestones/v1.0-REQUIREMENTS.md`

---


## v2.0 Go-Zero Adoption (Shipped: 2026-02-22)

**Phases completed:** 5 phases (7-11), 15 plans
**Go LOC:** 3,425 across rewritten codebase (11,139 added / 11,181 removed — full rewrite)
**Timeline:** 2026-02-15 → 2026-02-16
**Git range:** v1.0..HEAD (58 commits)

**Key accomplishments:**
- Full go-zero framework adoption — .api/.proto code generation replacing Chi/Dapr architecture
- PostgreSQL migration with goctl-generated CRUD models, UUIDv7 PKs, and NanoID short codes
- Kafka messaging via go-queue replacing Dapr pub/sub with click enrichment pipeline (GeoIP/UA/referer)
- zRPC service communication with circuit breaker and graceful degradation
- Docker Compose full-stack orchestration with Prometheus metrics, health checks, and multi-stage builds
- CI pipeline rewrite with testcontainers-go integration tests and golangci-lint v2

**Known tech debt:**
- IncrementClickCount is dead code (urls.click_count always 0, no user impact)
- analytics-consumer coverage at 76.4% (GeoIP requires .mmdb file)
- No automated full-stack E2E test (manual Docker smoke test only)
- Phase 9 missing SUMMARY.md files (work committed without documentation)
- Kafka topic auto-created with broker defaults (no explicit retention config)

**Archive:** `.planning/milestones/v2.0-ROADMAP.md`, `.planning/milestones/v2.0-REQUIREMENTS.md`

---

