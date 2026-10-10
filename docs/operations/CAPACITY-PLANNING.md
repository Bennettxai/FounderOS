# Capacity Planning

> **Status:** ACTIVE
> **Owner:** Roberto
> **Priority:** P2

---

## Current Baseline

| Resource | Current Spec | Provider | Cost |
|----------|-------------|----------|------|
| Backend | Cloud Run — min 1, max 10 instances, 1 vCPU, 1GB RAM | GCP | ~$5-20/mo |
| Frontend | Vercel — free tier or Cloud Run | Vercel/GCP | $0-20/mo |
| Database | Cloud SQL — db-f1-micro, PostgreSQL 15 + pgvector | GCP (Supabase) | ~$10-15/mo |
| Redis | Managed instance or Memorystore — basic | GCP | ~$5-10/mo |
| LLM API | Anthropic Claude — pay-per-token | Anthropic | Variable |
| **Total** | | | **~$20-65/mo** |

## Scaling Thresholds

### Cloud Run Backend

| Metric | Current Limit | Scale Action | Target |
|--------|--------------|--------------|--------|
| CPU utilization | >70% sustained (5 min) | Increase max instances | Max 10 → 20 |
| Memory utilization | >80% of 1GB | Increase memory per instance | 1GB → 2GB |
| Request latency p99 | >2 seconds | Add instances or optimize code | <2s p99 |
| Cold start frequency | >10% of requests | Set min instances to 2+ | <5% cold starts |
| Concurrent requests | >80 per instance | Increase max instances | Headroom for bursts |

### Database (Cloud SQL)

| Metric | Current Limit | Scale Action | Target |
|--------|--------------|--------------|--------|
| Active connections | >80% of max (25 for micro) | Upgrade tier to db-g1-small | <70% utilization |
| Storage usage | >80% of allocated | Increase storage or enable auto-grow | >20% free |
| CPU utilization | >70% sustained | Upgrade to db-custom-2-4096 | <60% sustained |
| Slow queries | >100ms average | Add indexes, optimize queries | <50ms average |
| Replication lag | >1 second (if read replicas) | Investigate replication bottleneck | <100ms lag |

### Redis

| Metric | Current Limit | Scale Action | Target |
|--------|--------------|--------------|--------|
| Memory usage | >80% of allocated | Increase instance size | <70% utilization |
| Eviction rate | Any evictions occurring | Increase memory or reduce TTLs | Zero evictions |
| Connection count | >80% of max | Review connection pooling | <60% of max |

## Cost Projections

| Scale | Users | Estimated Monthly Cost | Key Changes |
|-------|-------|----------------------|-------------|
| **Current** | 1-10 (dev team) | $20-65 | Free tiers, minimal usage |
| **Launch** | 10-100 | $50-150 | Cloud SQL upgrade to db-g1-small, 2+ Cloud Run instances |
| **Growth** | 100-1,000 | $200-500 | db-custom-2-4096, 5+ instances, Redis upgrade, higher LLM usage |
| **Scale** | 1,000-10,000 | $500-2,000 | Read replicas, connection pooling (PgBouncer), CDN, multiple regions |

**Note:** LLM API costs are the largest variable. At 1,000 active users generating apps:
- Estimated 5,000 LLM calls/day × ~$0.03/call = ~$150/day = ~$4,500/month for LLM alone
- Optimize with: response caching, shorter prompts, model tiering (use cheaper models for simple tasks)

## Growth Signals to Monitor

These metrics indicate when scaling action is needed:

| Signal | Where to Check | Action Threshold |
|--------|---------------|-----------------|
| DB connection count | `SELECT count(*) FROM pg_stat_activity;` | >20 active connections |
| Cloud Run instance count | GCP Console → Cloud Run → Metrics | Consistently at max instances |
| Cloud Run CPU | GCP Console → Cloud Run → Metrics | >70% for 15+ minutes |
| Memory pressure | GCP Console → Cloud Run → Metrics | >80% utilization |
| Request queue depth | Cloud Run request latency spike | p99 >5s |
| LLM API spend | Anthropic dashboard | >$500/month (review optimization) |
| Storage growth rate | Cloud SQL → Storage metrics | >1GB/month growth |

## Scaling Playbook

### Phase 1: Optimize First (0-100 users)

Before scaling infrastructure, optimize the application:
- Add database indexes for frequent queries
- Enable query result caching in Redis
- Optimize LLM prompts to reduce token usage
- Add connection pooling to Go backend (already configured: 25 max open, 10 idle)

### Phase 2: Vertical Scale (100-1,000 users)

- Upgrade Cloud SQL: `db-f1-micro` → `db-g1-small` → `db-custom-2-4096`
- Increase Cloud Run memory: 1GB → 2GB
- Upgrade Redis instance size
- Enable Cloud SQL automated backups with point-in-time recovery

### Phase 3: Horizontal Scale (1,000+ users)

- Add Cloud SQL read replicas for read-heavy queries
- Deploy PgBouncer for connection pooling
- Add CDN for frontend static assets
- Consider multi-region deployment for latency
- Implement request queuing for app generation pipeline

---

**Last Updated:** 2026-02-23
