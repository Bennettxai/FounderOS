# Incident Response Playbook

> **Status:** ACTIVE
> **Owner:** Roberto
> **Priority:** P0

---

## Severity Levels

| Level | Definition | Response Time | Examples |
|-------|-----------|---------------|---------|
| **SEV-1** | Service down, all users affected | 15 min | Backend crash, DB down, auth broken |
| **SEV-2** | Major feature broken, many users affected | 1 hour | App generation fails, SSE broken |
| **SEV-3** | Minor feature broken, workaround exists | 4 hours | UI glitch, slow queries |
| **SEV-4** | Cosmetic or minor inconvenience | Next business day | Typo, styling issue |

## On-Call Rotation

| Role | Primary | Backup |
|------|---------|--------|
| Backend / Infrastructure | Roberto | Pedro |
| Backend / Services | Pedro | Roberto |
| Frontend | Javaris | Pedro |

**Rotation schedule:** Weekly rotation for primary on-call. The primary for each domain is the default responder; backup steps in when primary is unavailable. Review and adjust quarterly.

## Escalation Matrix

```
SEV-1: Notify ALL team → Roberto (infra) + Pedro (backend) + Javaris (frontend)
       Response within 15 minutes. All hands on deck.
SEV-2: Notify owner → Relevant operator based on component
       Response within 1 hour. Owner drives resolution.
SEV-3: Create issue → Assign to next sprint
       Response within 4 hours. Fix in normal workflow.
SEV-4: Create issue → Backlog
       Next business day. Low priority.
```

## Incident Response Steps

### 1. Detect
- Cloud Run health check fails
- Sentry alert fires
- User reports issue
- Monitoring dashboard shows anomaly

### 2. Triage
- Determine severity level
- Identify affected component (backend/frontend/DB/infra)
- Check: Is this a new deployment? (rollback candidate)

### 3. Communicate
- **GitHub Issues:** Create issue with `incident` label and severity tag (`sev-1`, `sev-2`, etc.)
- **Team chat:** Post in team channel immediately for SEV-1/SEV-2
- **Status updates:** Post updates every 30 minutes for SEV-1, every 2 hours for SEV-2
- **Resolution:** Comment on the issue when resolved with root cause summary

### 4. Mitigate
- If deployment-related: **rollback immediately** (see rollback procedures below)
- If DB-related: check connection pool, run `EXPLAIN ANALYZE` on slow queries
- If memory/CPU: scale up Cloud Run instances

### 5. Resolve
- Fix root cause
- Deploy fix
- Verify fix in production

### 6. Post-Incident
- Write post-incident review within 48 hours
- Identify action items to prevent recurrence
- Update this playbook if new scenario discovered

## Rollback Procedures

### Backend Rollback

```bash
# List recent revisions
gcloud run revisions list --service=businessos-backend --region=us-central1

# Route 100% traffic to previous revision
gcloud run services update-traffic businessos-backend \
  --region=us-central1 \
  --to-revisions=PREVIOUS_REVISION=100
```

### Frontend Rollback

```bash
gcloud run services update-traffic businessos-frontend \
  --region=us-central1 \
  --to-revisions=PREVIOUS_REVISION=100
```

### Database Rollback

```bash
# Option A: Apply rollback migration
psql "$DATABASE_URL" -f supabase/migrations/XXX_rollback.sql

# Option B: Restore from backup
gcloud sql backups list --instance=businessos-db
gcloud sql backups restore BACKUP_ID --restore-instance=businessos-db
```

## Common Scenarios

### Backend Returns 500

```
1. Check logs: gcloud run services logs read businessos-backend --limit=50
2. Look for: panic, nil pointer, connection refused, timeout
3. If panic: check recent deployment, rollback if needed
4. If connection refused: check Cloud SQL / Redis connectivity
5. If timeout: check for slow queries, increase timeout or optimize
```

### Database Connection Pool Exhausted

```
1. Check current connections: SELECT count(*) FROM pg_stat_activity;
2. Kill idle connections: SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'idle' AND query_start < now() - interval '10 minutes';
3. If persistent: increase pool size in backend config or scale Cloud SQL
```

### SSE Stream Not Connecting

```
1. Check if backend is running: curl $BACKEND_URL/health
2. Check CORS headers: curl -I $BACKEND_URL/api/osa/apps/generate/1/stream
3. Check if EventSource URL is correct (queueItemId, not appId)
4. Check Cloud Run timeout (SSE needs long-lived connections)
```

### Auth Not Working

```
1. Check Google OAuth redirect URIs match production domain
2. Check BETTER_AUTH_SECRET is set correctly
3. Check session cookie domain matches frontend domain
4. Check CORS allows credentials from frontend origin
```

### Redis Out of Memory (OOM)

```
1. Check memory usage: redis-cli INFO memory
2. Identify large keys: redis-cli --bigkeys
3. Check eviction policy: redis-cli CONFIG GET maxmemory-policy
4. If urgent: flush non-critical caches (redis-cli FLUSHDB on cache DB only)
5. Long-term: increase Redis instance size or review TTL settings
```

### LLM Provider Timeout

```
1. Check Anthropic status page: https://status.anthropic.com
2. Verify API key is valid: curl with a minimal prompt
3. Check if rate limits are hit (429 responses in logs)
4. Fallback: switch AI_PROVIDER env var to alternate provider (e.g., groq)
5. If persistent: add request timeout and retry logic in orchestration layer
```

### Docker Sandbox Failure (Desktop)

```
1. Verify Docker is running: docker info
2. Check workspace image exists: docker images businessos-workspace:latest
3. If missing, rebuild: cd desktop/backend-go/docker/workspace && docker build -t businessos-workspace:latest .
4. Check init script line endings: file internal/terminal/businessos_init.sh (must be "ASCII text")
5. If CRLF: sed -i 's/\r$//' docker/workspace/businessos_init.sh
```

## Monitoring Links

| Dashboard | URL |
|-----------|-----|
| Cloud Run Console | https://console.cloud.google.com/run?project=miosa-460433 |
| Cloud SQL Console | https://console.cloud.google.com/sql?project=miosa-460433 |
| Sentry | Set up after Sprint 1 deployment (see [SENTRY_SETUP.md](../deployment/SENTRY_SETUP.md)) |
| GCP Logging | https://console.cloud.google.com/logs?project=miosa-460433 |

**Note:** A public status page will be set up as a Sprint 2 deliverable once the production domain is finalized.

---

**Last Updated:** 2026-02-23
