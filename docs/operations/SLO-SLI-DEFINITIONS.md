# SLO / SLI Definitions

> **Status:** ACTIVE
> **Owner:** Roberto (Infra agent)
> **Priority:** P0

---

## Terminology

| Term | Definition |
|------|-----------|
| **SLI** (Service Level Indicator) | A metric that measures service quality |
| **SLO** (Service Level Objective) | A target value for an SLI |
| **SLA** (Service Level Agreement) | A contract with users about SLOs (not needed yet) |
| **Error Budget** | How much failure is allowed before SLO is violated |

## Service Level Objectives

### API Availability

| SLI | Measurement | SLO | Error Budget (30 days) |
|-----|-------------|-----|----------------------|
| Uptime | `(successful requests / total requests) * 100` | 99.5% | 3.6 hours downtime |

### API Latency

| SLI | Measurement | SLO |
|-----|-------------|-----|
| p50 latency | Median response time | < 200ms |
| p95 latency | 95th percentile | < 500ms |
| p99 latency | 99th percentile | < 2000ms |

### App Generation Pipeline

| SLI | Measurement | SLO |
|-----|-------------|-----|
| Generation success rate | `(completed / attempted) * 100` | > 95% |
| Generation time | Time from request to completion | < 60 seconds |
| File persistence rate | `(files saved / files generated) * 100` | 100% |

### SSE Streaming

| SLI | Measurement | SLO |
|-----|-------------|-----|
| Connection success rate | SSE connections established / attempted | > 99% |
| Event delivery rate | Events received by client / events emitted | > 99% |
| Reconnection success rate | Auto-reconnects that succeed / attempted | > 95% |

### Frontend

| SLI | Measurement | SLO |
|-----|-------------|-----|
| Page load time | Time to interactive | < 3 seconds |
| Build success rate | CI builds that pass / total | > 99% |
| Error rate | JS errors per session | < 0.1% |

## Measurement

### How We Measure

| Metric | Source | Dashboard |
|--------|--------|-----------|
| API latency | Cloud Run built-in metrics | [GCP Cloud Run Console](https://console.cloud.google.com/run?project=miosa-460433) |
| Uptime | Cloud Run health checks (`/health` endpoint) | [GCP Cloud Run Console](https://console.cloud.google.com/run?project=miosa-460433) |
| Error rate | Sentry (after setup) + Cloud Run error logs | [GCP Logging](https://console.cloud.google.com/logs?project=miosa-460433) |
| Generation metrics | Application logs (structured slog output) | [GCP Logging](https://console.cloud.google.com/logs?project=miosa-460433) |

### Alerting

| Alert | Condition | Notification |
|-------|-----------|-------------|
| API down | Health check fails 2x consecutive | GitHub Issue (auto-created) + team chat |
| High error rate | >5% 5xx responses in 5 min window | GitHub Issue + team chat |
| High latency | p99 > 5s for 10 min | GitHub Issue + team chat |
| Generation failures | >3 failures in 1 hour | GitHub Issue + team chat |

**Alerting implementation plan:** GCP Cloud Monitoring will be configured to send alerts to a notification channel (team chat webhook + GitHub Issues via Cloud Functions). This will be set up as part of the Sprint 1 production deployment.

### Baseline Establishment

> **Important:** These SLOs are initial targets based on industry standards for similar systems. Baselines will be established after the Sprint 1 production deployment. Within 30 days of launch:
>
> 1. Collect real production data for all SLIs
> 2. Compare actual performance against targets
> 3. Adjust SLOs to be ambitious but achievable (within 2x of measured baseline)
> 4. Set error budgets based on adjusted SLOs

## Reviewing SLOs

- Review SLOs monthly for the first 3 months
- Adjust based on real data (baseline may be wrong)
- If consistently exceeding SLO: tighten it
- If consistently missing SLO: investigate and fix, or relax if unrealistic

---

**Last Updated:** 2026-02-23
