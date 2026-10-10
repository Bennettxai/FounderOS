# Load Testing Plan

> **Status:** PLANNED (Sprint 5 deliverable)
> **Owner:** Pedro
> **Priority:** P2

---

## Target Metrics

| Metric | Target | Rationale |
|--------|--------|-----------|
| Concurrent users | 100 | Expected user base at launch + 2x headroom |
| API throughput | 50 RPS | Covers 100 users with burst activity |
| Concurrent app generations | 10 | LLM-bound — each generation takes 15-60s |
| API p99 latency | < 2 seconds | SLO target (see [SLO-SLI-DEFINITIONS.md](../operations/SLO-SLI-DEFINITIONS.md)) |
| SSE connection duration | 5 minutes sustained | App generation streaming window |
| Error rate under load | < 1% | Non-generation endpoints |

## Tool

**k6** — already configured in the CI pipeline (`performance-tests.yml`).

k6 is an open-source load testing tool written in Go. It runs JavaScript test scripts and produces detailed metrics.

```bash
# Install k6
brew install k6  # macOS
# or: snap install k6  # Linux

# Run a test
k6 run test-script.js
```

## Test Scenarios

### 1. API Load Test (Standard)

Simulates normal usage patterns:

| Phase | Duration | Virtual Users | Target |
|-------|----------|--------------|--------|
| Ramp up | 2 min | 0 → 50 | Gradual load increase |
| Sustained | 5 min | 50 | Steady state |
| Peak | 2 min | 50 → 100 | Peak load |
| Cool down | 1 min | 100 → 0 | Graceful drain |

**Endpoints tested:**
- `GET /health` — health check baseline
- `POST /api/auth/login` — authentication
- `GET /api/workspaces` — workspace listing
- `POST /api/chat/v2/message` — chat message (core flow)
- `GET /api/workspaces/:id/apps` — app listing

### 2. App Generation Pipeline

Simulates concurrent app generation requests:

| Phase | Duration | Virtual Users | Target |
|-------|----------|--------------|--------|
| Ramp up | 1 min | 0 → 10 | Gradual generation requests |
| Sustained | 10 min | 10 | 10 concurrent generations |
| Measure | Continuous | — | Track completion time and success rate |

**Endpoints tested:**
- `POST /api/workspaces/:id/apps/generate-osa` — trigger generation
- `GET /api/osa/apps/generate/:id/stream` — SSE stream monitoring

### 3. SSE Streaming

Simulates many users watching app generation progress:

| Phase | Duration | Connections | Target |
|-------|----------|------------|--------|
| Connect | 1 min | 0 → 50 | Open SSE connections |
| Sustained | 5 min | 50 | Hold connections, receive events |
| Verify | Continuous | — | No dropped connections, all events received |

### 4. Spike Test

Simulates sudden traffic burst:

| Phase | Duration | Virtual Users | Target |
|-------|----------|--------------|--------|
| Normal | 2 min | 20 | Baseline |
| Spike | 30 sec | 20 → 200 | 10x sudden increase |
| Recover | 2 min | 200 → 20 | Recovery to normal |
| Verify | 1 min | 20 | No lingering effects |

**Success criteria:** Error rate < 5% during spike, full recovery within 2 minutes.

### 5. Endurance Test

Simulates sustained load over hours:

| Phase | Duration | Virtual Users | Target |
|-------|----------|--------------|--------|
| Ramp up | 5 min | 0 → 30 | Gradual increase |
| Sustained | 2 hours | 30 | Long-running steady state |
| Monitor | Continuous | — | Memory leaks, connection leaks, degradation |

**Success criteria:** No memory growth >20%, no latency degradation >10% over the test duration.

## Baseline Establishment

> **Timeline:** Measure after Sprint 1 production deployment.

1. Run the Standard API Load Test against production
2. Record p50, p95, p99 latency for each endpoint
3. Record throughput (RPS) at each phase
4. Record error rate at each phase
5. Store results as baseline for regression comparison

The CI pipeline (`performance-tests.yml`) already supports baseline comparison with a 20% degradation threshold.

## Integration with CI

Performance tests run automatically:
- **Weekly:** Sunday 2 AM UTC (all suites)
- **On release branches:** push to `release/*`
- **Manual:** via `workflow_dispatch` with suite selection

Results are stored as artifacts (90 days) and compared against baselines. A GitHub Issue is auto-created if regression exceeds 20%.

---

**Last Updated:** 2026-02-23
