# Phase 9: Messaging Migration - Research

**Researched:** 2026-02-16
**Domain:** Kafka message queue integration, zRPC client configuration, event-driven architecture
**Confidence:** HIGH

## Summary

Phase 9 migrates from the planned Dapr pub/sub to Kafka using go-zero's native `go-queue` (kq) library and implements zRPC client communication between URL Service and Analytics Service. The go-zero ecosystem provides first-class Kafka support through `kq.Pusher` for fire-and-forget publishing and `kq.Queue` for consumer groups with built-in concurrent processing. zRPC clients integrate via ServiceContext with automatic circuit breaking and support both direct connection mode (no Etcd) and service discovery.

**Key findings:**
- go-queue kq provides production-ready Kafka integration with minimal configuration
- Fire-and-forget publishing is natural with `kq.Pusher.Push()` (non-blocking, returns error for logging)
- Consumer idempotency requires application-level deduplication (database tracking pattern)
- zRPC clients inject via ServiceContext with built-in circuit breakers and timeout handling
- `threading.GoSafe` is go-zero's panic-safe goroutine wrapper for fire-and-forget tasks
- Modern Kafka deployments use KRaft mode (no ZooKeeper) simplifying Docker Compose setup

**Primary recommendation:** Use kq.Pusher in redirect handler with threading.GoSafe, implement standalone consumer service with kq.Queue, add Analytics zRPC client to URL Service via ServiceContext, and handle idempotency with short_code-based deduplication in clicks table.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| github.com/zeromicro/go-queue | Latest | Kafka producer/consumer | Official go-zero Kafka integration, production-tested |
| github.com/segmentio/kafka-go | (transitive) | Kafka client | Used by go-queue, modern pure-Go Kafka client |
| github.com/zeromicro/go-zero/zrpc | v1.9.2+ | gRPC client/server | go-zero native RPC framework with built-in resilience |
| github.com/oschwald/geoip2-golang | v1.11.0+ | GeoIP country lookup | Official MaxMind reader, most popular Go library |
| github.com/mileusna/useragent | v1.3.5+ | User agent parsing | Simple, deterministic, no external dependencies |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/zeromicro/go-zero/core/threading | v1.9.2+ | Safe goroutine execution | Fire-and-forget with panic recovery |
| apache/kafka | 3.7.0+ | Kafka broker (Docker) | Development and production message queue |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| kq (go-queue) | Raw kafka-go | More control but lose go-zero integration, no built-in consumer group handling |
| mileusna/useragent | device-detector-go | More comprehensive device detection but 10k+ test overhead, overkill for basic browser/device/OS |
| KRaft mode | ZooKeeper mode | ZooKeeper adds operational complexity, KRaft is Kafka's future (ZK deprecated in 3.x) |

**Installation:**
```bash
go get github.com/zeromicro/go-queue@latest
go get github.com/oschwald/geoip2-golang@latest
go get github.com/mileusna/useragent@latest
```

## Architecture Patterns

### Recommended Project Structure

```
services/
├── url-api/
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go           # Add KqPusherConf, AnalyticsRpc fields
│   │   ├── svc/
│   │   │   └── servicecontext.go   # Add KqPusher, AnalyticsRpc clients
│   │   └── logic/redirect/
│   │       └── redirectlogic.go    # Publish ClickEvent after redirect
│   └── etc/
│       └── url.yaml                # Add Kafka broker and Analytics RPC config
│
├── analytics-rpc/
│   ├── internal/
│   │   └── logic/
│   │       └── getclickcountlogic.go  # Already implemented (Phase 8)
│   └── etc/
│       └── analytics.yaml          # No changes needed
│
└── analytics-consumer/             # NEW standalone consumer service
    ├── consumer.go                 # Main entry point
    ├── internal/
    │   ├── config/
    │   │   └── config.go           # KqConsumerConf, DataSource, GeoIPDbPath
    │   ├── svc/
    │   │   └── servicecontext.go   # ClicksModel, GeoIP reader
    │   └── handler/
    │       └── clickevent.go       # ClickEvent consumer handler
    └── etc/
        └── consumer.yaml           # Kafka consumer config
```

### Pattern 1: Fire-and-Forget Kafka Publishing

**What:** Publish events asynchronously without blocking the request handler
**When to use:** Redirect handler must return immediately regardless of event publishing success
**Example:**
```go
// Source: Official go-queue examples + go-zero threading patterns
// https://github.com/zeromicro/go-queue/blob/master/example/kq/consumer/queue.go
// https://github.com/zeromicro/go-zero (threading.GoSafe usage)

package redirect

import (
  "context"
  "encoding/json"
  "go-shortener/common/events"
  "github.com/zeromicro/go-zero/core/threading"
  "github.com/zeromicro/go-zero/core/logx"
)

func (l *RedirectLogic) Redirect(req *types.RedirectRequest) (string, error) {
  // 1. Look up URL (blocking, required for redirect)
  url, err := l.svcCtx.UrlModel.FindOneByShortCode(l.ctx, req.Code)
  if err != nil {
    return "", err
  }

  // 2. Publish ClickEvent (fire-and-forget with panic safety)
  threading.GoSafe(func() {
    event := events.ClickEvent{
      ShortCode: req.Code,
      Timestamp: time.Now().Unix(),
      IP:        httpx.GetRemoteAddr(l.ctx),
      UserAgent: r.UserAgent(),
      Referer:   r.Referer(),
    }

    data, _ := json.Marshal(event)
    if err := l.svcCtx.KqPusher.Push(string(data)); err != nil {
      logx.WithContext(l.ctx).Errorw("failed to publish click event",
        logx.Field("code", req.Code),
        logx.Field("error", err.Error()),
      )
    }
  })

  // 3. Return immediately (user never waits for event publishing)
  return url.OriginalUrl, nil
}
```

**Key characteristics:**
- `threading.GoSafe` wraps goroutine with panic recovery (go-zero utility)
- Errors logged but don't affect redirect response
- `context.Background()` not needed (threading.GoSafe manages lifecycle)
- Push() returns error for observability (log but don't block)

### Pattern 2: Kafka Consumer with Idempotency

**What:** Process events from Kafka with deduplication to handle at-least-once delivery
**When to use:** Consumer must be idempotent (Kafka doesn't guarantee exactly-once without transactions)
**Example:**
```go
// Source: Idempotent Kafka consumer pattern
// https://nejckorasa.github.io/posts/idempotent-kafka-procesing/
// + go-queue consumer setup
// https://go-zero.dev/en/docs/tutorials/message-queue/kafka

package handler

import (
  "context"
  "encoding/json"
  "go-shortener/common/events"
)

type ClickEventHandler struct {
  ctx    context.Context
  svcCtx *svc.ServiceContext
}

func NewClickEventHandler(ctx context.Context, svcCtx *svc.ServiceContext) *ClickEventHandler {
  return &ClickEventHandler{ctx: ctx, svcCtx: svcCtx}
}

// Consume implements kq handler interface
func (h *ClickEventHandler) Consume(key, val string) error {
  var event events.ClickEvent
  if err := json.Unmarshal([]byte(val), &event); err != nil {
    logx.Errorw("invalid click event", logx.Field("error", err.Error()))
    return nil // Don't retry malformed messages
  }

  // Idempotency: Use (short_code, timestamp) as natural key
  // If click already exists, INSERT will fail (unique constraint)
  // This is acceptable: duplicate events from Kafka retries are dropped

  // Enrich event
  country := h.enrichGeoIP(event.IP)
  deviceType, browser := h.enrichUserAgent(event.UserAgent)
  trafficSource := h.enrichReferer(event.Referer)

  // Insert click record (fails silently if duplicate)
  click := model.Click{
    ShortCode:     event.ShortCode,
    ClickedAt:     time.Unix(event.Timestamp, 0),
    CountryCode:   country,
    DeviceType:    deviceType,
    TrafficSource: trafficSource,
  }

  if err := h.svcCtx.ClicksModel.Insert(h.ctx, &click); err != nil {
    // Log error but return nil (Kafka won't retry)
    // Database errors are transient; monitoring will catch patterns
    logx.Errorw("failed to insert click",
      logx.Field("code", event.ShortCode),
      logx.Field("error", err.Error()),
    )
  }

  return nil
}
```

**Idempotency strategy:**
- Database-level deduplication via unique constraint on (short_code, clicked_at_unix)
- Duplicate events fail INSERT silently (don't inflate counts)
- Consumer always returns nil (Kafka commits offset, no infinite retry)
- Transient failures logged for monitoring but don't block pipeline

**Alternative approaches considered:**
1. **Processed message ID tracking table**: Extra table overhead, requires cleanup
2. **Redis-based deduplication**: Adds Redis dependency, TTL management complexity
3. **Kafka transactional writes**: Requires exactly-once semantics, heavy coordination overhead

**Decision:** Use natural key (short_code + timestamp) as idempotency key. Simple, no extra infrastructure, leverages PostgreSQL constraint enforcement.

### Pattern 3: zRPC Client Injection

**What:** Add Analytics RPC client to URL Service for calling GetClickCount
**When to use:** Cross-service synchronous calls (e.g., enriching link details with click counts)
**Example:**
```go
// Source: go-zero zRPC client configuration patterns
// https://github.com/zeromicro/zero-doc (RPC调用 examples)

// internal/config/config.go
package config

import (
  "github.com/zeromicro/go-zero/rest"
  "github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
  rest.RestConf
  BaseUrl      string
  DataSource   string
  KqPusherConf struct {
    Brokers []string
    Topic   string
  }
  AnalyticsRpc zrpc.RpcClientConf  // zRPC client config
}

// internal/svc/servicecontext.go
package svc

import (
  "go-shortener/services/analytics-rpc/analytics"
  "github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
  Config       config.Config
  UrlModel     model.UrlsModel
  KqPusher     *kq.Pusher
  AnalyticsRpc analytics.Analytics  // zRPC client
}

func NewServiceContext(c config.Config) *ServiceContext {
  conn := sqlx.NewSqlConn("postgres", c.DataSource)
  return &ServiceContext{
    Config:       c,
    UrlModel:     model.NewUrlsModel(conn),
    KqPusher:     kq.NewPusher(c.KqPusherConf.Brokers, c.KqPusherConf.Topic),
    AnalyticsRpc: analytics.NewAnalytics(zrpc.MustNewClient(c.AnalyticsRpc)),
  }
}

// etc/url.yaml
AnalyticsRpc:
  Target: 127.0.0.1:8081  # Direct connection mode (no Etcd)
  Timeout: 2000           # 2s timeout
```

**Built-in resilience:**
- Circuit breaker enabled by default (adaptive, Google SRE algorithm)
- Timeout enforcement (fail fast if Analytics is slow)
- Retry disabled by default (for idempotent operations, enable via config)

### Pattern 4: Consumer Service Main Entry

**What:** Standalone service that runs kq.Queue consumer
**When to use:** Kafka consumers should run independently from RPC services (scaling, failure isolation)
**Example:**
```go
// Source: go-queue consumer initialization pattern
// https://github.com/zeromicro/go-queue/blob/master/example/kq/consumer/queue.go

package main

import (
  "context"
  "flag"
  "go-shortener/services/analytics-consumer/internal/config"
  "go-shortener/services/analytics-consumer/internal/handler"
  "go-shortener/services/analytics-consumer/internal/svc"

  "github.com/zeromicro/go-queue/kq"
  "github.com/zeromicro/go-zero/core/conf"
  "github.com/zeromicro/go-zero/core/service"
)

var configFile = flag.String("f", "etc/consumer.yaml", "config file")

func main() {
  flag.Parse()

  var c config.Config
  conf.MustLoad(*configFile, &c)

  ctx := context.Background()
  svcCtx := svc.NewServiceContext(c)

  // Initialize consumer with handler
  consumer := kq.MustNewQueue(c.KqConsumerConf,
    kq.WithHandle(handler.NewClickEventHandler(ctx, svcCtx).Consume))

  // Service group for graceful shutdown
  serviceGroup := service.NewServiceGroup()
  defer serviceGroup.Stop()

  serviceGroup.Add(consumer)

  logx.Info("Starting analytics consumer...")
  serviceGroup.Start()
}
```

**Configuration (etc/consumer.yaml):**
```yaml
Name: analytics-consumer
Log:
  ServiceName: analytics-consumer
  Mode: console
  Level: info

KqConsumerConf:
  Brokers:
    - 127.0.0.1:9092
  Group: analytics-consumer
  Topic: click-events
  Offset: first       # Start from beginning (development)
  Consumers: 4        # Goroutines fetching from Kafka
  Processors: 8       # Goroutines processing messages
  MinBytes: 1048576   # 1MB min fetch size
  MaxBytes: 10485760  # 10MB max fetch size

DataSource: postgres://postgres:postgres@localhost:5433/shortener?sslmode=disable
GeoIPDbPath: data/GeoLite2-Country.mmdb
```

**Key parameters:**
- **Consumers**: Parallel fetchers from Kafka (match partition count)
- **Processors**: Parallel message handlers (CPU-bound enrichment work)
- **Offset**: `first` (reprocess from start) or `last` (only new messages)
- **Group**: Consumer group ID (Kafka tracks offset per group)

### Anti-Patterns to Avoid

- **Blocking redirect on Kafka publish**: NEVER call `kq.Pusher.Push()` synchronously in handler. Use `threading.GoSafe` for fire-and-forget.
- **Infinite retry on consumer errors**: Return `nil` from consumer handler to commit offset. Transient failures should be monitored, not retried indefinitely.
- **Missing panic recovery in goroutines**: NEVER use raw `go func()` for fire-and-forget. Use `threading.GoSafe` (go-zero's panic-safe wrapper).
- **Sharing database connections across goroutines unsafely**: go-zero's sqlx.SqlConn is safe for concurrent use, but raw sql.DB may not be in custom code.
- **ZooKeeper in new Kafka deployments**: Use KRaft mode (Kafka 3.x+). ZooKeeper is deprecated and adds operational overhead.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Kafka consumer group management | Manual offset tracking, partition assignment | kq.Queue | Handles consumer group protocol, offset commits, rebalancing automatically |
| Goroutine panic handling | Custom recover() wrappers | threading.GoSafe | go-zero's battle-tested wrapper with logging integration |
| Circuit breakers for zRPC | Custom failure counting | Built-in zRPC interceptors | Adaptive algorithm from Google SRE, automatically configured |
| GeoIP database parsing | Manual MMDB reader | oschwald/geoip2-golang | Official MaxMind library, handles binary format correctly |
| User agent parsing | Regex-based string matching | mileusna/useragent | Deterministic parser, handles edge cases, 10k+ UA strings tested |
| Message deduplication framework | Custom processed-message tracking | Database natural key constraints | Simpler, leverages RDBMS guarantees, no extra table maintenance |

**Key insight:** Kafka guarantees at-least-once delivery by design. Exactly-once semantics require distributed transactions (heavy coordination). For analytics use case, application-level idempotency via database constraints is simpler and sufficient.

## Common Pitfalls

### Pitfall 1: Kafka Connection String Confusion

**What goes wrong:** Pusher/Queue fail to connect with "connection refused" or timeout errors
**Why it happens:** Docker networks vs localhost confusion, wrong port, broker not advertising correct listener
**How to avoid:**
- For services in Docker Compose: Use service name as hostname (`kafka:9092`)
- For local development (services outside Docker): Use `localhost:9092` with PLAINTEXT listener
- Verify Kafka's `KAFKA_ADVERTISED_LISTENERS` matches your connection mode
- Test connection: `docker exec -it kafka kafka-topics --list --bootstrap-server localhost:9092`

**Warning signs:**
- Error: "connection refused" → Broker not reachable at specified host:port
- Error: "i/o timeout" → Firewall or Docker network misconfiguration
- Consumer starts but no messages consumed → Wrong topic name or partition assignment failed

### Pitfall 2: Fire-and-Forget Without Panic Recovery

**What goes wrong:** Panics in goroutines crash entire service (no handler can catch them)
**Why it happens:** Panics don't cross goroutine boundaries. Raw `go func()` has no recovery mechanism.
**How to avoid:**
- ALWAYS use `threading.GoSafe` instead of `go func()` for fire-and-forget tasks
- Import: `github.com/zeromicro/go-zero/core/threading`
- Pattern: `threading.GoSafe(func() { /* your code */ })`

**Warning signs:**
- Service crashes with stack trace showing panic in goroutine
- No graceful degradation (entire process exits)
- Error logs stop abruptly (process died before flushing)

### Pitfall 3: Consumer Offset Never Commits

**What goes wrong:** Consumer reprocesses same messages repeatedly after restart
**Why it happens:** Handler returns error, causing kq.Queue to NOT commit offset
**How to avoid:**
- Return `nil` from handler for transient errors (log error, let monitoring catch patterns)
- Only return error for truly unrecoverable scenarios (malformed message structure)
- Idempotent handler design: reprocessing same message multiple times is safe

**Warning signs:**
- Same message appears in logs repeatedly across restarts
- Consumer lag metric stays constant (not progressing)
- Duplicate click events inflating analytics counts

### Pitfall 4: Missing Idempotency Causes Double-Counting

**What goes wrong:** Same click event processed multiple times, analytics counts inflated
**Why it happens:** Kafka delivers at-least-once. Consumer crashes after processing but before offset commit → message redelivered.
**How to avoid:**
- Add unique constraint on clicks table: `UNIQUE (short_code, clicked_at_unix)`
- Insert will fail for duplicate events (same short_code + timestamp)
- Handler catches constraint violation, logs warning, returns nil (commits offset)

**Warning signs:**
- Click counts don't match URL redirect counts
- Analytics show suspiciously high click rates after consumer restarts
- Database constraint violations in logs

### Pitfall 5: GeoIP Database Not Found

**What goes wrong:** Consumer fails to start or country detection returns empty results
**Why it happens:** GeoIP MMDB file path incorrect or file not included in Docker image
**How to avoid:**
- Download GeoLite2-Country.mmdb from MaxMind (free account required)
- Place in `data/` directory (gitignored, not checked in)
- Mount as volume in Docker Compose: `./data:/app/data:ro`
- Fallback pattern: If reader.Open() fails, log warning and use "Unknown" for all lookups

**Warning signs:**
- Error: "no such file or directory" for .mmdb file
- All click events show country_code = NULL or "Unknown"
- Consumer crashes on startup with file path error

### Pitfall 6: zRPC Timeout Too Short

**What goes wrong:** URL Service calls to Analytics RPC fail with "deadline exceeded" errors
**Why it happens:** Analytics query is slow (database index missing, large dataset scan)
**How to avoid:**
- Set reasonable timeout in config: `Timeout: 2000` (2 seconds is typical for simple queries)
- Monitor Analytics RPC response times (add logging/metrics)
- Circuit breaker will open automatically if failure rate is high (protects URL Service)
- Fallback: Return link details without click count if Analytics call fails

**Warning signs:**
- Error: "context deadline exceeded" in URL Service logs
- GetLinkDetail endpoint slow or returns 500 errors
- Circuit breaker opens (Analytics RPC marked unavailable)

### Pitfall 7: Kafka Topic Not Created Before Consumer Starts

**What goes wrong:** Consumer blocks indefinitely waiting for topic to exist
**Why it happens:** Kafka auto-create disabled or consumer starts before first publish
**How to avoid:**
- Enable auto-create in Kafka config: `KAFKA_AUTO_CREATE_TOPICS_ENABLE: true`
- OR pre-create topic: `docker exec kafka kafka-topics --create --topic click-events --bootstrap-server localhost:9092`
- Consumer will block until topic exists (not an error, but delays startup)

**Warning signs:**
- Consumer log shows "waiting for topic metadata" repeatedly
- No error messages but consumer never processes events
- `kafka-topics --list` doesn't show expected topic

## Code Examples

Verified patterns from official sources:

### Kafka Pusher Initialization

```go
// Source: go-zero official Kafka integration docs
// https://go-zero.dev/en/docs/tutorials/message-queue/kafka

package svc

import (
  "github.com/zeromicro/go-queue/kq"
)

type ServiceContext struct {
  Config   config.Config
  KqPusher *kq.Pusher
}

func NewServiceContext(c config.Config) *ServiceContext {
  return &ServiceContext{
    Config:   c,
    KqPusher: kq.NewPusher(c.KqPusherConf.Brokers, c.KqPusherConf.Topic),
  }
}

// Config structure
type Config struct {
  rest.RestConf
  KqPusherConf struct {
    Brokers []string
    Topic   string
  }
}
```

### Kafka Consumer Initialization

```go
// Source: go-queue repository examples
// https://github.com/zeromicro/go-queue/blob/master/example/kq/consumer/queue.go

package main

import (
  "context"
  "github.com/zeromicro/go-queue/kq"
)

func main() {
  var c kq.KqConf
  conf.MustLoad("config.yaml", &c)

  q := kq.MustNewQueue(c, kq.WithHandle(func(ctx context.Context, k, v string) error {
    // Process message
    fmt.Printf("Key: %s, Value: %s\n", k, v)
    return nil  // Return nil to commit offset
  }))

  defer q.Stop()
  q.Start()  // Blocks until service stops
}
```

### GeoIP Country Lookup

```go
// Source: oschwald/geoip2-golang official README
// https://github.com/oschwald/geoip2-golang

package handler

import (
  "net"
  "github.com/oschwald/geoip2-golang"
  "github.com/zeromicro/go-zero/core/logx"
)

type ClickEventHandler struct {
  geoipReader *geoip2.Reader
}

func NewClickEventHandler(dbPath string) (*ClickEventHandler, error) {
  reader, err := geoip2.Open(dbPath)
  if err != nil {
    return nil, err
  }
  return &ClickEventHandler{geoipReader: reader}, nil
}

func (h *ClickEventHandler) enrichGeoIP(ipStr string) string {
  ip := net.ParseIP(ipStr)
  if ip == nil {
    return "Unknown"
  }

  record, err := h.geoipReader.Country(ip)
  if err != nil {
    logx.Errorw("geoip lookup failed", logx.Field("error", err.Error()))
    return "Unknown"
  }

  return record.Country.IsoCode  // e.g., "US", "GB", "CN"
}

func (h *ClickEventHandler) Close() {
  h.geoipReader.Close()
}
```

### User Agent Parsing

```go
// Source: mileusna/useragent official README
// https://github.com/mileusna/useragent

package handler

import (
  "github.com/mileusna/useragent"
)

func (h *ClickEventHandler) enrichUserAgent(uaStr string) (deviceType, browser string) {
  ua := useragent.Parse(uaStr)

  // Device type
  if ua.Mobile {
    deviceType = "mobile"
  } else if ua.Tablet {
    deviceType = "tablet"
  } else if ua.Desktop {
    deviceType = "desktop"
  } else if ua.Bot {
    deviceType = "bot"
  } else {
    deviceType = "unknown"
  }

  // Browser
  browser = ua.Name  // e.g., "Chrome", "Firefox", "Safari"
  if browser == "" {
    browser = "Unknown"
  }

  return deviceType, browser
}
```

### Referer Traffic Source Extraction

```go
// Source: Common URL shortener pattern
// No external library needed

package handler

import (
  "net/url"
  "strings"
)

func (h *ClickEventHandler) enrichReferer(referer string) string {
  if referer == "" {
    return "direct"
  }

  u, err := url.Parse(referer)
  if err != nil || u.Host == "" {
    return "direct"
  }

  // Remove www prefix for cleaner domains
  domain := strings.TrimPrefix(u.Host, "www.")

  // Common social media and search engines
  switch {
  case strings.Contains(domain, "google.com"):
    return "google"
  case strings.Contains(domain, "facebook.com"):
    return "facebook"
  case strings.Contains(domain, "twitter.com"), strings.Contains(domain, "x.com"):
    return "twitter"
  case strings.Contains(domain, "linkedin.com"):
    return "linkedin"
  case strings.Contains(domain, "reddit.com"):
    return "reddit"
  default:
    return domain  // Generic domain as traffic source
  }
}
```

### zRPC Client Usage

```go
// Source: go-zero zRPC documentation
// https://github.com/zeromicro/zero-doc (RPC调用 examples)

package logic

import (
  "context"
  "go-shortener/services/analytics-rpc/analytics"
  "github.com/zeromicro/go-zero/core/logx"
)

func (l *GetLinkDetailLogic) GetLinkDetail(req *types.GetLinkDetailRequest) (*types.GetLinkDetailResponse, error) {
  // ... fetch URL from database ...

  // Call Analytics RPC to get click count
  resp, err := l.svcCtx.AnalyticsRpc.GetClickCount(l.ctx, &analytics.GetClickCountRequest{
    ShortCode: req.Code,
  })

  if err != nil {
    // Circuit breaker may be open or service down
    logx.WithContext(l.ctx).Errorw("analytics rpc failed",
      logx.Field("error", err.Error()))
    // Fallback: return link details without click count
    return &types.GetLinkDetailResponse{
      ShortCode:   url.ShortCode,
      OriginalUrl: url.OriginalUrl,
      TotalClicks: 0,  // Fallback value
    }, nil
  }

  return &types.GetLinkDetailResponse{
    ShortCode:   url.ShortCode,
    OriginalUrl: url.OriginalUrl,
    TotalClicks: resp.TotalClicks,
  }, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| ZooKeeper-based Kafka | KRaft mode (ZK-less) | Kafka 3.0 (2021), production-ready 3.3+ | Simpler deployment, fewer moving parts, better performance |
| Dapr pub/sub | go-zero go-queue (kq) | go-zero v1.3+ | Native integration, less abstraction overhead, production patterns built-in |
| Manual consumer offset management | kq.Queue auto-commit | go-queue v1.0+ | Less boilerplate, correct offset handling out-of-box |
| regex-based UA parsing | Dedicated UA parser libraries | 2015+ | Better edge case handling, mobile/tablet/desktop distinction |
| MaxMind GeoIP Legacy | GeoIP2 databases | 2013 | More accurate, IPv6 support, MMDB binary format |

**Deprecated/outdated:**
- **wurstmeister/kafka-docker**: Use official `apache/kafka` image (KRaft support, actively maintained)
- **Shopify/sarama**: Still works but go-queue uses segmentio/kafka-go (simpler API, better performance)
- **ZooKeeper**: Deprecated in Kafka 3.x, removed in Kafka 4.0. Use KRaft mode.

## Open Questions

1. **Consumer scaling strategy for production**
   - What we know: kq.Queue supports multiple consumer instances in same group (Kafka handles partition assignment)
   - What's unclear: Optimal Consumers vs Processors ratio for CPU vs I/O bound enrichment work
   - Recommendation: Start with Consumers=4, Processors=8 (2:1 ratio). Profile CPU usage and adjust.

2. **GeoIP database update frequency**
   - What we know: MaxMind updates GeoLite2 weekly, accuracy improves with fresh data
   - What's unclear: How to automate database updates without service restart
   - Recommendation: Phase 9 use static file. Phase 10 consider reload mechanism or periodic consumer restart.

3. **Kafka partition count for click-events topic**
   - What we know: More partitions = better parallelism but higher coordination overhead
   - What's unclear: Expected click event throughput (depends on production traffic)
   - Recommendation: Start with 4 partitions (matches Consumers=4). Scale up based on lag metrics.

4. **Circuit breaker tuning for Analytics RPC**
   - What we know: go-zero uses adaptive breaker (Google SRE algorithm) with sane defaults
   - What's unclear: Whether default `K=1.5` is optimal for Analytics query patterns
   - Recommendation: Start with defaults. Monitor false-positive circuit opens. Tune K if needed.

5. **Should GetClickCount call be cached?**
   - What we know: Click counts change frequently, caching reduces RPC load
   - What's unclear: Whether cache hit rate justifies added complexity (cache invalidation)
   - Recommendation: No caching in Phase 9 (simplicity). Add Redis cache in Phase 10 if metrics show high GetClickCount QPS.

## Sources

### Primary (HIGH confidence)

- [go-zero/zero-doc](https://github.com/zeromicro/zero-doc) - Official go-zero documentation (zRPC patterns, configuration)
- [go-zero framework](https://github.com/zeromicro/go-zero) - Source code (threading.GoSafe, kq integration)
- [go-queue repository](https://github.com/zeromicro/go-queue) - Kafka consumer/producer examples
- [go-queue consumer example](https://github.com/zeromicro/go-queue/blob/master/example/kq/consumer/queue.go) - kq.Queue usage pattern
- [go-zero Kafka tutorial](https://go-zero.dev/en/docs/tutorials/message-queue/kafka) - Pusher and Queue configuration
- [oschwald/geoip2-golang](https://pkg.go.dev/github.com/oschwald/geoip2-golang) - Official MaxMind GeoIP2 Go library
- [mileusna/useragent](https://github.com/mileusna/useragent) - User agent parser with deterministic approach

### Secondary (MEDIUM confidence)

- [Kafka Docker setup (OneUpTime)](https://oneuptime.com/blog/post/2026-01-21-kafka-docker-compose/view) - KRaft mode Docker Compose configuration (2026-01-21)
- [Baeldung Kafka Docker guide](https://www.baeldung.com/ops/kafka-docker-setup) - Production Docker patterns
- [Confluent Docker guide](https://www.confluent.io/blog/how-to-use-kafka-docker-composer/) - Official Kafka Docker Compose best practices
- [Idempotent Kafka consumer (Nejc Korasa)](https://nejckorasa.github.io/posts/idempotent-kafka-procesing/) - Database-backed idempotency pattern
- [Medium: Building Resilient IP Geolocation with Go](https://medium.com/@sattarkars45/building-a-resilient-ip-geolocation-system-with-maxmind-geoip2-and-go-c1db843d92f5) - GeoIP integration patterns

### Tertiary (LOW confidence)

- [Device detector Go libraries](https://github.com/topics/user-agent-parser) - Alternative UA parser options (not used but researched)
- [Kafka exactly-once semantics](https://www.confluent.io/blog/exactly-once-semantics-are-possible-heres-how-apache-kafka-does-it/) - Background on delivery guarantees

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Context7 verified go-zero kq support, official library docs checked
- Architecture: HIGH - Official go-queue examples, go-zero ServiceContext patterns verified
- Pitfalls: MEDIUM-HIGH - Some from docs, some inferred from common Kafka/Docker issues

**Research date:** 2026-02-16
**Valid until:** 2026-03-16 (30 days - stable ecosystem, Kafka 3.x mature)

**Note on Context7 limitations:** Context7 had limited kq-specific documentation. Primary research came from:
1. go-zero source code (threading.GoSafe usage patterns)
2. go-queue repository examples (kq.Pusher, kq.Queue)
3. Official go-zero.dev Kafka tutorial

All patterns cross-verified with official sources listed in Sources section.
