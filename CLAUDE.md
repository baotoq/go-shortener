# Go URL Shortener

A URL shortening service built with Go and the Kratos microservice framework.

## Project Overview

This project implements a full-featured URL shortener with:
- Custom short codes or auto-generated codes
- Click analytics tracking (event-driven)
- URL expiration support
- Redis caching (optional, degrades gracefully)
- SQLite database storage
- Both HTTP and gRPC APIs
- Domain-Driven Design (DDD) architecture

## Tech Stack

- **Framework**: [Kratos](https://go-kratos.dev/) v2.9.2
- **ORM**: [Ent](https://entgo.io/) (Facebook's entity framework)
- **Database**: SQLite3 (dev), PostgreSQL (production/tests)
- **Cache**: Redis (optional)
- **DI**: Google Wire
- **API**: Protocol Buffers (gRPC + HTTP)
- **Validation**: [ozzo-validation](https://github.com/go-ozzo/ozzo-validation)
- **Testing**: Testify (suite pattern), Mockery, Testcontainers

## Project Structure

```
go-shortener/
├── api/shortener/v1/       # Proto definitions & generated code
│   ├── shortener.proto     # API definitions
│   ├── error_reason.proto  # Error codes
│   └── *.pb.go            # Generated Go code
├── cmd/go-shortener/       # Application entry point
│   ├── main.go
│   ├── wire.go            # DI configuration
│   └── wire_gen.go        # Generated wire code
├── configs/
│   └── config.yaml        # Runtime configuration
├── ent/                   # Ent ORM
│   └── schema/url.go      # URL entity schema
├── internal/
│   ├── biz/               # Business logic layer (use cases)
│   │   ├── url.go         # URL usecase
│   │   ├── url_test.go    # Unit tests (testify suite + mockery)
│   │   └── event_handlers.go # Domain event handlers
│   ├── data/              # Data access layer
│   │   ├── data.go        # DB/Redis initialization
│   │   ├── url_repo.go    # URL repository implementation
│   │   └── url_repo_test.go # Integration tests (testcontainers)
│   ├── domain/            # Domain layer (DDD)
│   │   ├── url.go         # URL aggregate root
│   │   ├── aggregate.go   # AggregateRoot interface
│   │   ├── repository.go  # Repository interface
│   │   ├── uow.go         # Unit of Work interface
│   │   ├── valueobject.go # Re-exports value objects
│   │   ├── valueobject/   # Value objects
│   │   │   ├── shortcode.go     # ShortCode value object
│   │   │   ├── originalurl.go   # OriginalURL value object
│   │   │   └── errors.go        # Domain errors
│   │   └── event/         # Domain events
│   │       ├── event.go        # Event interface & dispatcher
│   │       ├── url_created.go  # URLCreated event
│   │       ├── url_clicked.go  # URLClicked event
│   │       ├── url_deleted.go  # URLDeleted event
│   │       └── url_expired.go  # URLExpired event
│   ├── mocks/             # Generated mocks (mockery)
│   ├── service/           # Service layer (handlers)
│   │   ├── shortener.go   # gRPC/HTTP handlers
│   │   └── shortener_test.go   # Service tests (testify suite)
│   ├── server/            # Server configuration
│   │   ├── http.go        # HTTP server
│   │   └── grpc.go        # gRPC server
│   └── conf/              # Configuration
│       └── conf.proto     # Config schema
└── third_party/           # Third-party protos
```

## Commands

### Development

```bash
# Install dependencies and tools (includes golangci-lint, lefthook)
make init

# Generate API code from proto files
make api

# Generate error codes
make errors

# Generate config proto
make config

# Run wire and format code
make generate

# Build binary
make build

# Run all generators
make all
```

### Running

```bash
# Start the server
go run ./cmd/go-shortener -conf ./configs/config.yaml

# Or run the built binary
./bin/go-shortener -conf ./configs/config.yaml
```

### Testing

```bash
# Run all tests (requires Docker for testcontainers)
go test ./...

# Run tests with verbose output
go test ./... -v

# Run specific layer tests
go test ./internal/biz/... -v
go test ./internal/data/... -v
go test ./internal/service/... -v
go test ./internal/domain/... -v

# Run with coverage
go test ./... -cover

# Generate mocks (on-demand, uses installed mockery binary)
go generate ./internal/domain/...
```

### Test Patterns

- **Testify Suite**: All tests use `suite.Suite` with `SetupTest()` for automatic setup
- **SUT naming**: System Under Test is named `sut` in test suites
- **AAA pattern**: All tests follow Arrange-Act-Assert pattern
- **Unit tests**: Use mockery-generated mocks
- **Integration tests**: Use testcontainers (PostgreSQL, Redis)

## API Endpoints

### HTTP API (port 8000)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/v1/urls` | Create a short URL |
| GET | `/v1/urls/{short_code}` | Get URL info |
| GET | `/r/{short_code}` | Redirect to original URL |
| GET | `/v1/urls/{short_code}/stats` | Get URL statistics |
| DELETE | `/v1/urls/{short_code}` | Delete a URL |
| GET | `/v1/urls` | List all URLs |

### gRPC API (port 9000)

Service: `shortener.v1.Shortener`
- `CreateURL`
- `GetURL`
- `RedirectURL`
- `GetURLStats`
- `DeleteURL`
- `ListURLs`

## Example Usage

```bash
# Create a short URL
curl -X POST http://localhost:8000/v1/urls \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com"}'

# Create with custom code
curl -X POST http://localhost:8000/v1/urls \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://github.com", "custom_code": "gh"}'

# Get URL info
curl http://localhost:8000/v1/urls/gh

# Redirect (returns 302)
curl -I http://localhost:8000/r/gh

# Get stats
curl http://localhost:8000/v1/urls/gh/stats

# List all URLs
curl http://localhost:8000/v1/urls

# Delete URL
curl -X DELETE http://localhost:8000/v1/urls/gh
```

## Configuration

Edit `configs/config.yaml`:

```yaml
server:
  http:
    addr: 0.0.0.0:8000
    timeout: 1s
  grpc:
    addr: 0.0.0.0:9000
    timeout: 1s
data:
  database:
    driver: sqlite3
    source: file:shortener.db?cache=shared&_fk=1
  redis:
    addr: 127.0.0.1:6379
    read_timeout: 0.2s
    write_timeout: 0.2s
```

## Architecture

The project follows DDD with Kratos clean architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                    Service Layer                             │
│  (HTTP/gRPC handlers, proto conversion)                     │
├─────────────────────────────────────────────────────────────┤
│                    Business Layer (Use Cases)                │
│  (Application logic, orchestration, event dispatch)         │
├─────────────────────────────────────────────────────────────┤
│                    Domain Layer                              │
│  (Aggregates, Entities, Value Objects, Domain Events)       │
├─────────────────────────────────────────────────────────────┤
│                    Data Layer                                │
│  (Repository implementations, database, caching)            │
└─────────────────────────────────────────────────────────────┘
```

Dependencies flow inward: Service -> Biz -> Domain <- Data

### Key DDD Concepts

- **Aggregate Root**: `URL` is the aggregate root with domain logic
- **Value Objects**: `ShortCode`, `OriginalURL` - immutable, validated on creation
- **Domain Events**: `URLCreated`, `URLClicked`, `URLDeleted`, `URLExpired`
- **Unit of Work**: Transaction management with event dispatch after commit
- **Repository Pattern**: Domain defines interface, data layer implements

### Event-Driven Click Tracking

Click counts are incremented via domain events:
1. `URL.RecordClick()` raises `URLClicked` event
2. `UnitOfWork.Do()` dispatches events after successful transaction
3. `ClickEventHandler` handles event and increments count in repository

## Error Handling

Custom error codes defined in `api/shortener/v1/error_reason.proto`:
- `URL_NOT_FOUND` (404)
- `URL_EXPIRED` (410)
- `INVALID_URL` (400)
- `SHORT_CODE_EXISTS` (409)
- `INVALID_SHORT_CODE` (400)

## Notes for Development

- Run `make generate` after modifying proto files
- Run `go generate ./ent` after modifying Ent schema
- Run `go generate ./internal/domain/...` to regenerate mocks
- Redis is optional - the app works without it (caching disabled)
- SQLite database file is created at `shortener.db` in the working directory
- Short codes are 6 characters by default (base64 URL-safe)
- Custom codes must be 3-20 characters (alphanumeric, underscore, hyphen)
- Integration tests require Docker to be running (testcontainers)
- Value objects use ozzo-validation for declarative validation
- All type conversions in Go must be explicit (e.g., `.String()` required)
