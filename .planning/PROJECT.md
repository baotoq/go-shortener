# Go URL Shortener

## What This Is

A URL shortener built with Go and Dapr, structured as two microservices communicating via Dapr building blocks. This is a learning project to practice real-world Go patterns — clean architecture, distributed systems with Dapr, and production-grade engineering practices.

## Core Value

Shorten a long URL and reliably redirect anyone who visits the short link. Everything else (analytics, events) is secondary to this working correctly.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Shorten a URL via API and receive a short link back
- [ ] Redirect to original URL when visiting a short link
- [ ] Track click count per short link
- [ ] URL service and Analytics service communicate via Dapr pub/sub
- [ ] Services call each other via Dapr service invocation
- [ ] Clean architecture layers (handler → service → repository)
- [ ] SQLite for persistent storage
- [ ] Docker setup for running services with Dapr sidecars
- [ ] Tests for all layers
- [ ] CI pipeline

### Out of Scope

- Web UI / frontend — API-only project
- User accounts / authentication — not needed for learning goals
- Custom aliases — keeping short codes auto-generated for v1
- Link expiration — adds complexity without teaching new Go/Dapr patterns
- Dapr state store — using SQLite directly to learn both raw DB access and Dapr separately

## Context

- Learning project: the goal is to practice Go patterns and Dapr, not ship a product
- Two services: **URL Service** (shorten + redirect) and **Analytics Service** (click tracking + stats)
- Dapr provides the glue: pub/sub for event-driven click tracking, service invocation for cross-service calls
- SQLite keeps infrastructure simple — no external DB server to manage
- Clean architecture ensures each layer is testable and swappable

## Constraints

- **Language**: Go — the entire point of the project
- **Runtime**: Dapr — learning distributed application patterns
- **Storage**: SQLite — zero-dependency, file-based database
- **Architecture**: Clean architecture with handler/service/repository layers
- **Deployment**: Docker Compose to orchestrate services + Dapr sidecars

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 2 services (URL + Analytics) | Clear separation of concerns, natural pub/sub boundary | — Pending |
| SQLite over Dapr state store | Learn raw DB access alongside Dapr; keep storage simple | — Pending |
| Dapr for inter-service communication | Learn distributed patterns (pub/sub, service invocation) | — Pending |
| Clean architecture | Learn production Go patterns (layered, testable, swappable) | — Pending |

---
*Last updated: 2026-02-14 after initialization*
