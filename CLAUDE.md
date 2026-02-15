# Go Shortener

URL shortener microservices project using Go and go-zero framework.

## Commands

```bash
goctl api go -api services/url-api/url.api -dir services/url-api -style gozero   # Regenerate URL service
goctl rpc protoc services/analytics-rpc/analytics.proto --go_out=services/analytics-rpc --go-grpc_out=services/analytics-rpc --zrpc_out=services/analytics-rpc --style gozero  # Regenerate Analytics service
```

## Architecture

Two microservices using go-zero framework:

- **url-api** (services/url-api): URL shortening API. go-zero REST service with .api spec code generation.
- **analytics-rpc** (services/analytics-rpc): Analytics zRPC service. Protobuf-generated gRPC server.

Each service follows go-zero convention: Handler → Logic → ServiceContext
Generated code from .api/.proto specs. Business logic in logic/ only.

```
services/{service}/
├── {service}.api or .proto  # Spec file (source of truth)
├── {service}.go             # Main entry point
├── etc/
│   └── {service}.yaml       # Configuration
└── internal/
    ├── config/              # Config struct
    ├── handler/             # HTTP handlers (generated)
    ├── logic/               # Business logic (manual)
    ├── svc/                 # ServiceContext (DI)
    └── types/               # Request/response types (generated)
```

## Key Patterns

- **goctl code generation**: .api/.proto specs generate handlers, types, routes. Never edit generated types.go.
- **Handler/Logic separation**: Handlers parse requests. Logic contains business rules. Never mix.
- **ServiceContext**: Dependency injection container. Initialized once, passed to all logic constructors.
- **RFC 7807 Problem Details**: Custom httpx.SetErrorHandler maps errors to Problem Details format.
- **Validation via tags**: .api type tags (range, options, optional, default) handle all request validation.
- **logx structured logging**: go-zero's built-in logger with context propagation.

## Gotchas

- Short codes: NanoID 8-char with collision retry (max 5 attempts)
- Rate limiter is in-memory per-instance, resets on restart
- GeoIP database (`data/GeoLite2-Country.mmdb`) is optional — falls back to "Unknown"
- Indentation: project uses 2 spaces for Go (see .editorconfig), not standard tabs

## Environment Variables

All have defaults suitable for local dev (no .env needed):
- `PORT` (8080/8081), `DATABASE_URL`, `BASE_URL`, `RATE_LIMIT`, `GEOIP_DB_PATH`, `KAFKA_BROKERS`

## Project Status

See `.planning/ROADMAP.md` for full plan. Phase 7 (Framework Foundation) in progress.
`.planning/STATE.md` tracks real-time progress and decisions.
