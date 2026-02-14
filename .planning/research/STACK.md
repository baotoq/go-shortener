# Stack Research

**Domain:** URL Shortener with Dapr Microservices
**Researched:** 2026-02-14
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.26 | Primary programming language | Latest stable (Feb 2026). Clean concurrency model perfect for microservices. Strong stdlib reduces dependencies. |
| Dapr Go SDK | v1.13.0 | Distributed application runtime | Official SDK for pub/sub and service invocation. Provides abstraction over infrastructure concerns. Latest stable release (Sept 2025). |
| modernc.org/sqlite | latest | Database driver | Pure Go SQLite implementation (no CGO). Single binary deployment. Perfect for learning projects with production-quality code. |
| Chi | v5.x | HTTP router | Lightweight, idiomatic, stdlib-compatible. Excellent middleware ecosystem. Preferred over stdlib ServeMux for microservices routing needs. |

### HTTP & Routing

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| go-chi/chi | v5.x | HTTP router and middleware framework | Primary router. Provides method-based routing, URL parameters, middleware composition. |
| go-chi/chi/middleware | (included) | Standard middleware collection | RequestID, logging, recovery, real IP extraction. Battle-tested components. |
| go-chi/cors | latest | CORS handling | If adding web frontend later. Not needed for API-only initially. |

### Validation & Data Handling

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| go-playground/validator/v10 | v10.x | Struct validation | Validate incoming requests. Tag-based validation reduces boilerplate. Use WithRequiredStructEnabled option for v11+ compatibility. |
| google/uuid | latest | UUID generation | Generate short URL identifiers. RFC 9562 compliant. Version 4 (random) UUIDs recommended. |

### Logging

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| log/slog | stdlib (Go 1.21+) | Structured logging | Default choice. Zero dependencies, part of stdlib, good performance. Use for all services. |
| github.com/rs/zerolog | v1.x | High-performance JSON logging | Only if benchmarks show slog is bottleneck (unlikely for this project). Provides zero-allocation logging. |

### Configuration

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| joho/godotenv | v1.x | .env file loading | Simple env var loading for local development. Load .env, access via os.Getenv(). |
| spf13/viper | v1.x | Advanced configuration | If you need multiple config sources (YAML, JSON, env), remote config, or nested structures. Overkill for simple projects. |

### Testing

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| stretchr/testify | v1.x | Test assertions and mocking | Standard testing toolkit. Use assert for tests, require for fatal checks, mock for interfaces. |
| vektra/mockery | v2.x | Mock generator | Generate mocks from interfaces. Run mockery in CI. Integrates with testify/mock. |
| golang.org/x/sync/errgroup | stdlib | Concurrent test utilities | Test concurrent operations cleanly. Not just for tests but useful there. |

### Development Tools

| Tool | Version | Purpose | Notes |
|------|---------|---------|-------|
| golangci-lint | v2.9.0 | Linting aggregator | Latest (Feb 2026). Run multiple linters in one pass. Configure via .golangci.yml. Use in CI and pre-commit. |
| go mod | stdlib | Dependency management | Use go.mod and go.sum. Run `go mod tidy` regularly. |
| Docker | latest | Containerization | Multi-stage builds. Distroless base images for production. |
| GitHub Actions | N/A | CI/CD | Go has excellent GitHub Actions support. Built-in caching, matrix testing. |

### Container Base Images

| Image | Size | Purpose | When to Use |
|-------|------|---------|-------------|
| golang:1.26 | ~800MB | Build stage | Multi-stage builds only. Contains full build toolchain. |
| gcr.io/distroless/static-debian12 | ~2MB | Production runtime | Default choice. No shell, minimal attack surface. 50% smaller than Alpine. |
| alpine:latest | ~5MB | Production runtime (alternative) | If you need shell for debugging. Still very small. Use golang:alpine for builds. |
| scratch | 0MB | Production runtime (minimal) | Most extreme. Only works with fully static binaries (CGO_ENABLED=0). Harder to debug. |

## Installation

```bash
# Initialize Go module
go mod init github.com/yourusername/go-shortener

# Core dependencies
go get github.com/dapr/go-sdk@v1.13.0
go get modernc.org/sqlite
go get github.com/go-chi/chi/v5
go get github.com/go-playground/validator/v10
go get github.com/google/uuid
go get github.com/joho/godotenv

# Testing dependencies
go get github.com/stretchr/testify
go get github.com/vektra/mockery/v2

# Install dev tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v2.9.0
go install github.com/vektra/mockery/v2@latest
```

## Alternatives Considered

| Category | Recommended | Alternative | Why Not Alternative |
|----------|-------------|-------------|---------------------|
| HTTP Router | Chi | stdlib http.ServeMux | ServeMux improved in Go 1.22 but Chi provides better middleware composition and URL params. |
| HTTP Router | Chi | Gin | Gin is faster but less idiomatic. More opinionated with custom context. Chi is stdlib-compatible. |
| Database Driver | modernc.org/sqlite | mattn/go-sqlite3 | mattn requires CGO, complicates cross-compilation. modernc is pure Go. |
| Database Layer | database/sql | GORM | GORM adds overhead and reflection. Learning project benefits from understanding raw SQL. Use database/sql directly. |
| Database Layer | database/sql | sqlx | sqlx is good middle ground but unnecessary for simple schema. Add if boilerplate becomes painful. |
| Logging | slog (stdlib) | logrus | logrus requires external dependency. slog is stdlib since Go 1.21. |
| Logging | slog | zap/zerolog | Higher performance but more complex. Premature optimization for this project. |
| Config | godotenv | viper | Viper is feature-rich but overkill. Simple env vars sufficient. |
| Validation | validator/v10 | Manual validation | validator reduces boilerplate, provides consistent error messages, industry standard. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| mattn/go-sqlite3 | Requires CGO, complicates builds and cross-compilation | modernc.org/sqlite (pure Go) |
| GORM for learning | Hides SQL complexity, uses reflection, harder to debug | database/sql with hand-written queries |
| gorilla/mux | Archived/maintenance mode as of 2022 | Chi (active, similar API) |
| Gin (for this project) | Custom context breaks stdlib compatibility, overkill for simple APIs | Chi with stdlib http.Handler |
| Old validator versions | validator v9 and older deprecated | validator/v10 with WithRequiredStructEnabled |
| Alpine without multi-stage | Large image with build tools | Multi-stage: golang:1.26 → distroless/static |
| Self-hosted GitHub runners with public repos | Security risk (code execution in your infra) | GitHub-hosted runners |

## Stack Patterns by Variant

**For Learning/Development:**
- Use godotenv for config
- Use slog with text handler for readable logs
- Use SQLite file (not :memory:) to inspect data
- Build with `go build` directly
- Run with `dapr run --app-id url-service -- go run ./cmd/url-service`

**For Production:**
- Use environment variables directly (no .env file)
- Use slog with JSON handler for structured logs
- Use distroless containers
- Build with `-ldflags="-w -s"` to strip debug info
- Run in Kubernetes with Dapr sidecar

**For CI/CD:**
- Cache Go modules (`~/.cache/go-build`, `~/go/pkg/mod`)
- Run golangci-lint before tests
- Generate mocks before running tests
- Build for multiple platforms if needed (GOOS/GOARCH)
- Use GitHub Actions caching (automatically handles Go)

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| Go 1.26 | Dapr SDK v1.13.0 | SDK supports Go 1.21+. Tested with 1.26. |
| validator/v10 | Go 1.21+ | Requires generics support (Go 1.18+). |
| Chi v5 | Go 1.16+ | Very stable. v5 is current major version. |
| modernc.org/sqlite | Go 1.22+ | Requires newer stdlib features. |
| golangci-lint v2.9.0 | Go 1.26 | v2 supports up to go1.26 as of Feb 2026. |
| distroless/static | CGO_ENABLED=0 binaries | Must build fully static. Set in Dockerfile. |

## Dockerfile Best Practices

```dockerfile
# Build stage
FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/service ./cmd/service

# Production stage
FROM gcr.io/distroless/static-debian12
COPY --from=builder /app/service /service
USER nonroot:nonroot
ENTRYPOINT ["/service"]
```

**Key points:**
- Multi-stage build: reduces final image from ~800MB to ~10MB
- `CGO_ENABLED=0`: fully static binary, no C dependencies
- `-ldflags="-w -s"`: strips debug info and symbol tables (~30% size reduction)
- distroless: no shell, minimal attack surface, passes security scans
- USER nonroot: don't run as root

## GitHub Actions CI/CD

```yaml
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.26'
          cache: true
      - name: Lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v2.9.0
      - name: Generate mocks
        run: go generate ./...
      - name: Test
        run: go test -v -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v5
        with:
          files: ./coverage.out
```

**Best practices:**
- Run lint before tests (fail fast)
- Use `-race` to detect race conditions
- Generate code (mocks) before testing
- Use actions/setup-go with `cache: true` for module caching
- Upload coverage to track trends
- Test on every push and PR

## Confidence Levels

| Component | Confidence | Source |
|-----------|-----------|--------|
| Go 1.26 | HIGH | Official Go release page (go.dev/doc/devel/release) |
| Dapr SDK v1.13.0 | HIGH | Official GitHub releases (github.com/dapr/go-sdk/releases) |
| modernc.org/sqlite | MEDIUM | Community consensus, multiple production users, but less official docs than mattn |
| Chi router | HIGH | Active maintenance, large ecosystem, widely adopted in 2025 |
| slog for logging | HIGH | Stdlib since Go 1.21, official recommendation for new projects |
| validator/v10 | HIGH | De facto standard for Go validation, actively maintained |
| Distroless images | HIGH | Google-maintained, industry best practice for Go containers |
| golangci-lint v2 | HIGH | Official releases, latest v2.9.0 supports Go 1.26 |

## Sources

### Official Documentation
- [Go 1.26 Release Notes](https://go.dev/doc/go1.26) — Latest Go version
- [Go Release History](https://go.dev/doc/devel/release) — Version verification
- [Dapr Go SDK](https://docs.dapr.io/developing-applications/sdks/go/) — Official SDK docs
- [Dapr Go SDK Releases](https://github.com/dapr/go-sdk/releases) — Version v1.13.0
- [Go slog Package](https://pkg.go.dev/log/slog) — Stdlib structured logging

### Framework and Library Documentation
- [Chi Router](https://go-chi.io/) — Lightweight HTTP router
- [Chi Middleware](https://pkg.go.dev/github.com/go-chi/chi/middleware) — Built-in middleware
- [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) — Pure Go SQLite driver
- [validator/v10](https://pkg.go.dev/github.com/go-playground/validator/v10) — Struct validation
- [google/uuid](https://pkg.go.dev/github.com/google/uuid) — UUID generation
- [testify](https://github.com/stretchr/testify) — Testing toolkit
- [mockery](https://vektra.github.io/mockery/latest/) — Mock generator

### Best Practices and Guides
- [The 8 best Go web frameworks for 2025](https://blog.logrocket.com/top-go-frameworks-2025/) — Framework comparison
- [The Go Ecosystem in 2025](https://blog.jetbrains.com/go/2025/11/10/go-language-trends-ecosystem-2025/) — Ecosystem trends
- [Which Go router should I use?](https://www.alexedwards.net/blog/which-go-router-should-i-use) — Router comparison
- [Alpine, distroless or scratch?](https://medium.com/google-cloud/alpine-distroless-or-scratch-caac35250e0b) — Container base images
- [GitHub Actions CI/CD Best Practices](https://github.com/github/awesome-copilot/blob/main/instructions/github-actions-ci-cd-best-practices.instructions.md)
- [Go Graceful Shutdown Patterns](https://victoriametrics.com/blog/go-graceful-shutdown/) — Production patterns
- [High-Performance Logging with slog](https://leapcell.io/blog/high-performance-structured-logging-in-go-with-slog-and-zerolog)

### Community Resources
- [Go Database Patterns: GORM, sqlx, and pgx Compared](https://dasroot.net/posts/2025/12/go-database-patterns-gorm-sqlx-pgx-compared/)
- [Comparing Go ORMs (2025)](https://encore.cloud/resources/go-orms)
- [Testing with Testify and Mockery](https://outcomeschool.com/blog/test-with-testify-and-mockery-in-go)
- [Go Linting Best Practices](https://medium.com/@tedious/go-linting-best-practices-for-ci-cd-with-github-actions-aa6d96e0c509)
- [golangci-lint v2 Changelog](https://golangci-lint.run/docs/product/changelog/)

---
*Stack research for: Go URL Shortener with Dapr Microservices*
*Researched: 2026-02-14*
*Confidence: HIGH — Verified with official docs and current releases*
