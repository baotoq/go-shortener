# Phase 14: Log Aggregation - Research

**Researched:** 2026-02-22
**Domain:** go-zero logx structured JSON logging + Loki + Grafana Alloy Docker Compose
**Confidence:** HIGH

---

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| LOG-01 | Services emit structured JSON logs via go-zero logx | go-zero logx `Encoding` defaults to `json` — currently the docker YAML configs set `Mode: console` without `Encoding`, which still defaults to JSON per LogConf struct. Adding explicit `Encoding: json` to all 6 YAML configs makes it unambiguous and verifiable. No Go code changes required. |
| LOG-02 | Loki aggregates logs from all three services in Docker Compose | Add `grafana/loki:3.0.0` service to docker-compose.yml with a `loki-config.yaml` using single-binary filesystem storage (no MinIO/S3 required). Loki exposes port 3100 for push and query. |
| LOG-03 | Log shipper (Alloy/Promtail) collects container logs and forwards to Loki | Add `grafana/alloy:v1.7.5` to docker-compose.yml mounting Docker socket. Configure `config.alloy` with `discovery.docker` + `discovery.relabel` + `loki.source.docker` pipeline. Extract `service` label from Docker Compose service name via `__meta_docker_container_label_com_docker_compose_service`. |
</phase_requirements>

---

## Summary

Phase 14 has three independent sub-problems: (1) ensuring structured JSON log output from go-zero services, (2) running Loki in Docker Compose, and (3) shipping container logs to Loki via Grafana Alloy.

For **LOG-01**, go-zero logx v1.10.0 already defaults `Encoding` to `json` — the current configs using `Mode: console` already produce JSON output. Explicitly adding `Encoding: json` to all docker YAML configs makes this an intentional, verifiable configuration rather than relying on a default. The JSON output format is `{"@timestamp":"...","level":"info","content":"...","caller":"...",<extra_fields>}`. No Go code changes are required.

For **LOG-02** and **LOG-03**, the standard 2025/2026 stack is Loki 3.x (single-binary, filesystem storage) + Grafana Alloy (replacing deprecated Promtail, which enters EOL March 2026). Alloy uses the Docker socket to discover all running containers and forwards logs to Loki with container-derived labels. The key label for `{service="url-api"}` style queries is extracted via `discovery.relabel` from the Docker Compose service name metadata label `__meta_docker_container_label_com_docker_compose_service`.

**Primary recommendation:** Add `Encoding: json` to all 6 YAML configs; add Loki + Alloy to docker-compose.yml; create `infra/loki/loki-config.yaml` and `infra/alloy/config.alloy`; wire Alloy to Loki using `loki.source.docker` pipeline with `service` label from Docker Compose metadata.

---

## Standard Stack

### Core

| Component | Version | Purpose | Why Standard |
|-----------|---------|---------|--------------|
| `grafana/loki` | 3.0.0 | Log aggregation backend; receives, indexes, and serves logs via LogQL | Industry-standard log backend for container environments; pairs natively with Grafana |
| `grafana/alloy` | v1.7.5 | Log shipper; discovers Docker containers and ships logs to Loki | Official successor to Promtail (deprecated Feb 2025, EOL March 2026); Grafana's recommended shipper |
| go-zero `logx` | bundled in go-zero v1.10.0 | Structured JSON logger; `Encoding: json` produces JSON lines to stdout | Already in use; `Encoding` field defaults to `json` — no new dependency |

### Supporting

| Component | Version | Purpose | When to Use |
|-----------|---------|---------|-------------|
| `loki-config.yaml` | — | Loki single-binary filesystem config | Required: auth_enabled=false, tsdb store, filesystem storage |
| `config.alloy` | — | Alloy River-language pipeline config | Required: Docker discovery + relabel + source + write pipeline |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Grafana Alloy | Promtail | Promtail is deprecated (Feb 2025), EOL March 2026 — do not use for new setups |
| Grafana Alloy | Loki Docker Driver | Docker logging driver is per-container config, harder to manage; Alloy is centralized |
| Loki single-binary filesystem | Loki microservices + MinIO | Microservices mode is for production scale; overkill for local dev |
| Loki 3.x | Loki 2.x | Loki 3.x is current stable; 2.x had different schema defaults |

---

## Architecture Patterns

### Recommended Project Structure

```
infra/
├── loki/
│   └── loki-config.yaml        # Loki single-binary filesystem config
└── alloy/
    └── config.alloy            # Alloy pipeline: Docker discovery → Loki

docker-compose.yml              # Add loki + alloy services

services/
├── url-api/etc/
│   ├── url.yaml               # Add Encoding: json
│   └── url-docker.yaml        # Add Encoding: json
├── analytics-rpc/etc/
│   ├── analytics.yaml         # Add Encoding: json
│   └── analytics-docker.yaml  # Add Encoding: json
└── analytics-consumer/etc/
    ├── consumer.yaml          # Add Encoding: json
    └── consumer-docker.yaml   # Add Encoding: json
```

### Pattern 1: go-zero logx JSON Configuration

**What:** Add `Encoding: json` to `Log:` block in all service YAML configs.

**Current state:** All docker YAMLs have `Log: Mode: console, Level: info` — `Encoding` is absent, which means it uses the default value of `json` per the LogConf struct. This already produces JSON. Making it explicit is the recommended practice.

**Why console mode + json encoding works for containers:** Docker captures stdout/stderr from containers. `Mode: console` writes to stdout. `Encoding: json` formats as JSON. This is the canonical container logging pattern — no file rotation, no volume mounts for logs needed.

**Example change (all 6 YAML configs):**
```yaml
# Before
Log:
  ServiceName: url-api
  Mode: console
  Level: info

# After
Log:
  ServiceName: url-api
  Mode: console
  Encoding: json
  Level: info
```

**JSON output format from go-zero logx v1.10.0:**
```json
{"@timestamp":"2026-02-22T10:30:09.962+08:00","caller":"logic/redirectlogic.go:42","content":"redirect","level":"info","code":"abc123xyz","span":"a1b2c3d4","trace":"e5f6a7b8c9d0e1f2"}
```

Field names: `@timestamp`, `level`, `content` (message), `caller`, plus any `logx.Field(...)` key-value pairs, plus OTel `trace` and `span` when context-propagated.

**Source:** Context7 query of `/zeromicro/go-zero` logx readme; WebSearch verified JSON field format from go-zero.dev documentation.
Confidence: HIGH

### Pattern 2: Loki Single-Binary Docker Compose Service

**What:** Run Loki in single-binary mode (the `all` target) with local filesystem storage.

**Key config fields:**
- `auth_enabled: false` — disables multi-tenancy (required for simple local dev)
- `store: tsdb` — current index format (schema v13, required for Loki 3.x)
- `object_store: filesystem` — no S3/MinIO required
- `http_listen_port: 3100` — standard Loki port

**`infra/loki/loki-config.yaml`:**
```yaml
auth_enabled: false

server:
  http_listen_port: 3100

common:
  ring:
    instance_addr: 127.0.0.1
    kvstore:
      store: inmemory
  replication_factor: 1
  path_prefix: /tmp/loki

schema_config:
  configs:
  - from: "2020-05-15"
    store: tsdb
    object_store: filesystem
    schema: v13
    index:
      prefix: index_
      period: 24h

storage_config:
  filesystem:
    directory: /tmp/loki/chunks
```

**docker-compose.yml addition:**
```yaml
  loki:
    image: grafana/loki:3.0.0
    ports:
      - "3100:3100"
    volumes:
      - ./infra/loki/loki-config.yaml:/etc/loki/config.yaml
      - loki-data:/tmp/loki
    command: -config.file=/etc/loki/config.yaml
    healthcheck:
      test: ["CMD-SHELL", "wget -q --tries=1 -O- http://localhost:3100/ready || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5
```

Add `loki-data:` to the `volumes:` block.

**Source:** Context7 query of `/websites/grafana_loki` configuration examples; verified against official Loki filesystem config example.
Confidence: HIGH

### Pattern 3: Grafana Alloy Docker Container Log Collection

**What:** Alloy runs as a container with Docker socket access, discovers all containers in the Compose project, attaches a `service` label from Docker Compose metadata, and ships logs to Loki.

**Key insight — extracting Docker Compose service name:** Docker Compose automatically adds labels to all containers it manages. The label `com.docker.compose.service` is present on every container and contains the service name (e.g., `url-api`, `analytics-rpc`, `analytics-consumer`). In Alloy's discovery pipeline, this maps to the meta-label `__meta_docker_container_label_com_docker_compose_service`. Using this as the source for a `service` label enables LogQL queries like `{service="url-api"}`.

**`infra/alloy/config.alloy`:**
```alloy
// Discover all Docker containers
discovery.docker "containers" {
  host             = "unix:///var/run/docker.sock"
  refresh_interval = "5s"
}

// Extract service label from Docker Compose metadata
discovery.relabel "containers" {
  targets = []

  rule {
    source_labels = ["__meta_docker_container_label_com_docker_compose_service"]
    target_label  = "service"
  }

  rule {
    source_labels = ["__meta_docker_container_name"]
    regex         = "/(.*)"
    target_label  = "container"
  }
}

// Collect logs from discovered containers
loki.source.docker "containers" {
  host             = "unix:///var/run/docker.sock"
  targets          = discovery.docker.containers.targets
  forward_to       = [loki.write.loki.receiver]
  relabel_rules    = discovery.relabel.containers.rules
  refresh_interval = "5s"
}

// Write logs to Loki
loki.write "loki" {
  endpoint {
    url = "http://loki:3100/loki/api/v1/push"
  }
}
```

**docker-compose.yml addition:**
```yaml
  alloy:
    image: grafana/alloy:v1.7.5
    ports:
      - "12345:12345"    # Alloy UI (optional but useful for debugging)
    volumes:
      - ./infra/alloy/config.alloy:/etc/alloy/config.alloy
      - /var/run/docker.sock:/var/run/docker.sock
    command: run --server.http.listen-addr=0.0.0.0:12345 --storage.path=/var/lib/alloy/data /etc/alloy/config.alloy
    depends_on:
      loki:
        condition: service_healthy
```

**Resulting LogQL queries:**
```logql
{service="url-api"}
{service="analytics-rpc"}
{service="analytics-consumer"}
{service=~"url-api|analytics-rpc|analytics-consumer"}
```

**Source:** Context7 query of `/grafana/loki` + `/websites/grafana_loki`; WebSearch verified via middlewaretechnologies.in article (Jan 2026) + Grafana official Alloy Docker monitoring docs.
Confidence: HIGH

### Anti-Patterns to Avoid

- **Using `Mode: file` in containers:** File mode writes to disk inside the container. Docker log collection via Alloy reads stdout/stderr. File-based logs inside a container require a shared volume mount — unnecessary complexity. Always use `Mode: console` in containerized services.
- **Using Promtail instead of Alloy:** Promtail is deprecated Feb 2025, EOL March 2026. All new setups should use Alloy.
- **Using Loki microservices mode locally:** The quick-start multi-service Loki setup (with MinIO, separate read/write/backend) is for production. Single-binary with filesystem storage is correct for local dev.
- **Not setting `auth_enabled: false`:** Without this, Loki 3.x requires a tenant ID header in all push/query requests. In local dev without multi-tenancy, this causes 401 errors.
- **Mounting Docker socket as read-only in Alloy:** `loki.source.docker` requires read access to the Docker daemon API (it calls Docker's streaming log API, not just reads files). The socket must be mounted read-write (default).

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Container log collection | Custom log forwarder reading container stdout | Grafana Alloy `loki.source.docker` | Handles Docker API streaming, reconnection, deduplication, backpressure |
| Log parsing/enrichment | Custom JSON parser in the pipeline | Alloy `loki.process` stage with `stage.json` | Built-in JSON field extraction for label promotion |
| Log storage | Custom log storage/query system | Grafana Loki | Mature, purpose-built, LogQL query language, Grafana integration |
| JSON log format | Custom log writer | go-zero logx with `Encoding: json` | Already in use; standardized JSON with OTel trace/span correlation |

**Key insight:** The entire log aggregation stack (JSON emit → ship → store → query) is configuration-only for this project. No new Go code is required.

---

## Common Pitfalls

### Pitfall 1: Loki schema_config `from` date must be in the past

**What goes wrong:** Loki refuses to start or silently ignores a schema config entry where `from` is in the future.
**Why it happens:** Loki uses the `from` date to determine which schema version applies. If the date is in the future, Loki has no active schema and fails to write.
**How to avoid:** Always use a historical date (e.g., `from: "2020-05-15"`) for the single-schema local dev config.
**Warning signs:** Loki starts but pushing logs returns 500 or schema-related errors.

### Pitfall 2: Alloy needs Docker socket from the host, not another container

**What goes wrong:** If the Docker socket is not mounted into the Alloy container, `loki.source.docker` cannot connect to the Docker daemon and silently collects no logs.
**Why it happens:** The Docker socket (`/var/run/docker.sock`) lives on the host, not inside other containers.
**How to avoid:** Mount `/var/run/docker.sock:/var/run/docker.sock` in the Alloy service definition.
**Warning signs:** Alloy starts without errors but Loki receives no log streams.

### Pitfall 3: `loki.source.docker` won't collect logs from containers started before Alloy

**What goes wrong:** Containers started before Alloy is running may not have their logs collected from the beginning.
**Why it happens:** `loki.source.docker` connects to the Docker streaming log API, which by default tails from the current position. Historical logs before Alloy starts may be missed.
**How to avoid:** Add `depends_on: loki: condition: service_healthy` to Alloy; the three application services should start after Alloy is ready. In practice, the application services restart after startup failures anyway.
**Warning signs:** First few log lines after `docker compose up` are missing in Loki.

### Pitfall 4: go-zero logx `Encoding` field silently defaults to `json`

**What goes wrong:** Not adding `Encoding: json` explicitly doesn't cause JSON logging to fail (JSON is the default), but the omission creates confusion about whether logs are actually JSON.
**Why it happens:** The `LogConf.Encoding` field has `json:",default=json"` — the default is `json` regardless of `Mode`.
**How to avoid:** Add `Encoding: json` explicitly to all YAML configs to make the intent clear and the configuration self-documenting.
**Warning signs:** Logs look like JSON but you're unsure if the `Encoding` field is actually being applied.

### Pitfall 5: `__meta_docker_container_label_com_docker_compose_service` is absent for non-Compose containers

**What goes wrong:** If any container in the Docker environment was not started by Docker Compose (e.g., standalone `docker run` containers), the Compose service label won't be present, causing the `service` label to be empty for those containers.
**Why it happens:** The `com.docker.compose.service` label is only injected by the Docker Compose runtime.
**How to avoid:** Add a fallback relabel rule using `__meta_docker_container_name` for containers without a Compose service label. For this project, all containers are Compose-managed, so this is not a concern.

### Pitfall 6: Loki `loki-data` volume permissions

**What goes wrong:** Loki fails to write to `/tmp/loki` if the Docker volume has incorrect ownership.
**Why it happens:** The Loki container runs as a non-root user in some versions; the named volume may be owned by root.
**How to avoid:** Use `/tmp/loki` as the path (world-writable) or add `user: "0"` to the Loki service definition in docker-compose.yml for local dev. Alternatively, set `path_prefix: /loki` with proper volume setup.
**Warning signs:** Loki starts but fails to write index or chunk files; logs show permission denied errors.

---

## Code Examples

### LOG-01: Adding Encoding to YAML configs

```yaml
# services/url-api/etc/url-docker.yaml (and url.yaml, analytics-docker.yaml, etc.)
Log:
  ServiceName: url-api
  Mode: console
  Encoding: json
  Level: info
```

This change is identical for all 6 YAML files (3 services × 2 environments), only `ServiceName` differs.

### LOG-02: Loki Docker Compose service

```yaml
# docker-compose.yml
  loki:
    image: grafana/loki:3.0.0
    ports:
      - "3100:3100"
    volumes:
      - ./infra/loki/loki-config.yaml:/etc/loki/config.yaml
      - loki-data:/tmp/loki
    command: -config.file=/etc/loki/config.yaml
    healthcheck:
      test: ["CMD-SHELL", "wget -q --tries=1 -O- http://localhost:3100/ready || exit 1"]
      interval: 10s
      timeout: 5s
      retries: 5
```

### LOG-03: Alloy Docker Compose service

```yaml
# docker-compose.yml
  alloy:
    image: grafana/alloy:v1.7.5
    ports:
      - "12345:12345"
    volumes:
      - ./infra/alloy/config.alloy:/etc/alloy/config.alloy
      - /var/run/docker.sock:/var/run/docker.sock
    command: run --server.http.listen-addr=0.0.0.0:12345 --storage.path=/var/lib/alloy/data /etc/alloy/config.alloy
    depends_on:
      loki:
        condition: service_healthy
```

### Loki single-binary filesystem config

```yaml
# infra/loki/loki-config.yaml
auth_enabled: false

server:
  http_listen_port: 3100

common:
  ring:
    instance_addr: 127.0.0.1
    kvstore:
      store: inmemory
  replication_factor: 1
  path_prefix: /tmp/loki

schema_config:
  configs:
  - from: "2020-05-15"
    store: tsdb
    object_store: filesystem
    schema: v13
    index:
      prefix: index_
      period: 24h

storage_config:
  filesystem:
    directory: /tmp/loki/chunks
```

### Alloy Docker log collection pipeline

```alloy
# infra/alloy/config.alloy
discovery.docker "containers" {
  host             = "unix:///var/run/docker.sock"
  refresh_interval = "5s"
}

discovery.relabel "containers" {
  targets = []

  rule {
    source_labels = ["__meta_docker_container_label_com_docker_compose_service"]
    target_label  = "service"
  }

  rule {
    source_labels = ["__meta_docker_container_name"]
    regex         = "/(.*)"
    target_label  = "container"
  }
}

loki.source.docker "containers" {
  host             = "unix:///var/run/docker.sock"
  targets          = discovery.docker.containers.targets
  forward_to       = [loki.write.loki.receiver]
  relabel_rules    = discovery.relabel.containers.rules
  refresh_interval = "5s"
}

loki.write "loki" {
  endpoint {
    url = "http://loki:3100/loki/api/v1/push"
  }
}
```

### Verifying Loki receives logs (manual check)

```bash
# Verify Loki is ready
curl http://localhost:3100/ready

# Query logs for url-api
curl -G http://localhost:3100/loki/api/v1/query_range \
  --data-urlencode 'query={service="url-api"}' \
  --data-urlencode 'start=0' \
  --data-urlencode 'limit=10'
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Promtail as log shipper | Grafana Alloy | Promtail deprecated Feb 2025, EOL March 2026 | Use Alloy for all new setups; Alloy config uses River language (.alloy files) |
| Loki 2.x BoltDB-shipper | Loki 3.x TSDB (schema v13) | Loki 3.0 (2024) | Schema config must use `store: tsdb` and `schema: v13`; BoltDB is removed in Loki 3.x |
| Loki separate index/chunk storage | `common.path_prefix` unified config | Loki 2.x+ | Simpler config via `common` block; `path_prefix` sets base for all storage paths |
| Jaeger exporter in OTel | OTLP (decided in Phase 13) | 2023 | Already handled; Loki has no interaction with tracing stack |

**Deprecated/outdated:**
- `grafana/promtail`: Deprecated Feb 2025, EOL March 2026. Do not use.
- Loki `schema: v11`, `v12`: Replaced by v13 in Loki 3.x. Use `schema: v13`.
- Loki `store: boltdb-shipper`: Removed in Loki 3.x. Use `store: tsdb`.

---

## Open Questions

1. **Should the local dev YAML configs (`url.yaml`, not `url-docker.yaml`) also get `Encoding: json`?**
   - What we know: Local dev uses `make run-*` which reads `etc/url.yaml`. Adding `Encoding: json` there too is consistent.
   - What's unclear: Local dev may benefit from `Encoding: plain` for readability during debugging.
   - Recommendation: Add `Encoding: json` to all 6 configs (both local and docker). Structured JSON is the standard for any log aggregation pipeline, and tools like `jq` make local JSON logs readable.

2. **Should the Alloy service collect logs from ALL containers (including postgres, kafka, jaeger) or only the three application services?**
   - What we know: By default, `discovery.docker` discovers all containers. The success criteria requires logs from the three application services to be queryable.
   - What's unclear: Whether non-app container logs add noise.
   - Recommendation: Collect all containers. The `{service="url-api"}` label filter in LogQL isolates app logs. No filtering needed in the Alloy config — let Loki indexing handle it. Simpler config is better.

3. **Phase 15 (Metrics) will add Grafana — should Phase 14 also add Grafana to Docker Compose?**
   - What we know: Phase 15 requires Grafana with auto-provisioned datasources including Loki. Phase 14's success criteria does not mention Grafana.
   - What's unclear: Whether adding Loki without Grafana is complete enough.
   - Recommendation: Phase 14 adds only Loki + Alloy. Phase 15 adds Grafana with datasource provisioning for Loki, Jaeger, and Prometheus. This matches the phase boundaries defined in REQUIREMENTS.md. Loki is queryable via curl or API even without Grafana.

---

## Sources

### Primary (HIGH confidence)

- `/zeromicro/go-zero` Context7 — logx LogConf struct with `Encoding` field, default `json`
- `/grafana/loki` Context7 — Alloy Docker log collection config (`loki.source.docker`, `discovery.docker`, `discovery.relabel`)
- `/websites/grafana_loki` Context7 — Loki filesystem storage configuration example (schema v13, tsdb, auth_enabled: false)
- `https://go-zero.dev/en/docs/tutorials/go-zero/configuration/log` — LogConf field documentation
- `https://grafana.com/docs/alloy/latest/reference/components/loki/loki.source.docker/` — loki.source.docker component reference

### Secondary (MEDIUM confidence)

- `https://middlewaretechnologies.in/2026/01/how-to-manage-container-logs-with-grafana-alloy-and-loki.html` — Complete working Docker Compose + Alloy config example (Jan 2026); verified versions Alloy v1.7.5, Loki 3.0.0
- `https://grafana.com/docs/alloy/latest/monitor/monitor-docker-containers/` — `__meta_docker_container_label_com_docker_compose_service` label for service name extraction
- WebSearch results confirming Promtail deprecation timeline (Feb 2025 deprecated, March 2026 EOL)

### Tertiary (LOW confidence)

- go-zero logx JSON field format (`{"@timestamp":...,"level":...,"content":...}`) — confirmed by multiple WebSearch results pointing to go-zero.dev documentation examples, but not directly verified via Context7 source code.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — Loki 3.x + Alloy is confirmed current standard; versions verified against real deployment examples from Jan 2026
- Architecture (log JSON format): HIGH — LogConf struct confirmed via Context7; JSON field names confirmed via multiple consistent WebSearch results
- Architecture (Alloy pipeline): HIGH — confirmed via Context7 loki.source.docker docs + official Alloy Docker monitoring guide
- Pitfalls: MEDIUM-HIGH — most from official docs; Loki volume permissions is LOW (from general experience)

**Research date:** 2026-02-22
**Valid until:** 2026-04-22 (stable stack — 60-day estimate; Alloy version may update but config API is stable)
