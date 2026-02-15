---
phase: 05-production-readiness
plan: 05
subsystem: ci-pipeline
tags: [github-actions, ci-cd, coverage, makefile, golangci-lint]
dependency-graph:
  requires: [docker-compose-dockerfiles, unit-tests, handler-tests]
  provides: [automated-ci-pipeline, coverage-enforcement]
  affects: [all-future-development]
tech-stack:
  added: [github-actions, go-test-coverage]
  patterns: [ci-pipeline, coverage-gates]
key-files:
  created:
    - .github/workflows/ci.yml
    - .coverage.yaml
  modified:
    - Makefile
decisions:
  - CI triggers on PRs and pushes to main/master only
  - Pipeline has lint -> test -> build flow with build depending on both
  - Coverage threshold enforced at 80% total, 70% per package
  - Generated sqlc code excluded from coverage (threshold 0)
  - cmd packages have reduced threshold (50%) due to wiring code
  - Docker images built in CI but not pushed (no registry configured)
  - go-test-coverage v2 used for threshold enforcement
metrics:
  tasks-completed: 2
  files-created: 2
  files-modified: 1
  completed: 2026-02-15
---

# Phase 05 Plan 05: GitHub Actions CI Pipeline Summary

GitHub Actions CI/CD pipeline with lint, test (with coverage enforcement), and build stages. Coverage thresholds configured via go-test-coverage. Makefile updated with local CI targets.

## What Was Built

### Task 1: GitHub Actions CI workflow

**.github/workflows/ci.yml:**

**Trigger:** PRs and pushes to main/master branches only.

**Job 1: lint**
- ubuntu-latest with Go 1.24
- Installs golangci-lint v2 via `go install`
- Runs with 5-minute timeout
- Uses `.golangci.yml` config from plan 05-04

**Job 2: test**
- ubuntu-latest with Go 1.24
- Runs all unit tests with race detection and coverage profiling
- Installs go-test-coverage v2 for threshold enforcement
- Reads `.coverage.yaml` for threshold configuration
- Uploads coverage artifact (7-day retention)

**Job 3: build**
- Depends on both lint and test (only runs if both pass)
- Builds both Go binaries with CGO_ENABLED=0 and ldflags optimization
- Sets up Docker Buildx for efficient image builds
- Builds both Docker images with GitHub Actions cache (gha)
- Images tagged with commit SHA but NOT pushed

**Commit:** 3aa0cae

---

### Task 2: Coverage configuration and Makefile CI targets

**.coverage.yaml:**
```yaml
threshold:
  total: 80      # Overall project coverage
  package: 70    # Per-package minimum

override:
  - path: sqlc dirs    → threshold: 0   # Generated code
  - path: cmd packages → threshold: 50  # Wiring code
```

**Makefile additions:**
- `make test` — Run all tests with race detection
- `make test-coverage` — Generate coverage profile and report
- `make lint` — Run golangci-lint
- `make ci` — Run lint, test-coverage, and build (full local CI)

**Commit:** (included in docs commit)

---

## Verification Results

```bash
# All tests pass
go test ./... -count=1
# ✅ 6 test packages pass

# CI workflow exists with valid structure
cat .github/workflows/ci.yml
# ✅ 3 jobs: lint, test, build

# Coverage config exists
cat .coverage.yaml
# ✅ threshold.total: 80

# Makefile has CI targets
make -n ci
# ✅ Runs lint, test-coverage, build
```

---

## Success Criteria

- ✅ GitHub Actions workflow exists with 3 jobs: lint, test, build
- ✅ CI triggers only on main/master per locked decision
- ✅ Coverage threshold enforced at 80% via go-test-coverage
- ✅ Docker images built in CI (not pushed)
- ✅ Makefile provides local CI workflow via `make ci`
- ✅ Generated sqlc code excluded from coverage threshold

---

## Files Created/Modified

### Created (2 files):
- `.github/workflows/ci.yml` (99 lines)
- `.coverage.yaml` (11 lines)

### Modified (1 file):
- `Makefile` — Added test, test-coverage, lint, ci targets
