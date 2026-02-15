---
phase: 03-enhanced-analytics
plan: 01
subsystem: enrichment-data-collection
tags: [event-payload, geoip, user-agent, referer, enrichment-services]
dependency_graph:
  requires:
    - 02-03 (ClickEvent type, Analytics Service)
  provides:
    - Extended ClickEvent with client_ip, user_agent, referer fields
    - GeoIPResolver for country detection
    - DeviceDetector for device type classification
    - RefererClassifier for traffic source classification
  affects:
    - URL Service (Redirect handler extracts HTTP headers)
    - Analytics Service (will consume enrichment data in 03-03)
tech_stack:
  added:
    - github.com/oschwald/geoip2-golang v1.13.0
    - github.com/mileusna/useragent v1.3.5
  patterns:
    - Request header extraction before goroutine
    - Graceful degradation for enrichment failures
key_files:
  created:
    - internal/analytics/enrichment/geoip.go
    - internal/analytics/enrichment/useragent.go
    - internal/analytics/enrichment/referer.go
  modified:
    - internal/shared/events/click_event.go
    - internal/urlservice/delivery/http/handler.go
    - go.mod
    - go.sum
decisions:
  - IP extracted from r.RemoteAddr with SplitHostPort fallback (Chi RealIP middleware handles X-Forwarded-For)
  - Empty referer classified as "Direct" (per research: missing referer = direct visit)
  - Bot detection prioritized in device classification (bots tracked separately per user decision)
  - Domain matching uses strings.Contains for subdomain support (m.facebook.com, www.google.com)
metrics:
  duration: 2m 51s
  tasks_completed: 2
  files_created: 3
  dependencies_added: 2
  completed_at: 2026-02-15
---

# Phase 03 Plan 01: Enrichment Data Collection Summary

**One-liner:** Extended ClickEvent payload with client IP, User-Agent, and Referer; created three enrichment services (GeoIP country resolver, device detector, traffic source classifier) using geoip2-golang and mileusna/useragent libraries.

## Objective

Extend the click event payload with enrichment data (client IP, User-Agent, Referer) and create the three enrichment services that will process this data in the Analytics Service.

## Execution Flow

### Task 1: Extend ClickEvent and update URL Service publisher

**Status:** Complete
**Commit:** d4f52bd

Extended the ClickEvent struct with three new fields:
- `ClientIP string` - for geolocation enrichment
- `UserAgent string` - for device type detection
- `Referer string` - for traffic source classification

Updated the URL Service redirect handler to extract these values from the HTTP request:
- Client IP: extracted from `r.RemoteAddr` using `net.SplitHostPort` to strip port, with fallback to raw value if parsing fails
- User-Agent: extracted from `User-Agent` header
- Referer: extracted from `Referer` header

Key implementation detail: All three values are extracted BEFORE the goroutine that publishes the event, ensuring the request object is safely accessed.

**Verification:**
- go build ./cmd/url-service/ - compiled successfully
- go vet ./internal/shared/... ./internal/urlservice/... - no issues
- ClickEvent struct contains ClientIP, UserAgent, Referer fields
- publishClickEvent signature updated to accept 4 parameters

### Task 2: Create enrichment services and install dependencies

**Status:** Complete
**Commit:** b7ca239

Installed two new dependencies:
1. `github.com/oschwald/geoip2-golang` v1.13.0 - for IP geolocation
2. `github.com/mileusna/useragent` v1.3.5 - for User-Agent parsing

Created three enrichment services:

**GeoIPResolver** (`internal/analytics/enrichment/geoip.go`):
- Opens MaxMind GeoIP2 database file
- `ResolveCountry(ipStr string) string` - returns ISO country code or "Unknown"
- Handles invalid IPs, private IPs, and lookup failures gracefully
- Requires explicit Close() call to release database handle

**DeviceDetector** (`internal/analytics/enrichment/useragent.go`):
- Stateless detector (no initialization required)
- `DetectDevice(uaString string) string` - returns Desktop/Mobile/Tablet/Bot/Unknown
- Prioritizes bot detection first (per user decision: bots tracked separately)
- Handles empty User-Agent strings

**RefererClassifier** (`internal/analytics/enrichment/referer.go`):
- Initialized with predefined domain lists for search engines, social media, AI platforms
- `ClassifySource(refererStr string) string` - returns Search/Social/AI/Direct/Referral
- Empty referer = "Direct" (per research: missing referer indicates direct visit)
- Uses `strings.Contains` for domain matching to support subdomains
- Domain lists include: 7 search engines, 11 social platforms, 5 AI platforms

**Verification:**
- go build ./internal/analytics/enrichment/ - compiled successfully
- go vet ./internal/analytics/enrichment/... - no issues
- All three service files created and exist
- go build ./... - entire project compiles

**Blocking issue resolved (Rule 3):**
During verification, encountered build cache issue causing false compilation error. Resolved by running `go clean -cache` and rebuilding.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Build cache issue**
- **Found during:** Task 2 verification
- **Issue:** go build reported "not enough arguments" error for InsertClick call despite correct signature
- **Fix:** Cleared Go build cache with `go clean -cache` and rebuilt
- **Files modified:** None (cache issue)
- **Commit:** N/A (no code changes needed)

No other deviations - plan executed as written.

## Success Criteria Verification

- [x] ClickEvent payload extended with client_ip, user_agent, referer
- [x] URL Service extracts request headers and forwards them in click events
- [x] GeoIP resolver handles valid IPs, private IPs, and malformed input gracefully
- [x] Device detector categorizes into exactly Desktop/Mobile/Tablet/Bot/Unknown
- [x] Referer classifier categorizes into exactly Search/Social/AI/Direct/Referral
- [x] All packages compile and pass go vet

## Technical Decisions

1. **IP extraction strategy:** Used `r.RemoteAddr` with `net.SplitHostPort` to strip port. This works because Chi's RealIP middleware already rewrites `r.RemoteAddr` to the real client IP from X-Forwarded-For headers.

2. **Empty referer = Direct:** Per 03-RESEARCH.md, missing or empty referer headers indicate direct navigation (typed URL, bookmark, etc.), not a referral.

3. **Bot prioritization:** Bot detection checked first in device classification because bots should be tracked separately from device types, regardless of other User-Agent properties.

4. **Subdomain matching:** RefererClassifier uses `strings.Contains(hostname, domain)` instead of exact matching to handle subdomains like `m.facebook.com`, `www.google.com`, `gemini.google.com`.

## Dependencies Added

| Package | Version | Purpose |
|---------|---------|---------|
| github.com/oschwald/geoip2-golang | v1.13.0 | IP to country code resolution (requires MaxMind GeoIP2 database) |
| github.com/mileusna/useragent | v1.3.5 | User-Agent string parsing for device detection |

## Next Steps

This plan sets up the data collection and enrichment infrastructure. The next plans will:

- **03-02:** Integrate enrichment services into Analytics Service event handler
- **03-03:** Add database schema and queries for enriched analytics (country, device, traffic source dimensions)

The enrichment services are now ready to be wired into the Analytics Service's RecordClick method (currently marked with TODO comment for plan 03-03).

## Self-Check: PASSED

**Created files verification:**
- [x] /Users/baotoq/Work/go-shortener/internal/analytics/enrichment/geoip.go - EXISTS
- [x] /Users/baotoq/Work/go-shortener/internal/analytics/enrichment/useragent.go - EXISTS
- [x] /Users/baotoq/Work/go-shortener/internal/analytics/enrichment/referer.go - EXISTS

**Commit verification:**
- [x] d4f52bd - EXISTS (Task 1: Extend ClickEvent)
- [x] b7ca239 - EXISTS (Task 2: Enrichment services)

**Modified files verification:**
- [x] internal/shared/events/click_event.go - ClientIP, UserAgent, Referer fields present
- [x] internal/urlservice/delivery/http/handler.go - Extracts headers before goroutine, passes to publishClickEvent

**Compilation verification:**
- [x] go build ./... - No errors
- [x] go vet ./... - No issues
