# Stack Research: go-zero Adoption

**Domain:** Go microservices framework migration (Chi/Dapr/SQLite to go-zero/zRPC/Kafka/PostgreSQL)
**Researched:** 2026-02-15
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| go-zero | v1.10.0 | Microservices framework with code generation | Industry-standard Go microservices framework with built-in service governance (rate limiting, circuit breaker, load shedding), OpenTelemetry tracing, and Prometheus metrics. Replaces Chi router + custom middleware entirely. Latest stable release (Feb 15, 2025) adds Go 1.23 support and bug fixes. |
| goctl | v1.10.0 | Code generation CLI | Generates handler/logic/model code from `.api` specs and zRPC services from `.proto` files. v1.10.0 adds enhanced Swagger support and improved Docker generation. Use same version as go-zero framework. |
| PostgreSQL | 13+ | Primary database | Production-grade relational DB replaces SQLite. go-zero sqlx requires PostgreSQL 13+ for full feature support. |
| jackc/pgx/v5 | v5.8.0 | PostgreSQL driver (stdlib compatibility) | Recommended PostgreSQL driver for Go. Use stdlib wrapper (`pgx/v5/stdlib`) for go-zero sqlx compatibility. Native pgx is 70-150x faster than database/sql wrappers but go-zero sqlx requires database/sql interface. v5.8.0 supports Go 1.24 and PostgreSQL 13+. |
| go-queue | v1.2.2 | Kafka/Beanstalkd pub/sub framework | go-zero's official queue abstraction wrapping segmentio/kafka-go. Provides unified API for Kafka producers/consumers with built-in error handling and retry. Latest stable (July 24, 2024). |
| segmentio/kafka-go | v0.4.50 | Kafka client library | Pure Go Kafka library with no C dependencies. Used by go-queue under the hood. v0.4.50 (Jan 15, 2025) adds DescribeGroups v5 API and Kafka 4.0 support. |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| google.golang.org/grpc | v1.78.0 | gRPC runtime | Required for zRPC. v1.78.0 (Feb 11, 2026) includes critical security enhancements (TLS, authentication). Automatically pulled by go-zero. |
| google.golang.org/protobuf | v1.36+ | Protocol Buffers runtime | Required for `.proto` code generation. Use `protoc-gen-go@v1.28.0` and `protoc-gen-go-grpc@v1.3.0` for goctl compatibility. |
| github.com/jaevor/go-nanoid | latest | Short code generation | Continue using existing NanoID library for 8-char short codes. go-zero doesn't provide ID generation. |
| github.com/oschwald/geoip2-golang/v2 | v2.0+ | GeoIP country lookup | Continue using for analytics enrichment. v2.0 (56% fewer allocations, 34% less memory) now uses `netip.Addr` instead of `net.IP` for Go 1.24 alignment. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| goctl | Code generation from `.api` and `.proto` files | Install: `go install github.com/zeromicro/go-zero/tools/goctl@v1.10.0` |
| protoc | Protocol Buffers compiler | Required for zRPC. Install via package manager or from protobuf releases. |
| protoc-gen-go | Protobuf Go plugin | Install: `go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0` |
| protoc-gen-go-grpc | gRPC Go plugin | Install: `go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0` |
| Docker Compose | Local orchestration | PostgreSQL, Kafka, Zookeeper, URL Service, Analytics Service |

## Installation

```bash
# Core framework and CLI
go get github.com/zeromicro/go-zero@v1.10.0
go install github.com/zeromicro/go-zero/tools/goctl@v1.10.0

# Kafka integration
go get github.com/zeromicro/go-queue@v1.2.2

# PostgreSQL driver (stdlib wrapper for go-zero sqlx)
go get github.com/jackc/pgx/v5@v5.8.0

# zRPC/gRPC dependencies (auto-installed by go-zero, but pinned for consistency)
go get google.golang.org/grpc@v1.78.0
go get google.golang.org/protobuf@latest

# Protobuf code generators
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Existing libraries to KEEP
go get github.com/jaevor/go-nanoid@latest
go get github.com/oschwald/geoip2-golang/v2@latest
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| go-zero sqlx + goctl model | sqlc | Use sqlc if you need complex SQL queries with compile-time validation. go-zero sqlx generates basic CRUD + caching but doesn't validate SQL syntax at compile time. For this project, go-zero sqlx is sufficient (simple queries, framework integration). |
| jackc/pgx/v5 (stdlib) | lib/pq | Never. lib/pq is deprecated and unmaintained. pgx is 70-150x faster and actively developed. |
| go-queue (Kafka) | confluent-kafka-go | Use confluent-kafka-go if you need Confluent-specific features (Schema Registry, ksqlDB). go-queue is pure Go (no CGo), simpler, and integrates natively with go-zero. |
| zRPC (gRPC) | Dapr service invocation | Use Dapr if multi-language polyglot services are required. For Go-only microservices, zRPC removes abstraction layer and provides better performance. |
| PostgreSQL | SQLite | Use SQLite for single-writer, embedded scenarios. PostgreSQL removes single-writer limitation and provides production-grade features (replication, JSONB, full-text search). |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Chi router | go-zero provides HTTP routing via `.api` specs | go-zero rest package with `.api` file definitions |
| Dapr | Adds unnecessary abstraction layer when go-zero provides native zRPC + queue | go-zero zRPC for service calls, go-queue for events |
| sqlc | Redundant when using go-zero sqlx model generation | goctl model for PostgreSQL code generation |
| Custom middleware (rate limit, circuit breaker) | go-zero has built-in service governance | go-zero built-in features (configured via YAML) |
| Zap logger | go-zero has built-in structured logging | go-zero logx package |
| Manual Prometheus metrics | go-zero auto-exports metrics on port 6470 | go-zero built-in telemetry (custom metrics via core/metric) |

## Stack Patterns by Variant

**For HTTP API services (URL Service):**
- Define routes in `.api` file with request/response types
- Generate handler/logic/types with `goctl api go -api url.api -dir .`
- Use go-zero sqlx models for database access
- Enable built-in rate limiting, circuit breaker in service config YAML
- Metrics auto-exported on `:6470/metrics`, health on `:6470/healthz`

**For RPC services (Analytics Service):**
- Define service in `.proto` file with request/response messages
- Generate zRPC server/client with `goctl rpc protoc analytics.proto --go_out=. --go-grpc_out=. --zrpc_out=.`
- Consume Kafka events via go-queue `kq.MustNewQueue()`
- Call other services via generated zRPC clients

**For async event consumers:**
- Use go-queue `kq.MustNewQueue()` with consumer struct implementing `Consume(key, val string) error`
- Configure via `KqConf` (brokers, group, topic, offset, consumers, processors)
- go-queue handles deserialization, error handling, and retries

**For database access:**
- Generate models from existing PostgreSQL database: `goctl model pg datasource -url="..." -table="..." -dir=./model`
- Or define DDL and generate: Use MySQL DDL approach (PostgreSQL DDL generation not supported, requires live DB connection)
- Models include CRUD operations, optional caching (Redis), and connection pooling

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| go-zero@v1.10.0 | goctl@v1.10.0 | MUST match framework version |
| goctl@v1.10.0 | protoc-gen-go@v1.28.0, protoc-gen-go-grpc@v1.3.0 | Recommended versions for zRPC generation |
| go-queue@v1.2.2 | segmentio/kafka-go@v0.4.50 | go-queue wraps kafka-go, version mismatch unlikely to cause issues |
| jackc/pgx/v5@v5.8.0 | PostgreSQL 13+ | Minimum PostgreSQL 13 for go-zero sqlx features |
| google.golang.org/grpc@v1.78.0 | google.golang.org/protobuf@v1.36+ | Use matching major versions |
| geoip2-golang/v2 | Go 1.24 | v2.0+ uses `netip.Addr` (Go 1.18+), v1.x uses deprecated `net.IP` |

## Integration with Existing Code

### Keep Using (No Changes)

| Library | Reason | Integration |
|---------|--------|-------------|
| github.com/jaevor/go-nanoid | go-zero doesn't provide ID generation | Call from logic layer when creating URLs |
| github.com/oschwald/geoip2-golang/v2 | go-zero doesn't provide GeoIP | Inject into service context, use in analytics logic |
| testcontainers-go | Testing strategy unchanged | Use for PostgreSQL and Kafka integration tests |

### Replace Entirely

| Old | New | Migration Path |
|-----|-----|----------------|
| Chi router | go-zero `.api` specs | Rewrite routes as `.api` file, regenerate with goctl |
| Dapr pub/sub | go-queue (Kafka) | Replace Dapr publish with `kq.Pusher.Push()`, Dapr subscribe with `kq.MustNewQueue()` |
| Dapr service invocation | zRPC clients | Define `.proto`, generate client, call via `client.Method(ctx, req)` |
| sqlc | goctl model (PostgreSQL) | Generate models from live PostgreSQL connection or migrate DDL |
| Custom middleware | go-zero built-in | Remove rate limiter, configure in service YAML |
| pkg/problemdetails | go-zero error handling | Use go-zero `httpx.Error()` with custom error types |

### Built-in Features (Replace Custom Code)

| Feature | go-zero Built-in | Configuration |
|---------|------------------|---------------|
| Rate limiting | Yes (token bucket) | Service config YAML: `MaxConns`, `MaxBytes` |
| Circuit breaker | Yes (Google SRE algorithm, 10s sliding window) | Automatic, no config needed |
| Load shedding | Yes (adaptive) | Automatic based on CPU/memory |
| Timeout control | Yes (chained) | Service config YAML: `Timeout` |
| Logging | Yes (logx package, structured JSON) | Service config YAML: `Log.Mode`, `Log.Level` |
| Metrics | Yes (Prometheus on `:6470/metrics`) | Auto-enabled, custom metrics via `core/metric` |
| Tracing | Yes (OpenTelemetry) | Service config YAML: `Telemetry.Name`, `Telemetry.Endpoint` |
| Health checks | Yes (`/healthz` on `:6470`) | Auto-enabled |

## Sources

- [go-zero GitHub Releases](https://github.com/zeromicro/go-zero/releases) - v1.10.0 verified (Feb 15, 2025)
- [goctl Installation](https://go-zero.dev/en/docs/tasks/installation/goctl) - Installation methods
- [go-zero Kafka Integration](https://go-zero.dev/en/docs/tutorials/message-queue/kafka) - go-queue usage
- [go-queue GitHub](https://github.com/zeromicro/go-queue) - v1.2.2 verified (July 24, 2024)
- [goctl RPC Documentation](https://go-zero.dev/en/docs/tutorials/cli/rpc) - zRPC code generation
- [jackc/pgx v5 Docs](https://pkg.go.dev/github.com/jackc/pgx/v5) - v5.8.0 verified (Dec 26, 2025)
- [segmentio/kafka-go Releases](https://github.com/segmentio/kafka-go/releases) - v0.4.50 verified (Jan 15, 2025)
- [go-zero Monitoring](https://go-zero.dev/en/docs/tutorials/monitor/index) - Built-in telemetry features
- [gRPC Quickstart](https://grpc.io/docs/languages/go/quickstart/) - protoc plugin versions
- [go-zero API Syntax](https://go-zero.dev/en/docs/tasks/dsl/api) - `.api` file format
- [goctl Model Generation](https://go-zero.dev/en/docs/tutorials/cli/model) - Database code generation
- [go-zero Architecture Evolution](https://go-zero.dev/en/docs/concepts/architecture-evolution) - Handler/Logic/Model pattern
- [go-zero Circuit Breaker](https://go-zero.dev/en/docs/tutorials/service/governance/breaker) - Built-in resilience
- [Go Database Patterns Comparison](https://dasroot.net/posts/2025/12/go-database-patterns-gorm-sqlx-pgx-compared/) - sqlx vs pgx performance
- [oschwald/geoip2-golang GitHub](https://github.com/oschwald/geoip2-golang) - v2.0+ features
- [gRPC-Go v1.78.0](https://grpc.io/docs/languages/go/quickstart/) - Security enhancements (Feb 11, 2026)

---
*Stack research for: go-zero microservices adoption*
*Researched: 2026-02-15*
