# CI Gates

> **Status:** ACTIVE
> **Owner:** Roberto
> **Priority:** P1

---

## Overview

BusinessOS uses 9 GitHub Actions workflows to gate code quality, security, and deployment. All workflows are in `.github/workflows/`.

## Workflow Summary

| Workflow | File | Trigger | Required for Merge |
|----------|------|---------|-------------------|
| Backend Unit Tests | `backend-tests.yml` | PR + push (backend changes) | Yes |
| Backend Integration Tests | `backend-integration-tests.yml` | PR + push (backend changes) | Yes |
| Frontend Tests | `frontend-tests.yml` | PR + push (frontend changes) | Yes |
| Security Tests | `security-tests.yml` | PR + push + weekly schedule | Yes |
| Security Scan | `security-scan.yml` | PR + push + weekly schedule | Informational |
| E2E Tests | `e2e-tests.yml` | PR + push (main/release) | Yes |
| Performance Tests | `performance-tests.yml` | Weekly + release branches | Informational |
| Deploy Backend | `deploy-backend.yml` | Push to main only | N/A (deploy) |
| Deploy Frontend | `deploy-frontend.yml` | Push to main only | N/A (deploy) |

## Required Gates (Must Pass Before Merge)

### Backend Unit Tests (`backend-tests.yml`)

- **Trigger:** PRs and pushes to main/dev branches when `desktop/backend-go/**` changes
- **What runs:** `go test -short -race ./...`
- **Coverage threshold:** 40% (temporary, will increase to 70%)
- **Key feature:** Race detection enabled

### Backend Integration Tests (`backend-integration-tests.yml`)

- **Trigger:** Same as unit tests + manual trigger
- **What runs:** Full test suite against PostgreSQL (pgvector) + Redis
- **Coverage threshold:** 60% (informational, will become blocking)
- **Timeout:** 20 minutes
- **Post-run:** Comments on PR with test results and coverage

### Frontend Tests (`frontend-tests.yml`)

- **Trigger:** PRs and pushes when `frontend/**` changes
- **What runs:**
  1. TypeScript check (`npm run check`) — informational
  2. ESLint — informational
  3. Vitest tests (`npm test`) — **blocking**
  4. SvelteKit build (`npm run build`) — **blocking**
- **Artifacts:** Build output retained for 7 days

### Security Tests (`security-tests.yml`)

- **Trigger:** PRs, pushes, weekly schedule (Monday 6 AM UTC)
- **What runs:**
  1. Custom security test suite (SQL injection, XSS, JWT, CSRF, privilege escalation, rate limiting, security headers)
  2. GoSec static analysis → SARIF upload
  3. Nancy dependency vulnerability scan
- **Timeout:** 15 minutes

### E2E Tests (`e2e-tests.yml`)

- **Trigger:** Pushes and PRs to main/release branches
- **What runs:** Playwright E2E tests against full stack (backend + frontend + DB)
- **Setup:** Starts backend on port 8001, waits for health check (30 attempts)
- **Timeout:** 60 minutes
- **Artifacts:** HTML report (30 days), test videos on failure (7 days)

## Informational Gates (Don't Block Merge)

### Security Scan (`security-scan.yml`)

- **Trigger:** PRs, pushes, weekly schedule (Monday 2 AM UTC)
- **What runs:**
  1. `govulncheck` (Go vulnerability check)
  2. `npm audit` (frontend dependencies, moderate+)
  3. Trivy container scan (CRITICAL + HIGH)
  4. Snyk scan (if token available)
- **Output:** SARIF results uploaded to GitHub Security tab

### Performance Tests (`performance-tests.yml`)

- **Trigger:** Weekly (Sunday 2 AM UTC), release branches, manual
- **What runs:** k6 load tests (OSA, hybrid, spike, endurance)
- **Baseline comparison:** Flags >20% degradation
- **On failure:** Creates GitHub Issue with action items
- **Timeout:** 180 minutes (3 hours)

## Deployment Workflows

### Deploy Backend (`deploy-backend.yml`)

- **Trigger:** Push to `main` only + manual
- **Steps:** Tests → Docker build → Push to GCR → Deploy to Cloud Run → Smoke tests
- **Rollback:** Automatic rollback to previous revision if smoke tests fail
- **Config:** Min 1, max 10 instances, 1GB RAM
- **Required secrets:** `GCP_SA_KEY`

### Deploy Frontend (`deploy-frontend.yml`)

- **Trigger:** Push to `main` only + manual
- **Steps:** TypeScript check (strict) → Build → Vercel deploy → Smoke tests
- **Required secrets:** `VERCEL_TOKEN`

## Branch Protection Rules

| Rule | Setting |
|------|---------|
| Required status checks | backend-tests, frontend-tests, security-tests |
| Require PR reviews | 1 approval minimum |
| Dismiss stale reviews | Yes (on new push) |
| Require up-to-date branch | Yes (must be current with main) |

## Common CI Failures and Fixes

| Failure | Cause | Fix |
|---------|-------|-----|
| `go test` timeout | Slow integration test or deadlock | Check for missing `t.Parallel()`, increase timeout, or debug specific test |
| `npm run build` fails | TypeScript error or missing import | Check svelte-check output, fix type errors |
| Security test fails | New endpoint missing auth check | Add authorization middleware to new endpoint |
| E2E backend won't start | Port 8001 already in use or migration fails | Check CI logs for startup errors, verify migrations |
| Docker build fails | Missing dependency or changed Dockerfile | Check Dockerfile and go.sum are in sync |

---

**Last Updated:** 2026-02-23
