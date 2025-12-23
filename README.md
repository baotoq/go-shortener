# Go URL Shortener

A URL shortening service built with Go and the [Kratos](https://go-kratos.dev/) microservice framework.

## Features

- Create short URLs with auto-generated or custom codes
- Click analytics tracking
- URL expiration support
- Redis caching (optional, degrades gracefully)
- SQLite database storage
- Both HTTP and gRPC APIs

## Tech Stack

- **Framework**: [Kratos](https://go-kratos.dev/) v2.9.2
- **ORM**: [Ent](https://entgo.io/)
- **Database**: SQLite3
- **Cache**: Redis (optional)
- **DI**: Google Wire
- **API**: Protocol Buffers (gRPC + HTTP)

## Quick Start

### Prerequisites

- Go 1.22+
- Make
- Protocol Buffers compiler (`protoc`)

### Installation

```bash
# Install dependencies and tools
make init

# Generate code
make all

# Run the server
go run ./cmd/go-shortener -conf ./configs/config.yaml
```

### API Usage

```bash
# Create a short URL
curl -X POST http://localhost:8000/v1/urls \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://example.com"}'

# Create with custom code
curl -X POST http://localhost:8000/v1/urls \
  -H "Content-Type: application/json" \
  -d '{"original_url": "https://github.com", "custom_code": "gh"}'

# Redirect (returns 302)
curl -I http://localhost:8000/r/gh

# Get URL info
curl http://localhost:8000/v1/urls/gh

# Get stats
curl http://localhost:8000/v1/urls/gh/stats

# List all URLs
curl http://localhost:8000/v1/urls

# Delete URL
curl -X DELETE http://localhost:8000/v1/urls/gh
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

## Project Structure

```
go-shortener/
├── api/shortener/v1/       # Proto definitions & generated code
├── cmd/go-shortener/       # Application entry point
├── configs/                # Configuration files
├── ent/                    # Ent ORM schema
├── internal/
│   ├── biz/                # Business logic layer
│   ├── data/               # Data access layer
│   ├── service/            # Service handlers
│   ├── server/             # HTTP/gRPC servers
│   └── conf/               # Configuration
└── third_party/            # Third-party protos
```

## Development

```bash
# Generate API code from proto files
make api

# Generate error codes
make errors

# Run wire and format code
make generate

# Build binary
make build

# Run tests
go test ./... -v

# Run tests with coverage
go test ./... -cover
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

## Docker

```bash
# Build
docker build -t go-shortener .

# Run
docker run --rm -p 8000:8000 -p 9000:9000 -v $(pwd)/configs:/data/conf go-shortener
```

## License

MIT
