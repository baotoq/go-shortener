# Pitfalls Research

**Domain:** Go + Dapr URL Shortener Microservices
**Researched:** 2026-02-14
**Confidence:** MEDIUM-HIGH

## Critical Pitfalls

### Pitfall 1: SQLite Concurrent Write Corruption

**What goes wrong:**
SQLite database becomes corrupted when multiple Dapr sidecars attempt concurrent writes to the same database file. Data loss occurs, application crashes with database lock errors.

**Why it happens:**
Developers use SQLite for simplicity in "production-ready" deployments without understanding it does not support concurrent writes. Multiple service instances or both URL Service and Analytics Service sharing the same SQLite file creates race conditions.

**How to avoid:**
- Use SQLite only for single-instance development/testing
- In production, use a proper concurrent database (PostgreSQL, MySQL, or distributed state store)
- Never mount the same SQLite file to multiple Dapr sidecars
- Configure one SQLite database per service if using SQLite in development

**Warning signs:**
- "database is locked" errors in logs
- Intermittent state store failures
- Database file corruption messages
- Works in single instance but fails when scaling

**Phase to address:**
Phase 1 (Foundation) - Document SQLite limitations clearly in setup; Phase 3+ (Production) - Migrate to concurrent database before multi-instance deployment

**Source confidence:** HIGH - [Official Dapr SQLite docs](https://docs.dapr.io/reference/components-reference/supported-state-stores/setup-sqlite/)

---

### Pitfall 2: Missing Context Propagation in Service Chains

**What goes wrong:**
Timeouts fail to cascade properly. Parent request times out but downstream service continues processing. Resources leak. Debugging is impossible because trace context is lost between services.

**Why it happens:**
Developers create new contexts with `context.Background()` in handler/service layers instead of propagating the request context. Not understanding that "only the outer layer knows how long a request should live."

**How to avoid:**
- Always propagate incoming request context through entire call chain
- Pass context as first parameter to all service/repository methods
- Use `r.Context()` from HTTP request, never `context.Background()`
- Set timeouts only at API boundary (handler layer)
- Verify context cancellation with `select { case <-ctx.Done(): return ctx.Err() }`

**Warning signs:**
- Timeouts not working as expected
- "context deadline exceeded" only at top level, never in logs from downstream services
- Resource cleanup not happening (DB connections, HTTP clients left open)
- Distributed traces show disconnected spans

**Phase to address:**
Phase 1 (Foundation) - Establish context propagation pattern from first handler; Phase 2 (Testing) - Add tests verifying context timeout propagation

**Source confidence:** HIGH - [Grab Engineering on context deadlines](https://engineering.grab.com/context-deadlines-and-how-to-set-them), [Rotational Labs on microservice context chains](https://rotational.io/blog/contexts-in-go-microservice-chains/)

---

### Pitfall 3: Pub/Sub Retry Explosion (Double Retry Problem)

**What goes wrong:**
Messages retry exponentially more times than intended. A single failed message creates a retry storm. Message broker queues fill up. System becomes unresponsive under error conditions.

**Why it happens:**
Dapr resiliency policies augment (not replace) the message broker's built-in retries. Configuring retries without understanding the pub/sub component's implicit retry behavior creates multiplicative retries. Example: 3 Dapr retries × 5 broker retries = 15 total attempts.

**How to avoid:**
- Read your pub/sub component's retry documentation before configuring Dapr resiliency
- Understand that Dapr resiliency augments, not overrides, broker retries
- Configure dead letter topics for non-retriable failures
- Set max retry limits on both Dapr and broker sides
- Make message handlers idempotent (critical for at-least-once delivery)

**Warning signs:**
- Messages retrying far more than configured retry count
- Same message processed multiple times in quick succession
- Logs show "clustered" retry patterns (bursts of same message)
- Queue depth growing under error conditions

**Phase to address:**
Phase 2 (Analytics Service) - Configure dead letter topics and idempotent handlers from day one; Phase 3 (Production) - Load test retry behavior before deployment

**Source confidence:** HIGH - [Dapr pub/sub troubleshooting](https://docs.dapr.io/developing-applications/sdks/dotnet/dotnet-troubleshooting/dotnet-troubleshooting-pubsub/), [Diagrid fault tolerance blog](https://www.diagrid.io/blog/fault-tolerant-microservices-made-easy-with-dapr-resiliency)

---

### Pitfall 4: Sidecar Resource Limits Not Set

**What goes wrong:**
Dapr sidecars consume unbounded memory/CPU. Sidecars get OOM killed in production. Node resources exhausted. Application appears to work in development but fails under load.

**Why it happens:**
Default Dapr sidecar configuration has no resource limits. Developers test locally without resource constraints. Production Kubernetes doesn't enforce limits until nodes are starved.

**How to avoid:**
- Set sidecar resource limits from day one: CPU limit 300m/request 100m, Memory limit 1000Mi/request 250Mi
- Configure `GOMEMLIMIT` to 90% of memory limit (prevents OOM kills)
- Monitor actual sidecar resource usage and tune based on workload
- Use resource quotas in Kubernetes namespaces to catch missing limits early

**Warning signs:**
- Sidecars restarting with OOM errors under load
- Node CPU/memory pressure correlating with traffic spikes
- `kubectl top pods` shows sidecars using more resources than application
- Works in local Docker but fails in Kubernetes

**Phase to address:**
Phase 3 (Production Readiness) - Add resource limits to all Dapr configurations before Kubernetes deployment

**Source confidence:** HIGH - [Dapr Kubernetes production guidelines](https://docs.dapr.io/operations/hosting/kubernetes/kubernetes-production/)

---

### Pitfall 5: Testing Against Mocks Instead of Dapr Contract

**What goes wrong:**
Tests pass but integration fails. Dapr behavior differs from mocks. Production issues emerge that tests never caught. False confidence in test coverage.

**Why it happens:**
Developers mock Dapr client interfaces without understanding actual Dapr semantics (retries, timeouts, serialization). Mocks return success but real Dapr has failure modes. Clean architecture creates distance between code and Dapr, hiding integration issues.

**How to avoid:**
- Write integration tests against real Dapr sidecar (use `dapr run` in CI)
- Mock only at repository boundary, not Dapr client
- Test with real Dapr components configured (even if in-memory like Redis)
- Verify serialization/deserialization with actual Dapr state store
- Include failure mode tests (sidecar unavailable, state store timeout)

**Warning signs:**
- 100% unit test coverage but integration issues in staging
- Serialization errors in production that tests never caught
- Tests don't require Dapr sidecar to run
- No integration tests in CI pipeline

**Phase to address:**
Phase 1 (Foundation) - Establish integration testing pattern with Dapr from first service; Phase 2 (Testing) - Add Dapr integration tests to CI

**Source confidence:** MEDIUM - [Diagrid Dapr testing blog](https://www.diagrid.io/blog/mastering-the-dapr-inner-development-loop-part-1-code-unit-test), [Go testing best practices](https://dasroot.net/posts/2026/01/go-testing-excellence-table-driven-tests-mocking/)

---

### Pitfall 6: Ignoring the Read/Write Imbalance

**What goes wrong:**
URL redirects (reads) are slow. Database becomes bottleneck. Caching strategy is an afterthought. System can't scale to handle traffic.

**Why it happens:**
Developers treat URL shortener as "simple CRUD" without recognizing the 100:1 read/write ratio. Focus on URL creation (write path) instead of redirect (read path). No caching strategy from the start.

**How to avoid:**
- Optimize aggressively for read path from Phase 1
- Cache shortened URL mappings (in-memory, Redis, or Dapr state store with TTL)
- Use appropriate indexes on lookup columns
- Measure redirect latency separately from creation latency
- Design schema for fast single-key lookups, not complex queries

**Warning signs:**
- Redirect endpoint slower than create endpoint
- Database CPU high on read operations
- No caching layer in architecture
- Redirect latency grows with database size

**Phase to address:**
Phase 1 (Foundation) - Design for read optimization; Phase 2+ - Add caching before performance becomes issue

**Source confidence:** HIGH - [System Design School URL shortener](https://systemdesignschool.io/problems/url-shortener/solution), [Hello Interview Bit.ly design](https://www.hellointerview.com/learn/system-design/problem-breakdowns/bitly)

---

### Pitfall 7: Service Invocation Without Resiliency Policies

**What goes wrong:**
Transient network failures cause complete request failures. One slow service cascades to all callers. No retry, timeout, or circuit breaker protection. Availability drops under partial failures.

**Why it happens:**
Developers assume Dapr handles resiliency automatically. Not configuring explicit resiliency policies. Not understanding that retries should only apply to idempotent operations.

**How to avoid:**
- Configure Dapr resiliency policies explicitly for service invocation
- Set appropriate timeouts based on SLAs (not arbitrary defaults)
- Use circuit breakers to prevent cascading failures
- Only retry idempotent operations
- Add constant or exponential backoff to retries

**Warning signs:**
- Transient failures never recover automatically
- Single service failure takes down entire system
- No retry behavior visible in logs
- Timeout errors with no circuit breaker activation

**Phase to address:**
Phase 2 (Service Integration) - Add resiliency policies when URL Service calls Analytics Service; Phase 3 (Production) - Load test failure scenarios

**Source confidence:** HIGH - [Dapr resiliency policies](https://docs.dapr.io/operations/resiliency/policies/), [Diagrid resiliency blog](https://www.diagrid.io/blog/fault-tolerant-microservices-made-easy-with-dapr-resiliency)

---

### Pitfall 8: Sidecar Health Check Misconfiguration

**What goes wrong:**
Pods in CrashLoopBackoff. Kubernetes kills healthy pods. Dapr sidecar initializes slower than health probe timeout. Service appears down when it's actually starting.

**Why it happens:**
Default Kubernetes liveness probe timeouts are too aggressive for Dapr initialization. Slow component initialization (state store connection, pub/sub setup) exceeds health check deadlines. Developers don't tune probe delays.

**How to avoid:**
- Increase liveness probe initial delay significantly (30-60s minimum)
- Configure startup probes separately from liveness probes (Kubernetes 1.16+)
- Use Dapr's health endpoint `/v1.0/healthz` for probes
- Check component initialization time in logs and set delays accordingly
- Enable debug logging to diagnose slow component loading

**Warning signs:**
- Pods restart repeatedly during initialization
- "CrashLoopBackoff" status in `kubectl get pods`
- Logs show components initializing successfully, then pod restarts
- Works locally but fails in Kubernetes

**Phase to address:**
Phase 3 (Kubernetes Deployment) - Configure health probes before first Kubernetes deployment

**Source confidence:** HIGH - [Dapr common issues](https://docs.dapr.io/operations/troubleshooting/common_issues/), [Dapr sidecar health](https://docs.dapr.io/operations/resiliency/health-checks/sidecar-health/)

---

## Technical Debt Patterns

Shortcuts that seem reasonable but create long-term problems.

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Using SQLite in production | No setup, simple, works locally | Cannot scale horizontally, concurrent write corruption, data loss | Never in production. Development/testing only. |
| Skipping resiliency policies | Faster development, less config | Cascading failures, no fault tolerance, poor availability | Never. Critical for production microservices. |
| Hardcoding Dapr app IDs | Quick service invocation setup | No service discovery, breaks in orchestrated environments | Only for single-environment prototypes. |
| Not setting resource limits | Works in development, no tuning needed | OOM kills, node starvation, unpredictable performance | Never. Set from day one even if generous. |
| Mocking Dapr instead of integration tests | Fast tests, no Dapr dependency | False confidence, integration bugs in production | Unit tests only. Must have integration tests too. |
| Sharing state between services | Simpler architecture, one database | Tight coupling, concurrent access issues, violates service boundaries | Never. Each service owns its state. |
| Using context.Background() everywhere | Compiles fine, no errors | Timeouts don't work, resource leaks, impossible debugging | Never. Always propagate request context. |
| Disabling mTLS for "simpler" development | Easier local testing | Security vulnerabilities, production/dev parity issues | Only local development. Enable for staging/prod. |

---

## Integration Gotchas

Common mistakes when connecting to external services.

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| Dapr State Store | Assuming transactions work across different keys | Use single-key operations or state store's native transactions. Dapr doesn't provide distributed transactions. |
| Dapr Pub/Sub | Not configuring dead letter topics | Always configure DLT. Messages will retry forever or get lost without it. |
| Dapr Service Invocation | Using gRPC streaming with resiliency policies | Resiliency only applies to initial handshake. Stream interruptions don't auto-reconnect. Use HTTP or handle reconnection manually. |
| Docker networking with Dapr | Not setting `DAPR_HOST_IP` environment variable | Set `DAPR_HOST_IP` or service invocation fails with connection errors. |
| Dapr Components | Creating components in wrong Kubernetes namespace | Components must be in same namespace as application, or use shared namespace scope. |
| SQLite State Store | Running over network filesystem (NFS/SMB) | SQLite over network causes corruption. Use local disk only. |
| Message Broker | Assuming auto-topic-creation in production | Many orgs disable auto-creation. Manually create topics via CLI/console. |
| Dapr in Docker Compose | Using production component configs | Use in-memory components (Redis) for local dev, not production state stores. |

---

## Performance Traps

Patterns that work at small scale but fail as usage grows.

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|----------------|
| No caching on redirect path | Slow redirects, high DB load | Cache URL mappings with TTL, use in-memory or Redis | >1000 redirects/sec |
| Linear scan for short URL generation | Slow URL creation, race conditions | Use distributed ID generation (Snowflake) or hash-based approach | >100 concurrent creates |
| Single database instance | High latency, read/write contention | Read replicas for redirect path, write master for creation | >10k requests/sec |
| Sidecar overhead on every call | Latency budget consumed by hops | Optimize sidecar config, use gRPC over HTTP, batch operations | Latency-sensitive (<10ms) |
| Unbounded retry loops | Resource exhaustion during outages | Max retry limits, exponential backoff, circuit breakers | First major incident |
| Synchronous analytics events | Redirect latency includes analytics time | Pub/sub for async analytics, never block redirect path | >100ms analytics processing |
| No connection pooling | Connection exhaustion, slow queries | Configure DB connection pools in repositories | >50 concurrent connections |
| Large payload serialization | High CPU, slow Dapr calls | Use appropriate data structures, avoid sending full objects | Payloads >10KB |

---

## Security Mistakes

Domain-specific security issues beyond general web security.

| Mistake | Risk | Prevention |
|---------|------|------------|
| Predictable short URL generation (sequential IDs) | Users can enumerate all shortened URLs, discover private links | Use random/hash-based generation with sufficient entropy (6+ char base62) |
| No validation on target URLs | Open redirect vulnerability, phishing attacks | Whitelist allowed domains or add warning page for external redirects |
| Storing secrets in Dapr component YAML | Secrets in git, exposed in Kubernetes | Use Dapr secret stores (Kubernetes secrets, HashiCorp Vault, Azure Key Vault) |
| Disabling mTLS between services | Service spoofing, eavesdropping on inter-service traffic | Enable Dapr mTLS in production, use app-to-Dapr API tokens |
| No rate limiting on URL creation | DoS via unlimited URL creation, database bloat | Implement rate limiting per IP/user in handler layer |
| Logging full URLs in analytics | Privacy leaks, sensitive data in logs | Hash or truncate URLs in logs, sanitize sensitive query params |
| Unrestricted URL creation | Spam, abuse, malicious links | Require authentication, implement moderation, track abuse patterns |

---

## Architectural Pitfalls

Mistakes in system design and service boundaries.

| Pitfall | Impact | Better Approach |
|---------|--------|-----------------|
| Shared database between URL and Analytics services | Tight coupling, schema conflicts, concurrent access issues | Each service owns its database. Analytics reads via pub/sub events, not direct DB access. |
| Synchronous analytics on redirect path | Redirect latency includes analytics processing, blocks on analytics failures | Async pub/sub events. Fire-and-forget. Analytics never blocks redirects. |
| No saga/compensation for distributed operations | Partial failures leave inconsistent state | Use Dapr workflows or implement saga pattern for multi-service operations (future concern). |
| Putting business logic in Dapr components | Vendor lock-in, untestable logic, violates clean architecture | Components are infrastructure. Business logic stays in service layer. |
| Over-abstracting Dapr interfaces | Premature abstraction, harder to debug, lost Dapr features | Use Dapr SDK directly in repository layer. Don't hide Dapr behind custom abstractions unless needed. |
| Mixing HTTP and gRPC without strategy | Inconsistent error handling, different retry semantics | Choose one protocol per service type. HTTP for external APIs, gRPC for internal services (or vice versa). |
| No service versioning strategy | Breaking changes break all clients simultaneously | Plan for versioning from Phase 1 (URL path versioning simplest for HTTP APIs). |

---

## "Looks Done But Isn't" Checklist

Things that appear complete but are missing critical pieces.

- [ ] **Dapr Components:** Often missing `scopes` field - verify components load only for intended apps, not globally
- [ ] **Health Probes:** Often missing adjusted `initialDelaySeconds` - verify probes account for Dapr initialization time
- [ ] **Resource Limits:** Often missing on Dapr sidecars - verify both app and sidecar have limits/requests set
- [ ] **Resiliency Policies:** Often missing timeout configuration - verify policies include timeouts, not just retries
- [ ] **Context Propagation:** Often missing in service layer - verify context passed through all layers, not recreated
- [ ] **Idempotency:** Often missing in pub/sub handlers - verify handlers can safely process duplicate messages
- [ ] **Dead Letter Topics:** Often missing in pub/sub config - verify DLT configured for all subscriptions
- [ ] **Integration Tests:** Often missing Dapr sidecar - verify tests run against real Dapr, not mocks only
- [ ] **Error Handling:** Often missing context.Canceled checks - verify proper handling of timeout/cancellation vs. other errors
- [ ] **Observability:** Often missing trace propagation - verify distributed traces connect across service boundaries
- [ ] **Secret Management:** Often missing secret store refs - verify no hardcoded secrets in component YAML
- [ ] **Database Indexes:** Often missing on lookup columns - verify indexes on `short_code` for fast redirect lookups

---

## Recovery Strategies

When pitfalls occur despite prevention, how to recover.

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| SQLite corruption in production | MEDIUM-HIGH | 1. Immediately switch traffic to healthy instance 2. Deploy proper DB (PostgreSQL) 3. Migrate data from backup 4. Add multi-instance tests to CI |
| Context timeout not propagating | LOW | 1. Add context parameter to all functions 2. Replace `context.Background()` with propagated context 3. Add integration test verifying timeout cascade |
| Retry explosion from double retries | MEDIUM | 1. Disable Dapr retry OR broker retry (choose one) 2. Configure dead letter topic 3. Drain and reset message queues 4. Redeploy with fixed config |
| Sidecar OOM kills | LOW | 1. Increase memory limits as hotfix 2. Add `GOMEMLIMIT` 3. Monitor actual usage 4. Tune to appropriate values |
| Missing integration tests | MEDIUM | 1. Add `dapr run` to CI pipeline 2. Write tests for critical paths (URL create, redirect, analytics event) 3. Test against real components |
| Slow redirects without caching | MEDIUM | 1. Add in-memory cache layer 2. Configure Dapr state store with TTL 3. Add cache-aside pattern in repository |
| No resiliency policies | LOW-MEDIUM | 1. Create resiliency spec YAML 2. Configure timeouts/retries/circuit breakers 3. Test failure scenarios 4. Deploy and monitor |
| Hardcoded Dapr app IDs | LOW | 1. Extract to config/env vars 2. Use service discovery 3. Update deployment manifests |

---

## Pitfall-to-Phase Mapping

How roadmap phases should address these pitfalls.

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| SQLite concurrent writes | Phase 1 (Foundation) | Document limitations; Phase 3 (Production) - Migrate before multi-instance deployment | Test with multiple instances, verify no corruption |
| Missing context propagation | Phase 1 (Foundation) | Establish pattern from first handler | Integration test with timeout, verify cascade |
| Pub/Sub retry explosion | Phase 2 (Analytics Service) | Configure DLT and idempotent handlers | Load test with forced failures, count retries |
| Sidecar resource limits | Phase 3 (Production Readiness) | Add limits to all Dapr configs | `kubectl describe pod`, verify limits present |
| Testing against mocks only | Phase 1 (Foundation) & Phase 2 (Testing) | Add Dapr integration tests to CI | CI must run `dapr run`, tests must pass |
| Read/write imbalance | Phase 1 (Foundation) | Design for read optimization from start | Benchmark redirect latency <10ms |
| No resiliency policies | Phase 2 (Service Integration) | Add resiliency when services communicate | Test timeout/retry/circuit breaker scenarios |
| Sidecar health check misconfiguration | Phase 3 (Kubernetes Deployment) | Configure probes before K8s deployment | Deploy to K8s, verify no CrashLoopBackoff |
| Predictable short URLs | Phase 1 (Foundation) | Use random/hash-based generation | Security review, attempt enumeration |
| Shared database anti-pattern | Phase 1 (Foundation) & Phase 2 (Analytics) | Each service owns its database | Schema changes don't affect other service |
| No distributed tracing | Phase 3 (Production Readiness) | Configure OpenTelemetry/Jaeger | Verify traces span URL → Analytics services |
| Missing dead letter topics | Phase 2 (Analytics Service) | Configure DLT with pub/sub component | Force handler failure, verify DLT receives message |

---

## Sources

### Official Documentation (HIGH Confidence)
- [Dapr SQLite State Store](https://docs.dapr.io/reference/components-reference/supported-state-stores/setup-sqlite/)
- [Dapr Common Issues](https://docs.dapr.io/operations/troubleshooting/common_issues/)
- [Dapr Kubernetes Production Guidelines](https://docs.dapr.io/operations/hosting/kubernetes/kubernetes-production/)
- [Dapr Service Invocation Overview](https://docs.dapr.io/developing-applications/building-blocks/service-invocation/service-invocation-overview/)
- [Dapr Pub/Sub Troubleshooting](https://docs.dapr.io/developing-applications/sdks/dotnet/dotnet-troubleshooting/dotnet-troubleshooting-pubsub/)
- [Dapr Resiliency Policies](https://docs.dapr.io/operations/resiliency/policies/)
- [Dapr Observability](https://docs.dapr.io/concepts/observability-concept/)

### Engineering Blogs (MEDIUM-HIGH Confidence)
- [Grab Engineering: Context Deadlines](https://engineering.grab.com/context-deadlines-and-how-to-set-them)
- [Rotational Labs: Contexts in Go Microservice Chains](https://rotational.io/blog/contexts-in-go-microservice-chains/)
- [Diagrid: Fault Tolerant Microservices with Dapr Resiliency](https://www.diagrid.io/blog/fault-tolerant-microservices-made-easy-with-dapr-resiliency)
- [Diagrid: Dapr Inner Development Loop - Unit Testing](https://www.diagrid.io/blog/mastering-the-dapr-inner-development-loop-part-1-code-unit-test)

### System Design Resources (MEDIUM Confidence)
- [System Design School: URL Shortener](https://systemdesignschool.io/problems/url-shortener/solution)
- [Hello Interview: Design Bit.ly](https://www.hellointerview.com/learn/system-design/problem-breakdowns/bitly)

### Community Resources (MEDIUM Confidence)
- [Go Testing Excellence 2026](https://dasroot.net/posts/2026/01/go-testing-excellence-table-driven-tests-mocking/)
- [Three Dots Labs: Clean Architecture in Go](https://threedots.tech/post/introducing-clean-architecture/)
- [Better Stack: Golang Timeouts](https://betterstack.com/community/guides/scaling-go/golang-timeouts/)

### Event Sourcing & Consistency (MEDIUM Confidence)
- [Azure: Event Sourcing Pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/event-sourcing)
- [SoftwareMill: Event Sourcing Consistency](https://softwaremill.com/things-i-wish-i-knew-when-i-started-with-event-sourcing-part-2-consistency/)

---

*Pitfalls research for: Go + Dapr URL Shortener Microservices*
*Researched: 2026-02-14*
*Next: Use this to inform phase planning and prevent common mistakes in roadmap execution*
