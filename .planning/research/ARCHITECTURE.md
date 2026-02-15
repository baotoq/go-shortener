# Architecture Research: go-zero Microservices Migration

**Domain:** URL Shortener Microservices
**Researched:** 2026-02-15
**Confidence:** HIGH

## Standard go-zero Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      Client Layer (HTTP/gRPC)                    │
├─────────────────────────────────────────────────────────────────┤
│  API Service (HTTP)              RPC Service (gRPC)              │
│  ┌──────────────────────┐       ┌──────────────────────┐        │
│  │ handler/             │       │ server/              │        │
│  │  - routes.go         │       │  - grpc_server.go    │        │
│  │  - *handler.go       │◄──────┤  - register service  │        │
│  └──────┬───────────────┘       └──────┬───────────────┘        │
│         │                               │                        │
├─────────┼───────────────────────────────┼────────────────────────┤
│  ┌──────▼───────────────┐       ┌──────▼───────────────┐        │
│  │ logic/               │       │ logic/               │        │
│  │  - *logic.go         │       │  - *logic.go         │        │
│  │  (business logic)    │       │  (business logic)    │        │
│  └──────┬───────────────┘       └──────┬───────────────┘        │
│         │                               │                        │
│         │  ServiceContext (svc/)        │                        │
│         │  - DB clients                 │                        │
│         │  - RPC clients                │                        │
│         │  - Queue producers/consumers  │                        │
│         │                               │                        │
├─────────┼───────────────────────────────┼────────────────────────┤
│  ┌──────▼───────────────┐       ┌──────▼───────────────┐        │
│  │ model/               │       │ model/               │        │
│  │  - *model.go         │       │  - *model.go         │        │
│  │  (sqlx queries)      │       │  (sqlx queries)      │        │
│  └──────────────────────┘       └──────────────────────┘        │
├─────────────────────────────────────────────────────────────────┤
│                     Infrastructure Layer                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │PostgreSQL│  │   Kafka  │  │   etcd   │  │  Redis   │        │
│  │          │  │(go-queue)│  │(optional)│  │(optional)│        │
│  └──────────┘  └──────────┘  └──────────┘  └──────────┘        │
└─────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Handler | HTTP request/response handling, validation, routing | Generated from .api spec via goctl |
| Logic | Business logic, orchestration, validation | Manual implementation in *logic.go files |
| ServiceContext (svc) | Dependency injection container | Holds DB, RPC clients, config, queues |
| Model | Database access layer (CRUD) | Generated from PostgreSQL via goctl model |
| Types | Request/Response DTOs | Generated from .api spec type blocks |
| RPC Server | gRPC service implementation | Generated from .proto via goctl rpc |
| Queue (go-queue) | Kafka producer/consumer | Configured via KqConf, implements Consume() |

## Recommended Project Structure for go-zero Monorepo

```
go-shortener/
├── go.work                     # Go workspace for multi-module monorepo
├── docker-compose.yml          # PostgreSQL, Kafka, Zookeeper, services
├── shared/                     # Shared module
│   ├── go.mod
│   └── events/                 # Event type definitions
│       ├── click_event.go
│       └── link_deleted_event.go
├── services/
│   ├── url/                    # URL Service (API)
│   │   ├── go.mod
│   │   ├── api/
│   │   │   ├── url.api         # API spec (routes, types)
│   │   │   ├── etc/
│   │   │   │   └── url.yaml    # Service config
│   │   │   ├── internal/
│   │   │   │   ├── config/
│   │   │   │   │   └── config.go
│   │   │   │   ├── handler/    # Generated + manual routes
│   │   │   │   │   ├── routes.go
│   │   │   │   │   ├── shortenhandler.go
│   │   │   │   │   ├── redirecthandler.go
│   │   │   │   │   └── linkshandler.go
│   │   │   │   ├── logic/      # Business logic
│   │   │   │   │   ├── shortenlogic.go
│   │   │   │   │   ├── redirectlogic.go
│   │   │   │   │   └── linkslogic.go
│   │   │   │   ├── svc/        # Dependency injection
│   │   │   │   │   └── servicecontext.go
│   │   │   │   ├── types/      # Generated request/response
│   │   │   │   │   └── types.go
│   │   │   │   └── model/      # Generated DB models
│   │   │   │       ├── urlmodel.go
│   │   │   │       └── vars.go
│   │   │   └── url.go          # Main entry point
│   │   └── Dockerfile
│   └── analytics/              # Analytics Service (RPC + Kafka consumer)
│       ├── go.mod
│       ├── rpc/
│       │   ├── analytics.proto # Protobuf service definition
│       │   ├── etc/
│       │   │   └── analytics.yaml
│       │   ├── internal/
│       │   │   ├── config/
│       │   │   │   └── config.go
│       │   │   ├── server/     # gRPC server
│       │   │   │   └── analyticsserver.go
│       │   │   ├── logic/      # RPC business logic
│       │   │   │   ├── getclickcountlogic.go
│       │   │   │   └── getanalyticssummarylogic.go
│       │   │   ├── svc/        # Dependency injection
│       │   │   │   └── servicecontext.go
│       │   │   └── model/      # Generated DB models
│       │   │       ├── clickmodel.go
│       │   │       └── vars.go
│       │   ├── pb/             # Generated protobuf
│       │   │   └── analytics.pb.go
│       │   └── analytics.go    # Main entry point
│       ├── consumer/           # Kafka consumer
│       │   ├── etc/
│       │   │   └── consumer.yaml
│       │   ├── internal/
│       │   │   ├── config/
│       │   │   │   └── config.go
│       │   │   ├── consumer/   # Kafka message handler
│       │   │   │   └── clickconsumer.go
│       │   │   └── svc/
│       │   │       └── servicecontext.go
│       │   └── consumer.go     # Main entry point
│       └── Dockerfile
└── README.md
```

### Structure Rationale

- **Go workspace (go.work):** Enables multi-module monorepo with local replace directives for shared code without publishing
- **shared/events/:** Event schemas shared between services (ClickEvent, LinkDeletedEvent)
- **services/url/api/:** API service generated from url.api spec, HTTP REST endpoints
- **services/analytics/rpc/:** RPC service generated from analytics.proto, gRPC endpoints
- **services/analytics/consumer/:** Separate binary for Kafka consumer (can run independently)
- **internal/ boundary:** Each service's internal code is isolated, cannot be imported by other services
- **model/ per service:** Each service owns its database tables, no shared database access

## Architectural Patterns

### Pattern 1: Handler → Logic → Model (3-Layer)

**What:** go-zero's standard layering separates HTTP concerns, business logic, and data access.

**When to use:** All API and RPC services. This is go-zero's core pattern.

**Trade-offs:**
- **Pros:** Clear separation of concerns, testable layers, generated boilerplate
- **Cons:** More files than simple handler-only pattern, logic layer can become fat

**Example:**
```go
// handler/shortenhandler.go (generated, minimal edits)
func ShortenHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.ShortenRequest
        if err := httpx.Parse(r, &req); err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
            return
        }

        l := logic.NewShortenLogic(r.Context(), svcCtx)
        resp, err := l.Shorten(&req)
        if err != nil {
            httpx.ErrorCtx(r.Context(), w, err)
        } else {
            httpx.OkJsonCtx(r.Context(), w, resp)
        }
    }
}

// logic/shortenlogic.go (manual business logic)
type ShortenLogic struct {
    logx.Logger
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func (l *ShortenLogic) Shorten(req *types.ShortenRequest) (*types.ShortenResponse, error) {
    // Validate URL
    if err := validateURL(req.OriginalURL); err != nil {
        return nil, err
    }

    // Check for duplicate
    existing, err := l.svcCtx.URLModel.FindByOriginalURL(l.ctx, req.OriginalURL)
    if err == nil {
        return &types.ShortenResponse{ShortCode: existing.ShortCode}, nil
    }

    // Generate short code
    shortCode := generateShortCode()
    url, err := l.svcCtx.URLModel.Insert(l.ctx, &model.URL{
        ShortCode:   shortCode,
        OriginalURL: req.OriginalURL,
    })
    if err != nil {
        return nil, err
    }

    return &types.ShortenResponse{ShortCode: url.ShortCode}, nil
}

// model/urlmodel.go (generated by goctl)
type URLModel interface {
    Insert(ctx context.Context, data *URL) (*URL, error)
    FindByShortCode(ctx context.Context, shortCode string) (*URL, error)
    FindByOriginalURL(ctx context.Context, originalURL string) (*URL, error)
}
```

### Pattern 2: ServiceContext Dependency Injection

**What:** ServiceContext is a struct holding all dependencies (DB, RPC clients, queues, config). Initialized once in main.go, passed to all handlers/logic.

**When to use:** Always. This is go-zero's dependency injection pattern.

**Trade-offs:**
- **Pros:** Single source of truth for dependencies, easy to mock for testing
- **Cons:** Can become bloated with too many dependencies

**Example:**
```go
// svc/servicecontext.go
type ServiceContext struct {
    Config         config.Config
    URLModel       model.URLModel
    AnalyticsRpc   analyticsclient.Analytics  // RPC client
    KafkaProducer  *kq.Pusher                  // Kafka producer
}

func NewServiceContext(c config.Config) *ServiceContext {
    conn := sqlx.NewMysql(c.DataSource)

    return &ServiceContext{
        Config:         c,
        URLModel:       model.NewURLModel(conn),
        AnalyticsRpc:   analyticsclient.NewAnalytics(zrpc.MustNewClient(c.AnalyticsRpc)),
        KafkaProducer:  kq.NewPusher(c.Kafka.Brokers, c.Kafka.Topic),
    }
}
```

### Pattern 3: API Service Calls RPC Service (zRPC Client)

**What:** API service injects RPC client via ServiceContext, logic layer makes gRPC calls.

**When to use:** When API service needs data/operations from RPC service (e.g., URL service fetching click counts from Analytics RPC).

**Trade-offs:**
- **Pros:** Type-safe gRPC with protobuf, built-in load balancing and circuit breaking
- **Cons:** Requires protobuf definitions, adds network hop

**Example:**
```go
// URL Service calls Analytics RPC to get click count

// svc/servicecontext.go (URL Service)
type ServiceContext struct {
    Config         config.Config
    URLModel       model.URLModel
    AnalyticsRpc   analyticsclient.Analytics  // RPC client injected
}

// logic/linkslogic.go (URL Service)
func (l *LinksLogic) GetLinkWithClicks(req *types.LinkDetailRequest) (*types.LinkDetailResponse, error) {
    // Fetch URL from database
    url, err := l.svcCtx.URLModel.FindByShortCode(l.ctx, req.ShortCode)
    if err != nil {
        return nil, err
    }

    // Call Analytics RPC to get click count
    clickResp, err := l.svcCtx.AnalyticsRpc.GetClickCount(l.ctx, &analytics.ClickCountRequest{
        ShortCode: req.ShortCode,
    })
    if err != nil {
        return nil, err
    }

    return &types.LinkDetailResponse{
        ShortCode:   url.ShortCode,
        OriginalURL: url.OriginalURL,
        TotalClicks: clickResp.Count,
    }, nil
}
```

### Pattern 4: Kafka Producer (Fire-and-Forget Events)

**What:** URL Service publishes ClickEvent to Kafka after redirect. Uses go-queue's kq.Pusher in goroutine for non-blocking.

**When to use:** Asynchronous events where user shouldn't wait (analytics, notifications).

**Trade-offs:**
- **Pros:** User not blocked, Kafka ensures delivery, decouples services
- **Cons:** Eventual consistency, requires Kafka infrastructure

**Example:**
```go
// logic/redirectlogic.go (URL Service)
func (l *RedirectLogic) Redirect(req *types.RedirectRequest) (*types.RedirectResponse, error) {
    // Find URL
    url, err := l.svcCtx.URLModel.FindByShortCode(l.ctx, req.ShortCode)
    if err != nil {
        return nil, err
    }

    // Publish click event (fire-and-forget in goroutine)
    go func() {
        event := events.ClickEvent{
            ShortCode: req.ShortCode,
            Timestamp: time.Now(),
            ClientIP:  req.ClientIP,
            UserAgent: req.UserAgent,
            Referer:   req.Referer,
        }
        data, _ := json.Marshal(event)
        l.svcCtx.KafkaProducer.Push(string(data))
    }()

    return &types.RedirectResponse{RedirectURL: url.OriginalURL}, nil
}
```

### Pattern 5: Kafka Consumer (go-queue Consumer)

**What:** Analytics Service runs separate consumer binary that implements Consume(key, val string) error, processes ClickEvents from Kafka.

**When to use:** Processing async events from Kafka.

**Trade-offs:**
- **Pros:** Independent scaling, retry logic, consumer groups
- **Cons:** Separate binary to deploy, complexity vs HTTP webhook

**Example:**
```go
// consumer/internal/consumer/clickconsumer.go
type ClickConsumer struct {
    svcCtx *svc.ServiceContext
}

func NewClickConsumer(svcCtx *svc.ServiceContext) *ClickConsumer {
    return &ClickConsumer{svcCtx: svcCtx}
}

func (c *ClickConsumer) Consume(key, val string) error {
    var event events.ClickEvent
    if err := json.Unmarshal([]byte(val), &event); err != nil {
        return err
    }

    // Enrich and store
    countryCode := c.svcCtx.GeoIP.ResolveCountry(event.ClientIP)
    deviceType := c.svcCtx.DeviceDetector.DetectDevice(event.UserAgent)
    trafficSource := c.svcCtx.RefererClassifier.ClassifySource(event.Referer)

    return c.svcCtx.ClickModel.Insert(context.Background(), &model.Click{
        ShortCode:     event.ShortCode,
        Timestamp:     event.Timestamp.Unix(),
        CountryCode:   countryCode,
        DeviceType:    deviceType,
        TrafficSource: trafficSource,
    })
}

// consumer/consumer.go (main)
func main() {
    c := config.Config{}
    conf.MustLoad(*configFile, &c)

    svcCtx := svc.NewServiceContext(c)
    consumer := consumer.NewClickConsumer(svcCtx)

    queue := kq.MustNewQueue(c.Kafka, consumer)
    defer queue.Stop()

    fmt.Println("Analytics consumer started...")
    queue.Start()
}
```

## Data Flow

### Request Flow (API Service)

```
HTTP Request (POST /api/v1/urls)
    ↓
Handler: ShortenHandler
    - Parse request → types.ShortenRequest
    - Call Logic layer
    ↓
Logic: ShortenLogic.Shorten()
    - Validate URL
    - Check duplicate (Model.FindByOriginalURL)
    - Generate short code
    - Insert (Model.Insert)
    - Return types.ShortenResponse
    ↓
Handler: httpx.OkJsonCtx()
    - Serialize response
    ↓
HTTP Response (200 OK, JSON)
```

### Event Flow (Kafka Pub/Sub)

```
URL Service: Redirect Handler
    - Find URL (Model.FindByShortCode)
    - Return 302 redirect
    - (goroutine) Publish ClickEvent → Kafka
    ↓
Kafka Topic: "clicks"
    - Persist event
    - Consumer group reads
    ↓
Analytics Service: Consumer
    - Consume(key, val)
    - Unmarshal ClickEvent
    - Enrich (GeoIP, Device, Referer)
    - Insert Click (Model.Insert)
```

### RPC Flow (API → RPC Service)

```
URL Service: GetLinkWithClicks Handler
    ↓
Logic: LinksLogic.GetLinkWithClicks()
    - Fetch URL (URLModel.FindByShortCode)
    - Call AnalyticsRpc.GetClickCount() → gRPC
    ↓
Analytics RPC Service: GetClickCountLogic
    - ClickModel.CountByShortCode()
    - Return ClickCountResponse
    ↓
URL Service: Combine data
    - Return LinkDetailResponse (URL + click count)
```

### Key Data Flows

1. **Shorten URL:** HTTP → Handler → Logic (validate, dedupe, insert) → Model → DB → Response
2. **Redirect:** HTTP → Handler → Logic (find URL) → Model → DB → 302 Redirect + Kafka publish (async)
3. **Analytics Consumer:** Kafka → Consumer.Consume() → Enrich → Model.Insert() → DB
4. **Link Detail (with clicks):** HTTP → Handler → Logic (fetch URL + RPC call) → Analytics RPC → DB → Response

## Migration from Current Clean Architecture to go-zero

### Current Architecture (v1.0)

```
internal/
├── urlservice/
│   ├── delivery/http/      (Chi handlers, middleware, router)
│   ├── usecase/            (URLService, business logic)
│   ├── repository/sqlite/  (sqlc-generated, URLRepository)
│   ├── domain/             (URL entity, errors)
│   └── database/           (SQLite connection, migrations)
└── analytics/
    ├── delivery/http/      (Chi handlers)
    ├── usecase/            (AnalyticsService, GeoIP, Device)
    ├── repository/sqlite/  (sqlc-generated, ClickRepository)
    ├── domain/             (Click entity)
    └── database/           (SQLite connection)
```

### go-zero Architecture (v2.0 Target)

```
services/
├── url/api/
│   ├── url.api             (.api spec defines routes + types)
│   ├── internal/
│   │   ├── handler/        (generated from .api)
│   │   ├── logic/          (manual, migrate from usecase/)
│   │   ├── svc/            (ServiceContext, replaces DI in handlers)
│   │   ├── types/          (generated from .api)
│   │   └── model/          (generated from PostgreSQL)
└── analytics/
    ├── rpc/
    │   ├── analytics.proto (protobuf service definition)
    │   ├── internal/
    │   │   ├── server/     (gRPC server)
    │   │   ├── logic/      (manual, migrate from usecase/)
    │   │   ├── svc/        (ServiceContext)
    │   │   └── model/      (generated from PostgreSQL)
    └── consumer/           (Kafka consumer, separate binary)
        └── internal/
            └── consumer/   (ClickConsumer.Consume())
```

### Mapping Table: Old → New

| Current Component | go-zero Component | Migration Action |
|-------------------|-------------------|------------------|
| delivery/http/handler.go | handler/*handler.go | REWRITE: Generate from .api, move business logic to logic/ |
| delivery/http/router.go | handler/routes.go | GENERATE: From .api service block |
| usecase/url_service.go | logic/*logic.go | REFACTOR: Extract business logic, inject via ServiceContext |
| usecase/url_repository.go (interface) | model/*model.go (interface) | REGENERATE: goctl model pg datasource |
| repository/sqlite/*.go | model/*model.go (impl) | REGENERATE: goctl model for PostgreSQL |
| domain/url.go | types/types.go + model/*model.go | SPLIT: DTOs in types/, entities in model/ |
| Dapr pub/sub | go-queue (Kafka) | REPLACE: kq.Pusher (producer) + kq.Queue (consumer) |
| Dapr service invocation | zRPC client | REPLACE: analyticsclient.Analytics (generated from .proto) |
| Chi router + middleware | rest.Server (built-in) | REPLACE: go-zero's rest server with built-in middleware |
| sqlc queries | goctl model sqlx | REPLACE: goctl generates CRUD from PostgreSQL schema |

### New Components in go-zero

| Component | Purpose | Notes |
|-----------|---------|-------|
| .api spec | Route + type definitions | DSL for HTTP services, replaces manual routing |
| .proto spec | RPC service definitions | Protobuf for gRPC services |
| ServiceContext | Dependency injection | Central place for DB, RPC clients, queues |
| go-queue (kq) | Kafka integration | Replaces Dapr pub/sub with native Kafka |
| zRPC client | gRPC client with LB | Replaces Dapr service invocation |
| goctl CLI | Code generation | Generates handlers, models, RPC stubs |

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| PostgreSQL | goctl model + sqlx | goctl generates models from DB schema, sqlx for queries |
| Kafka | go-queue (kq.Pusher, kq.Queue) | Producer: kq.Pusher.Push(), Consumer: kq.Queue with Consume() |
| GeoIP (MaxMind) | Custom service in ServiceContext | Manual integration, same as v1.0 |
| Analytics RPC | zRPC client | analyticsclient.Analytics generated from .proto |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| URL Service ↔ Analytics RPC | zRPC (gRPC) | For GetClickCount, GetAnalyticsSummary synchronous calls |
| URL Service → Analytics Consumer | Kafka (async) | For ClickEvent publishing after redirect |
| Handler ↔ Logic | Direct method call | Handler calls NewXxxLogic(ctx, svcCtx).Method() |
| Logic ↔ Model | Interface call | Logic calls svcCtx.URLModel.Insert(), FindByShortCode() |
| Shared events | Go module import | URL and Analytics both import shared/events via go.work |

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 0-10k users | Single instance per service, single Kafka partition, single PostgreSQL instance. go-zero's built-in rate limiting and circuit breaking sufficient. |
| 10k-100k users | Horizontal scaling of API service (stateless), increase Kafka partitions and consumer instances, add PostgreSQL connection pooling, enable Redis caching for hot URLs (add svcCtx.RedisClient). |
| 100k+ users | Separate read/write databases (PostgreSQL replication), Redis cluster for caching, Kafka cluster with 10+ partitions, multiple Analytics consumer instances, consider etcd service discovery for zRPC load balancing. |

### Scaling Priorities

1. **First bottleneck:** PostgreSQL connection exhaustion. Fix: Increase connection pool size in sqlx.NewMysql(), add Redis caching for FindByShortCode().
2. **Second bottleneck:** Single Kafka partition slows analytics processing. Fix: Increase partitions, run multiple consumer instances (go-queue handles consumer group).

## Anti-Patterns

### Anti-Pattern 1: Putting Business Logic in Handlers

**What people do:** Write validation, database calls directly in handler functions instead of logic layer.

**Why it's wrong:** Violates go-zero's layering, makes testing harder (can't test logic without HTTP), breaks generated code updates.

**Do this instead:** Handlers only parse/serialize. All business logic in logic/*.go. Model calls in logic, not handler.

### Anti-Pattern 2: Sharing Database Models Across Services

**What people do:** Create a shared/models/ package that both URL and Analytics services import.

**Why it's wrong:** Violates microservice autonomy, creates tight coupling, shared database access is an anti-pattern.

**Do this instead:** Each service owns its tables. Use RPC or events for cross-service data access. Analytics never queries url_service.urls table.

### Anti-Pattern 3: Synchronous Kafka Publish Blocking Request

**What people do:** Call KafkaProducer.Push() in main request path, wait for Kafka ack.

**Why it's wrong:** Adds latency, Kafka failures break redirects (core value).

**Do this instead:** Fire-and-forget in goroutine. User gets 302 redirect immediately. Analytics is eventually consistent.

### Anti-Pattern 4: Not Using ServiceContext for Dependencies

**What people do:** Create database connections inside logic constructors, use global variables for RPC clients.

**Why it's wrong:** Hard to test (can't inject mocks), violates go-zero's dependency injection pattern.

**Do this instead:** Initialize all dependencies in NewServiceContext(), pass ServiceContext to logic, inject mocks in tests.

### Anti-Pattern 5: Ignoring go-zero's Generated Code Structure

**What people do:** Delete generated handler/routes.go, rewrite with custom patterns.

**Why it's wrong:** Breaks goctl regeneration, loses middleware integration, makes upgrades painful.

**Do this instead:** Keep generated files, customize via .api spec or middleware. Edit logic/, not handler/.

## Docker Compose Topology

### Services and Dependencies

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_DB: shortener
      POSTGRES_USER: shortener
      POSTGRES_PASSWORD: shortener
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  zookeeper:
    image: confluentinc/cp-zookeeper:7.6.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
      ZOOKEEPER_TICK_TIME: 2000

  kafka:
    image: confluentinc/cp-kafka:7.6.0
    depends_on:
      - zookeeper
    environment:
      KAFKA_BROKER_ID: 1
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    ports:
      - "9092:9092"

  url-service:
    build:
      context: ./services/url
    depends_on:
      - postgres
      - kafka
    environment:
      - POSTGRES_DSN=postgres://shortener:shortener@postgres:5432/shortener?sslmode=disable
      - KAFKA_BROKERS=kafka:9092
      - KAFKA_TOPIC=clicks
    ports:
      - "8080:8080"

  analytics-rpc:
    build:
      context: ./services/analytics
      dockerfile: Dockerfile.rpc
    depends_on:
      - postgres
    environment:
      - POSTGRES_DSN=postgres://shortener:shortener@postgres:5432/shortener?sslmode=disable
    ports:
      - "8081:8081"  # gRPC port

  analytics-consumer:
    build:
      context: ./services/analytics
      dockerfile: Dockerfile.consumer
    depends_on:
      - postgres
      - kafka
      - analytics-rpc
    environment:
      - POSTGRES_DSN=postgres://shortener:shortener@postgres:5432/shortener?sslmode=disable
      - KAFKA_BROKERS=kafka:9092
      - KAFKA_TOPIC=clicks
      - KAFKA_GROUP=analytics-group
```

**No Dapr sidecars:** go-zero provides native Kafka and gRPC, Dapr layer removed.

**3 service binaries:** url-service (API), analytics-rpc (gRPC), analytics-consumer (Kafka consumer).

**Shared PostgreSQL:** Each service has separate schema/tables, no shared table access.

## Sources

### go-zero Framework Documentation
- [Project Structure | go-zero Documentation](https://go-zero.dev/en/docs/concepts/layout) - Official project layout recommendations
- [Framework Overview | go-zero Documentation](https://go-zero.dev/en/docs/concepts/overview) - Architecture and components
- [API Syntax | go-zero Documentation](https://go-zero.dev/en/docs/tasks/dsl/api) - .api file DSL specification
- [goctl RPC | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/cli/rpc) - RPC service generation from protobuf
- [goctl Model | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/cli/model) - Database model generation
- [Message Queue | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/message-queue/kafka) - Kafka integration with go-queue
- [gRPC Gateway | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/gateway/grpc) - API gateway calling RPC services
- [MySQL Database Operations | go-zero Documentation](https://go-zero.dev/en/docs/tasks/mysql) - Database integration patterns
- [Service Example | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/grpc/server/example) - zRPC server examples
- [Service Connection | go-zero Documentation](https://go-zero.dev/en/docs/tutorials/grpc/client/conn) - zRPC client integration

### Community Articles and Examples
- [Developing a RESTful API With Go | by Kevin Wan | Better Programming](https://betterprogramming.pub/developing-a-restful-api-with-go-b5150f693277) - go-zero API service patterns
- [Go zero microservice framework tutorial](https://architecture.pub/articles/196239252.html) - ServiceContext and dependency injection examples
- [Using gorm gen in go-zero | by seth-shi | Medium](https://seth-shi.medium.com/using-gorm-gen-in-go-zero-the-built-in-sqlx-in-go-zero-is-too-hard-to-use-881b6edf49ed) - Database model patterns

### Go Monorepo and Architecture Patterns
- [Go Monorepos for Growing Teams](https://jamescun.com/posts/golang-monorepo-structure/) - Monorepo organization with pkg/ and svc/
- [Building a Monorepo in Golang - Earthly Blog](https://earthly.dev/blog/golang-monorepo/) - Go workspace patterns
- [Shared Go Packages in a Monorepo | 1Password](https://passage.1password.com/post/shared-go-packages-in-a-monorepo) - Replace directives and shared modules
- [How to Use Go Workspaces for Monorepos | OneUptime](https://oneuptime.com/blog/post/2026-02-01-go-workspaces-monorepos/view) - Go 1.21+ workspaces for monorepos
- [Scalable and Maintainable Applications through Three-Layered Architecture in Go - GeeksforGeeks](https://www.geeksforgeeks.org/go-language/scalable-and-maintainable-applications-through-three-layered-architecture-in-go/) - Handler-Service-Repository pattern

### Kafka and Infrastructure
- [GitHub - zeromicro/go-queue](https://github.com/zeromicro/go-queue) - go-zero's Kafka pub/sub framework
- [GitHub - zeromicro/go-zero](https://github.com/zeromicro/go-zero) - Official go-zero repository with examples

---
*Architecture research for: URL Shortener Microservices Migration to go-zero*
*Researched: 2026-02-15*
*Confidence: HIGH - Based on official go-zero documentation, current v1.0 codebase analysis, and verified migration patterns*
