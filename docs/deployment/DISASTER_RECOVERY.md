# Disaster Recovery Plan

## Overview

This document outlines the disaster recovery procedures for BusinessOS, including backup strategies, restore procedures, incident response, and recovery time objectives (RTO/RPO).

## Table of Contents

1. [Recovery Objectives](#recovery-objectives)
2. [Backup Strategy](#backup-strategy)
3. [Database Recovery](#database-recovery)
4. [Application Recovery](#application-recovery)
5. [Incident Response](#incident-response)
6. [Testing & Validation](#testing--validation)

---

## Recovery Objectives

### Definitions

**RTO (Recovery Time Objective):** Maximum acceptable time to restore service after an outage.

**RPO (Recovery Point Objective):** Maximum acceptable data loss measured in time.

### BusinessOS Objectives

| Service | RTO | RPO | Priority |
|---------|-----|-----|----------|
| Database (PostgreSQL) | 1 hour | 1 hour | Critical |
| Cache (Redis) | 30 minutes | None (ephemeral) | High |
| Backend API | 15 minutes | N/A | Critical |
| Frontend | 5 minutes | N/A | High |
| Background Jobs | 1 hour | N/A | Medium |

---

## Backup Strategy

### 1. Database Backups (Cloud SQL / Supabase)

#### Automated Backups

**Cloud SQL Configuration:**

```bash
# Enable automated backups
gcloud sql instances patch businessos-db \
  --backup-start-time=03:00 \
  --enable-bin-log \
  --retained-backups-count=7 \
  --retained-transaction-log-days=7
```

**Supabase Configuration:**
- Supabase automatically performs daily backups
- Point-in-time recovery (PITR) available for up to 7 days
- Backups retained for 30 days on Pro plan

#### Backup Schedule

| Backup Type | Frequency | Retention | Storage |
|-------------|-----------|-----------|---------|
| Automated snapshot | Daily @ 3 AM UTC | 7 days | Cloud SQL / Supabase |
| Transaction logs | Continuous | 7 days | Cloud SQL / Supabase |
| Manual snapshot | Weekly (Sunday) | 30 days | Cloud Storage |
| Full export | Monthly | 90 days | Cloud Storage (off-site) |

#### Manual Backup Commands

**Cloud SQL:**

```bash
# Create manual backup
gcloud sql backups create \
  --instance=businessos-db \
  --description="Manual backup $(date +%Y%m%d)"

# Export to Cloud Storage
gcloud sql export sql businessos-db \
  gs://businessos-backups/manual/businessos-$(date +%Y%m%d).sql \
  --database=businessos
```

**Supabase:**

```bash
# Export via pg_dump (requires database credentials)
pg_dump "postgresql://user:pass@db.supabase.co:5432/postgres" \
  --format=custom \
  --file=businessos-backup-$(date +%Y%m%d).dump

# Upload to Cloud Storage
gsutil cp businessos-backup-$(date +%Y%m%d).dump \
  gs://businessos-backups/supabase/
```

### 2. Redis Backups

**Redis Persistence Configuration:**

```yaml
# docker-compose.yml or redis.conf
command: >
  redis-server
  --appendonly yes              # Enable AOF
  --appendfsync everysec        # Fsync every second
  --save 900 1                  # Snapshot after 900s if 1 key changed
  --save 300 10                 # Snapshot after 300s if 10 keys changed
  --save 60 10000               # Snapshot after 60s if 10000 keys changed
```

**Backup Redis Data:**

```bash
# Trigger manual save
redis-cli BGSAVE

# Copy RDB file
cp /var/lib/redis/dump.rdb \
   /backups/redis/dump-$(date +%Y%m%d).rdb

# Upload to Cloud Storage
gsutil cp /backups/redis/dump-$(date +%Y%m%d).rdb \
  gs://businessos-backups/redis/
```

**Recovery Note:** Redis cache is ephemeral. In disaster recovery, start with empty Redis - sessions will rebuild from database.

### 3. Application Configuration Backups

**Backup Environment Variables:**

```bash
# Export secrets from GitHub
gh secret list --repo businessos/main > secrets-list.txt

# Backup GCP secrets
gcloud secrets versions access latest \
  --secret="DATABASE_URL" > backup-secrets/DATABASE_URL.txt

# Store encrypted in Cloud Storage
gpg --encrypt --recipient ops@businessos.com backup-secrets/
gsutil cp backup-secrets/*.gpg gs://businessos-backups/secrets/
```

**Backup Infrastructure Configuration:**

```bash
# Backup Terraform state
gsutil cp terraform.tfstate \
  gs://businessos-backups/terraform/terraform-$(date +%Y%m%d).tfstate

# Backup Kubernetes manifests (if using)
kubectl get all --all-namespaces -o yaml > k8s-state-$(date +%Y%m%d).yaml
gsutil cp k8s-state-$(date +%Y%m%d).yaml gs://businessos-backups/k8s/
```

---

## Database Recovery

### Scenario 1: Restore from Automated Backup

**Cloud SQL:**

```bash
# List available backups
gcloud sql backups list --instance=businessos-db

# Restore from backup
gcloud sql backups restore BACKUP_ID \
  --backup-instance=businessos-db \
  --backup-project=PROJECT_ID

# Alternatively, create new instance from backup
gcloud sql instances create businessos-db-restored \
  --backup=BACKUP_ID \
  --backup-instance=businessos-db
```

**Supabase:**

1. Go to [Supabase Dashboard](https://app.supabase.com)
2. Navigate to **Database** > **Backups**
3. Select backup date/time
4. Click **Restore**
5. Confirm restoration

**Estimated Time:** 15-30 minutes (depending on database size)

### Scenario 2: Point-in-Time Recovery

Recover database to specific timestamp (within last 7 days):

```bash
# Cloud SQL PITR
gcloud sql instances clone businessos-db \
  businessos-db-pitr \
  --point-in-time=2026-01-15T10:30:00Z
```

**Supabase PITR:**
- Available on Pro plan
- Use dashboard to select exact recovery point

**Estimated Time:** 20-40 minutes

### Scenario 3: Restore from Manual Export

```bash
# Download backup from Cloud Storage
gsutil cp gs://businessos-backups/manual/businessos-20260115.sql ./

# Restore to Cloud SQL
gcloud sql import sql businessos-db \
  gs://businessos-backups/manual/businessos-20260115.sql \
  --database=businessos

# Or restore to Supabase
psql "postgresql://user:pass@db.supabase.co:5432/postgres" \
  < businessos-20260115.sql
```

**Estimated Time:** 30-60 minutes (depending on size)

### Database Recovery Checklist

- [ ] Identify backup to restore (date/time)
- [ ] Verify backup integrity
- [ ] Stop application traffic (set maintenance mode)
- [ ] Create database snapshot before restore (safety)
- [ ] Execute restore command
- [ ] Verify data integrity post-restore
- [ ] Test database connectivity
- [ ] Run smoke tests (verify critical data)
- [ ] Resume application traffic
- [ ] Monitor logs for errors

---

## Application Recovery

### Scenario 1: Backend Service Down

**Symptoms:**
- Health checks failing
- 5xx errors
- Cloud Run instances crashing

**Recovery Steps:**

1. **Identify Issue:**
   ```bash
   # Check Cloud Run logs
   gcloud logging read "resource.type=cloud_run_revision" \
     --limit=100 \
     --format=json

   # Check service status
   gcloud run services describe businessos-backend \
     --region=us-central1
   ```

2. **Rollback to Previous Version:**
   ```bash
   # List revisions
   gcloud run revisions list \
     --service=businessos-backend \
     --region=us-central1

   # Rollback to previous revision
   gcloud run services update-traffic businessos-backend \
     --region=us-central1 \
     --to-revisions=PREVIOUS_REVISION=100
   ```

3. **If Rollback Fails, Redeploy Last Known Good Version:**
   ```bash
   # Get last successful commit from GitHub
   LAST_GOOD_SHA=$(git log --grep="Deployment: SUCCESS" -1 --format=%H)

   # Checkout and deploy
   git checkout $LAST_GOOD_SHA
   ./deploy-backend.sh
   ```

**Estimated Time:** 10-15 minutes

### Scenario 2: Frontend Service Down

**Recovery Steps:**

1. **Vercel Rollback:**
   ```bash
   # List deployments
   vercel ls

   # Rollback to previous deployment
   vercel rollback [DEPLOYMENT_URL]
   ```

2. **Redeploy from Git:**
   ```bash
   # Trigger deployment from last good commit
   git checkout main
   git reset --hard LAST_GOOD_COMMIT
   git push origin main --force

   # Or manually trigger deployment
   vercel --prod
   ```

**Estimated Time:** 5-10 minutes

### Scenario 3: Complete Regional Outage

**Multi-Region Failover:**

1. **Update DNS to Secondary Region:**
   ```bash
   # Point traffic to backup region
   gcloud dns record-sets transaction start --zone=businessos-zone

   gcloud dns record-sets transaction add \
     --name=api.businessos.com. \
     --type=A \
     --zone=businessos-zone \
     --ttl=300 \
     --rrdatas=BACKUP_IP

   gcloud dns record-sets transaction execute --zone=businessos-zone
   ```

2. **Activate Standby Infrastructure:**
   ```bash
   # Scale up standby Cloud Run service
   gcloud run services update businessos-backend-standby \
     --region=us-east1 \
     --min-instances=3

   # Update database replica to primary
   gcloud sql instances promote-replica businessos-db-replica
   ```

**Estimated Time:** 30-60 minutes

---

## Incident Response

### Incident Severity Levels

| Severity | Definition | Response Time | Example |
|----------|------------|---------------|---------|
| **P0 - Critical** | Complete service outage | Immediate | Database down, backend crashed |
| **P1 - High** | Major feature broken, data loss risk | < 15 min | Payment processing failing |
| **P2 - Medium** | Degraded performance, non-critical feature down | < 1 hour | Slow API responses, notifications delayed |
| **P3 - Low** | Minor issue, cosmetic bug | < 4 hours | UI glitch, typo |

### Incident Response Workflow

```
┌─────────────────────────────────────────────────────────────────┐
│                     INCIDENT DETECTED                           │
│                   (Alert, User Report)                          │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  1. ACKNOWLEDGE                                                 │
│     • Assign incident commander                                 │
│     • Create incident channel (#incident-YYYYMMDD-001)          │
│     • Update status page                                        │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  2. ASSESS                                                      │
│     • Determine severity (P0-P3)                                │
│     • Identify affected services                                │
│     • Estimate user impact                                      │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  3. MITIGATE                                                    │
│     • Execute runbook procedure                                 │
│     • Rollback if needed                                        │
│     • Engage escalation path                                    │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  4. MONITOR                                                     │
│     • Verify service restored                                   │
│     • Check error rates normalized                              │
│     • Confirm user impact resolved                              │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  5. COMMUNICATE                                                 │
│     • Update status page                                        │
│     • Notify affected users                                     │
│     • Document timeline                                         │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│  6. POST-MORTEM                                                 │
│     • Root cause analysis                                       │
│     • Action items                                              │
│     • Update runbooks                                           │
└─────────────────────────────────────────────────────────────────┘
```

### Incident Commander Responsibilities

1. **Own the incident** from detection to resolution
2. **Coordinate responders** and delegate tasks
3. **Make decisions** on mitigation strategies
4. **Communicate** with stakeholders
5. **Document** actions and timeline
6. **Lead post-mortem** review

### On-Call Rotation

Recommended setup:
- **Primary on-call:** 24/7 coverage, 1-week shifts
- **Secondary on-call:** Backup escalation
- **Manager on-call:** Escalation for P0 incidents

**Tools:**
- PagerDuty / Opsgenie for alert routing
- Slack for incident coordination
- Statuspage.io for customer communication

---

## Testing & Validation

### Quarterly Disaster Recovery Drills

**Test 1: Database Restore (Q1, Q3)**

```bash
# 1. Create test database
gcloud sql instances create businessos-test-restore \
  --tier=db-f1-micro \
  --region=us-central1

# 2. Restore from backup
gcloud sql backups restore LATEST_BACKUP_ID \
  --backup-instance=businessos-db \
  --instance=businessos-test-restore

# 3. Validate data
psql -h TEST_DB_IP -U postgres -d businessos \
  -c "SELECT COUNT(*) FROM users;"

# 4. Cleanup
gcloud sql instances delete businessos-test-restore
```

**Test 2: Application Rollback (Q2, Q4)**

```bash
# 1. Deploy test revision
gcloud run deploy businessos-backend-test \
  --image=gcr.io/PROJECT_ID/businessos-backend:v1.0.0

# 2. Simulate failure (manual trigger)

# 3. Execute rollback procedure
gcloud run services update-traffic businessos-backend-test \
  --to-revisions=PREVIOUS_REVISION=100

# 4. Measure rollback time
# Target: < 5 minutes

# 5. Cleanup
gcloud run services delete businessos-backend-test
```

**Test 3: Regional Failover (Annual)**

Full simulation of regional outage:
- Switch DNS to backup region
- Promote database replica
- Verify application functionality
- Measure total recovery time
- Document lessons learned

### Validation Checklist

After any recovery:

- [ ] Database accessible and responding
- [ ] All critical tables present
- [ ] User authentication working
- [ ] API endpoints responding
- [ ] Frontend loading correctly
- [ ] Background jobs processing
- [ ] Monitoring dashboards show green
- [ ] No data corruption detected
- [ ] Sample user workflows tested
- [ ] Incident documented in runbook

---

## Contact Information

### Escalation Path

| Role | Contact | Escalation Level |
|------|---------|------------------|
| Primary On-Call | on-call@businessos.com | L1 |
| Engineering Manager | manager@businessos.com | L2 |
| CTO | cto@businessos.com | L3 |
| External Support | Supabase, GCP, Vercel support | L3 |

### Vendor Support

**Supabase:**
- Support: support@supabase.com
- Status: https://status.supabase.com
- Docs: https://supabase.com/docs

**Google Cloud Platform:**
- Support: https://cloud.google.com/support
- Status: https://status.cloud.google.com

**Vercel:**
- Support: support@vercel.com
- Status: https://www.vercel-status.com

---

## Appendix

### Backup Storage Locations

| Backup Type | Primary Storage | Secondary Storage | Retention |
|-------------|-----------------|-------------------|-----------|
| Database snapshots | Cloud SQL / Supabase | Cloud Storage | 7 days |
| Manual exports | Cloud Storage (us-central1) | Cloud Storage (eu-west1) | 90 days |
| Application configs | GitHub repo | Cloud Storage | Indefinite |
| Infrastructure state | GCS (us) | GCS (eu) | 1 year |

### Recovery Time Estimates

Based on production data size (as of 2026-01):

| Database Size | Backup Time | Restore Time |
|---------------|-------------|--------------|
| < 1 GB | 2 minutes | 5 minutes |
| 1-10 GB | 10 minutes | 20 minutes |
| 10-100 GB | 30 minutes | 60 minutes |
| 100+ GB | 60+ minutes | 2+ hours |

### Useful Commands Reference

```bash
# Cloud SQL
gcloud sql backups list --instance=businessos-db
gcloud sql instances describe businessos-db
gcloud sql operations list --instance=businessos-db

# Cloud Run
gcloud run revisions list --service=businessos-backend
gcloud run services describe businessos-backend
gcloud run logs read businessos-backend

# Monitoring
gcloud monitoring policies list
gcloud monitoring uptime-checks list

# Storage
gsutil ls gs://businessos-backups/
gsutil du -s gs://businessos-backups/
```

---

## Document Maintenance

This document should be reviewed and updated:
- **Quarterly:** After disaster recovery drills
- **After incidents:** Update based on lessons learned
- **Annually:** Full review of all procedures

**Last Reviewed:** 2026-01-18
**Next Review:** 2026-04-18
**Owner:** DevOps Team
