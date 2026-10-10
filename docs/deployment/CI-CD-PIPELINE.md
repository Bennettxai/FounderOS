# CI/CD Pipeline

> **Status:** ACTIVE
> **Owner:** Roberto
> **Priority:** P1

---

## Overview

BusinessOS uses GitHub Actions for continuous integration and deployment. There are 9 workflows organized into three categories: testing, security, and deployment.

## Pipeline Architecture

```
                    ┌─────────────────┐
                    │   Pull Request   │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼              ▼              ▼
     ┌──────────────┐ ┌──────────────┐ ┌──────────────┐
     │ Backend Tests │ │Frontend Tests│ │Security Tests│
     │  Unit + Int.  │ │ Build+Check  │ │ Custom Suite │
     └──────┬───────┘ └──────┬───────┘ └──────┬───────┘
              │              │              │
              └──────────────┼──────────────┘
                             ▼
                    ┌─────────────────┐
                    │   E2E Tests     │
                    │  (Playwright)   │
                    └────────┬────────┘
                             │
                             ▼
                    ┌─────────────────┐
                    │  Merge to main  │
                    └────────┬────────┘
                             │
              ┌──────────────┼──────────────┐
              ▼                             ▼
     ┌──────────────┐             ┌──────────────┐
     │Deploy Backend│             │Deploy Frontend│
     │  Cloud Run   │             │    Vercel     │
     └──────┬───────┘             └──────┬───────┘
              │                          │
              ▼                          ▼
     ┌──────────────┐             ┌──────────────┐
     │ Smoke Tests  │             │ Smoke Tests  │
     │ (auto-rollback│             │              │
     │  on failure)  │             │              │
     └──────────────┘             └──────────────┘
```

## Workflows

### Testing Workflows

#### 1. Backend Unit Tests (`backend-tests.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | PR + push to main/dev branches |
| **Path filter** | `desktop/backend-go/**` |
| **Runs** | `go test -short -race ./...` |
| **Coverage** | 40% threshold (to be increased) |
| **Output** | Coverage report uploaded to Codecov |

#### 2. Backend Integration Tests (`backend-integration-tests.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | PR + push + manual |
| **Path filter** | `desktop/backend-go/**` |
| **Services** | PostgreSQL (pgvector), Redis |
| **Runs** | `go test ./...` (full suite, no `-short`) |
| **Coverage** | 60% threshold (informational) |
| **Timeout** | 20 minutes |
| **Output** | PR comment with test results |

#### 3. Frontend Tests (`frontend-tests.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | PR + push to main/dev branches |
| **Path filter** | `frontend/**` |
| **Runs** | TypeScript check → ESLint → Vitest → SvelteKit build |
| **Blocking** | Build failure and test failure block merge |
| **Artifacts** | Build output (7 days) |

#### 4. E2E Tests (`e2e-tests.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | Push + PR to main/release branches |
| **Stack** | Full: Go backend + PostgreSQL + SvelteKit + Playwright |
| **Setup** | Backend on port 8001, health check with 30 retries |
| **Timeout** | 60 minutes |
| **Artifacts** | HTML report (30 days), failure videos (7 days) |

### Security Workflows

#### 5. Security Tests (`security-tests.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | PR + push + weekly (Monday 6 AM UTC) |
| **Runs** | Custom security suite (8 categories), GoSec, Nancy |
| **Categories** | SQL injection, XSS, JWT auth, CSRF, privilege escalation, rate limiting, sensitive data, security headers |
| **Output** | SARIF results → GitHub Security tab |
| **Timeout** | 15 minutes |

#### 6. Security Scan (`security-scan.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | PR + push + weekly (Monday 2 AM UTC) |
| **Runs** | govulncheck, npm audit, Trivy (Docker), Snyk (optional) |
| **Severity** | CRITICAL + HIGH only (Trivy) |
| **Output** | SARIF results → GitHub Security tab |

### Performance Workflow

#### 7. Performance Tests (`performance-tests.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | Weekly (Sunday 2 AM UTC), release branches, manual |
| **Tool** | k6 load testing |
| **Suites** | OSA load, hybrid architecture, spike, endurance |
| **Baseline** | Flags >20% regression |
| **On failure** | Creates GitHub Issue with action items |
| **Timeout** | 180 minutes |

### Deployment Workflows

#### 8. Deploy Backend (`deploy-backend.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | Push to `main` only, manual |
| **Path filter** | `desktop/backend-go/**` |
| **Steps** | Test → Docker build → Push to GCR → Deploy to Cloud Run → Smoke test |
| **Cloud Run config** | Min 1, max 10 instances, 1GB RAM |
| **Rollback** | Automatic on smoke test failure |
| **Required secrets** | `GCP_SA_KEY` |

#### 9. Deploy Frontend (`deploy-frontend.yml`)

| Property | Value |
|----------|-------|
| **Triggers** | Push to `main` only, manual |
| **Path filter** | `frontend/**` |
| **Steps** | TypeScript check (strict) → Build → Vercel deploy → Smoke test |
| **Required secrets** | `VERCEL_TOKEN` |

## Required Secrets

| Secret | Used By | Purpose |
|--------|---------|---------|
| `GCP_SA_KEY` | deploy-backend | GCP authentication for Cloud Run deployment |
| `VERCEL_TOKEN` | deploy-frontend | Vercel deployment authentication |
| `CODECOV_TOKEN` | backend-tests | Coverage report upload |
| `SNYK_TOKEN` | security-scan | Snyk vulnerability scanning (optional) |

## Troubleshooting CI Failures

| Issue | Cause | Resolution |
|-------|-------|------------|
| Backend test timeout | Integration test waiting on DB | Check PostgreSQL service health, increase timeout |
| Frontend build fails | TypeScript or Svelte compilation error | Run `npm run check` locally, fix errors |
| Deploy rollback triggered | Smoke test failed post-deploy | Check Cloud Run logs, fix and re-push |
| E2E backend won't start | Port conflict or migration failure | Check CI logs, verify migration SQL |
| Security scan finds CVE | Vulnerable dependency | Update dependency, or add to ignore list if false positive |
| Performance regression | Code change degrades performance | Review recent changes, profile, fix before next release |

## Adding a New Workflow

1. Create `.github/workflows/new-workflow.yml`
2. Follow existing naming conventions
3. Add path filters to avoid unnecessary runs
4. Set appropriate timeout
5. Add to branch protection rules if it should block merges
6. Document in this file

---

**Last Updated:** 2026-02-23
