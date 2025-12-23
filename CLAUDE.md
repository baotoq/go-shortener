# Go URL Shortener

A URL shortening service built with Go and the Kratos microservice framework.

## Project Overview

This project implements a full-featured URL shortener with:
- Custom short codes or auto-generated codes
- Click analytics tracking
- URL expiration support
- Redis caching (optional, degrades gracefully)
- SQLite database storage
- Both HTTP and gRPC APIs

## Tech Stack

- **Framework**: [Kratos](https://go-kratos.dev/) v2.9.2
- **ORM**: [Ent](https://entgo.io/) (Facebook's entity framework)
- **Database**: SQLite3
- **Cache**: Redis (optional)
- **DI**: Google Wire
- **API**: Protocol Buffers (gRPC + HTTP)

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
│   ├── biz/               # Business logic layer
│   │   ├── url.go         # URL usecase
│   │   └── url_test.go    # Unit tests
│   ├── data/              # Data access layer
│   │   ├── data.go        # DB/Redis initialization
│   │   ├── url.go         # URL repository
│   │   └── url_test.go    # Unit tests
│   ├── service/           # Service layer (handlers)
│   │   ├── shortener.go   # gRPC/HTTP handlers
│   │   └── shortener_test.go
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
# Install dependencies and tools
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
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run specific layer tests
go test ./internal/biz/... -v
go test ./internal/data/... -v
go test ./internal/service/... -v

# Run with coverage
go test ./... -cover
```

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

The project follows Kratos clean architecture:

1. **Service Layer** (`internal/service/`) - Handles HTTP/gRPC requests, converts proto messages
2. **Business Layer** (`internal/biz/`) - Contains business logic, validation, domain models
3. **Data Layer** (`internal/data/`) - Repository implementation, database operations, caching

Dependencies flow inward: Service -> Biz -> Data

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
- Redis is optional - the app works without it (caching disabled)
- SQLite database file is created at `shortener.db` in the working directory
- Short codes are 6 characters by default (base64 URL-safe)
- Custom codes must be 3-20 characters (alphanumeric, underscore, hyphen)
