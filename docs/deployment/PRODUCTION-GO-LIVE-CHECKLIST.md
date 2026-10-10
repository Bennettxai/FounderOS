# Production Go-Live Checklist

> **Every step required to take BusinessOS from local development to live production.**
> Covers: Web (Cloud Run), Desktop (Electron), environment rotation, database migration, secrets, monitoring, rollback.
>
> Last Updated: 2026-02-23
> Status: CHECKLIST (execute in order, top to bottom)

---

## Phase 0: Pre-Flight (Before Anything Else)

### Code Readiness

- [ ] All Sprint 1 bug fixes merged (FK violation, prompt overwrite, SSE routes, hardcoded URLs)
- [ ] `cd desktop/backend-go && go build ./cmd/server` → EXIT 0
- [ ] `cd desktop/backend-go && go test ./...` → ALL PASS
- [ ] `cd frontend && npm run build` → EXIT 0
- [ ] `cd frontend && npm run check` → EXIT 0
- [ ] No hardcoded credentials in codebase (run: `grep -r "sk-ant\|password.*=.*\"[^\"]*\"" --include="*.go" --include="*.ts" desktop/backend-go/ frontend/src/`)
- [ ] No `localhost` URLs in production code paths
- [ ] All environment variables use `os.Getenv()` or config struct (no hardcoded values)

### GCP Project Setup

- [ ] GCP project exists: `miosa-460433`
- [ ] Billing enabled on GCP project
- [ ] APIs enabled:
  ```bash
  gcloud services enable \
    run.googleapis.com \
    sqladmin.googleapis.com \
    redis.googleapis.com \
    secretmanager.googleapis.com \
    containerregistry.googleapis.com \
    cloudbuild.googleapis.com \
    compute.googleapis.com
  ```
- [ ] `gcloud` CLI authenticated: `gcloud auth login`
- [ ] Docker authenticated: `gcloud auth configure-docker`

---

## Phase 1: Database (Cloud SQL)

### Create Instance

```bash
# Create Cloud SQL PostgreSQL instance (if not already created)
gcloud sql instances create businessos-db \
  --database-version=POSTGRES_15 \
  --tier=db-f1-micro \
  --region=us-central1 \
  --root-password=$(openssl rand -base64 32)

# Create database
gcloud sql databases create businessos --instance=businessos-db

# Get connection name (needed for Cloud Run)
gcloud sql instances describe businessos-db --format='value(connectionName)'
# → miosa-460433:us-central1:businessos-db
```

### Run Migrations

```bash
# Option A: Cloud SQL Proxy (recommended for migration)
cloud-sql-proxy miosa-460433:us-central1:businessos-db --port=5433 &

# Apply all 93 migrations
for f in supabase/migrations/*.sql; do
  echo "Applying: $f"
  psql "postgresql://postgres:PASSWORD@localhost:5433/businessos" -f "$f"
done

# Verify migration count
psql "postgresql://postgres:PASSWORD@localhost:5433/businessos" \
  -c "SELECT count(*) FROM information_schema.tables WHERE table_schema='public';"
```

### Checklist

- [ ] Cloud SQL instance running
- [ ] Database `businessos` created
- [ ] All 93 migrations applied (including 090_tenant_org_foundation.sql)
- [ ] Connection string noted: `postgresql://user:pass@/businessos?host=/cloudsql/miosa-460433:us-central1:businessos-db`
- [ ] pgvector extension enabled: `CREATE EXTENSION IF NOT EXISTS vector;`

---

## Phase 2: Redis (Memorystore or Cloud Run Sidecar)

### Option A: GCP Memorystore (Production)

```bash
gcloud redis instances create businessos-redis \
  --region=us-central1 \
  --tier=BASIC \
  --size=1
```

### Option B: Redis Container (Budget)

Deploy Redis as a separate Cloud Run service or use a managed Redis from Upstash/Railway.

### Checklist

- [ ] Redis instance accessible from Cloud Run
- [ ] Connection URL noted: `redis://HOST:PORT`
- [ ] Verified connection: `redis-cli -h HOST -p PORT PING`

---

## Phase 3: Backend Deployment (Cloud Run)

### Build & Push Container

```bash
# Build backend Docker image
cd desktop/backend-go
docker build -t gcr.io/miosa-460433/businessos-backend:latest .
docker push gcr.io/miosa-460433/businessos-backend:latest
```

### Deploy to Cloud Run

```bash
gcloud run deploy businessos-backend \
  --image=gcr.io/miosa-460433/businessos-backend:latest \
  --platform=managed \
  --region=us-central1 \
  --allow-unauthenticated \
  --port=8080 \
  --memory=512Mi \
  --cpu=1 \
  --min-instances=0 \
  --max-instances=10 \
  --add-cloudsql-instances=miosa-460433:us-central1:businessos-db \
  --set-env-vars="^||^DATABASE_URL=postgresql://postgres:PASSWORD@/businessos?host=/cloudsql/miosa-460433:us-central1:businessos-db||REDIS_URL=redis://HOST:PORT||ENVIRONMENT=production||ANTHROPIC_API_KEY=sk-ant-xxx||BETTER_AUTH_SECRET=$(openssl rand -hex 32)||CORS_ALLOWED_ORIGINS=https://businessos.app,https://app.businessos.app||PORT=8080"
```

### Verify Backend

```bash
# Get service URL
BACKEND_URL=$(gcloud run services describe businessos-backend --region=us-central1 --format='value(status.url)')

# Health check
curl "$BACKEND_URL/health"
# Expected: {"status":"ok"}

# API check
curl "$BACKEND_URL/api/health"
# Expected: 200 OK with JSON response
```

### Checklist

- [ ] Docker image built and pushed to GCR
- [ ] Cloud Run service deployed
- [ ] Cloud SQL connection working (check logs: `gcloud run services logs read businessos-backend`)
- [ ] Redis connection working
- [ ] Health endpoint returns 200
- [ ] CORS configured for production domains only
- [ ] Environment variables set (no hardcoded secrets)

---

## Phase 4: Frontend Deployment (Cloud Run)

### Build & Push Container

```bash
# Build frontend Docker image
cd frontend
docker build -t gcr.io/miosa-460433/businessos-frontend:latest .
docker push gcr.io/miosa-460433/businessos-frontend:latest
```

### Deploy to Cloud Run

```bash
gcloud run deploy businessos-frontend \
  --image=gcr.io/miosa-460433/businessos-frontend:latest \
  --platform=managed \
  --region=us-central1 \
  --allow-unauthenticated \
  --port=3000 \
  --memory=256Mi \
  --cpu=1 \
  --min-instances=0 \
  --max-instances=5 \
  --set-env-vars="^||^PUBLIC_API_URL=$BACKEND_URL||PUBLIC_SUPABASE_URL=https://your-project.supabase.co||PUBLIC_SUPABASE_ANON_KEY=your-anon-key||ORIGIN=https://businessos.app"
```

### Checklist

- [ ] Frontend Docker image built and pushed
- [ ] Cloud Run service deployed
- [ ] `PUBLIC_API_URL` points to backend Cloud Run URL
- [ ] Frontend loads without console errors
- [ ] Auth flow works (login → redirect → dashboard)

---

## Phase 5: DNS & Custom Domain

```bash
# Map custom domain to frontend
gcloud run domain-mappings create \
  --service=businessos-frontend \
  --domain=businessos.app \
  --region=us-central1

# Map API subdomain to backend
gcloud run domain-mappings create \
  --service=businessos-backend \
  --domain=api.businessos.app \
  --region=us-central1
```

### DNS Records (at your registrar)

| Type | Name | Value |
|------|------|-------|
| CNAME | `businessos.app` | `ghs.googlehosted.com.` |
| CNAME | `api.businessos.app` | `ghs.googlehosted.com.` |

### Checklist

- [ ] DNS records configured
- [ ] SSL certificates provisioned by Cloud Run (automatic)
- [ ] `https://businessos.app` loads frontend
- [ ] `https://api.businessos.app/health` returns 200
- [ ] CORS updated to include custom domains

---

## Phase 6: Auth Configuration (Production)

### Google OAuth

```bash
# In Google Cloud Console → APIs & Services → Credentials
# Create OAuth 2.0 Client ID:
#   Application type: Web application
#   Authorized redirect URIs:
#     - https://businessos.app/api/auth/callback/google
#     - https://api.businessos.app/api/auth/callback/google
```

### BetterAuth

Set these environment variables on backend Cloud Run:

| Variable | Value |
|----------|-------|
| `BETTER_AUTH_SECRET` | Random 64-char hex string |
| `BETTER_AUTH_URL` | `https://api.businessos.app` |
| `GOOGLE_CLIENT_ID` | From Google Cloud Console |
| `GOOGLE_CLIENT_SECRET` | From Google Cloud Console |

### Checklist

- [ ] Google OAuth client configured for production URLs
- [ ] BetterAuth secret set (unique per environment)
- [ ] Login flow works: Google OAuth → callback → session → dashboard
- [ ] Session cookies have `Secure` and `HttpOnly` flags
- [ ] Session expiration configured (default: 30 days)

---

## Phase 7: GitHub Secrets (CI/CD)

Set these in your GitHub repository (Settings → Secrets and variables → Actions):

### Required Secrets

| Secret | Value | Used By |
|--------|-------|---------|
| `GCP_PROJECT_ID` | `miosa-460433` | All deploy workflows |
| `GCP_SA_KEY` | Service account JSON key (base64) | GCP authentication |
| `DATABASE_URL` | Cloud SQL connection string | Backend deploy |
| `REDIS_URL` | Redis connection URL | Backend deploy |
| `ANTHROPIC_API_KEY` | Anthropic API key | Backend runtime |
| `BETTER_AUTH_SECRET` | Auth secret (hex) | Backend runtime |
| `GOOGLE_CLIENT_ID` | OAuth client ID | Auth |
| `GOOGLE_CLIENT_SECRET` | OAuth client secret | Auth |
| `SUPABASE_URL` | Supabase project URL | Frontend runtime |
| `SUPABASE_ANON_KEY` | Supabase anon key | Frontend runtime |

### Optional Secrets (Desktop Builds)

| Secret | Value | Used By |
|--------|-------|---------|
| `APPLE_ID` | Apple Developer email | macOS code signing |
| `APPLE_PASSWORD` | App-specific password | macOS notarization |
| `APPLE_TEAM_ID` | Apple Team ID | macOS signing |
| `APPLE_IDENTITY` | Developer ID Application string | macOS signing |
| `WIN_CSC_LINK` | Windows code signing cert (base64) | Windows signing |
| `WIN_CSC_KEY_PASSWORD` | Cert password | Windows signing |

### Checklist

- [ ] All required secrets set
- [ ] CI/CD deploy workflow updated (`.github/workflows/deploy-backend.yml`)
- [ ] CI/CD deploy workflow updated (`.github/workflows/deploy-frontend.yml`) — Cloud Run, NOT Vercel
- [ ] Test deployment via CI: push to `main`, verify auto-deploy succeeds

---

## Phase 8: Desktop App (Electron Packaging)

> See `docs/deployment/ELECTRON-PACKAGING-GUIDE.md` for full details.

### Quick Steps

```bash
# 1. Build Go backend binaries for all platforms
cd desktop/backend-go
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" \
  -o ../desktop/resources/bin/darwin-arm64/businessos-server ./cmd/server

# 2. Install Electron dependencies
cd desktop && npm install

# 3. Build DMG (macOS)
npm run make -- --platform=darwin

# 4. Output: desktop/out/make/BusinessOS.dmg
```

### Checklist

- [ ] Go binaries built for target platforms
- [ ] `desktop/resources/icons/` has icon.icns, icon.ico, icon.png
- [x] `forge.config.ts` publisher defaults to `Miosa-osa/businessos-5`
- [ ] DMG builds without errors
- [ ] DMG installs and launches correctly
- [ ] Backend sidecar starts (check: `lsof -i :18080`)
- [ ] Frontend connects to sidecar backend
- [ ] SQLite database created at `~/Library/Application Support/BusinessOS/`

---

## Phase 9: Monitoring & Observability

### Cloud Run Metrics (Built-in)

```bash
# View logs
gcloud run services logs read businessos-backend --region=us-central1 --limit=100

# View metrics in console
# https://console.cloud.google.com/run?project=miosa-460433
```

### Error Tracking (Sentry)

See `docs/deployment/SENTRY_SETUP.md` for Sentry integration.

- [ ] Sentry DSN configured for backend
- [ ] Sentry DSN configured for frontend
- [ ] Error alerts configured (email or Slack)

### Uptime Monitoring

```bash
# Set up Google Cloud Monitoring uptime check
# https://console.cloud.google.com/monitoring/uptime?project=miosa-460433
#
# Check: GET https://api.businessos.app/health every 5 minutes
# Alert: Email + Slack if down for 2+ consecutive checks
```

### Checklist

- [ ] Cloud Run logs accessible
- [ ] Sentry configured (optional but recommended)
- [ ] Uptime monitoring configured
- [ ] Alert channels set up (email/Slack)

---

## Phase 10: Smoke Test (E2E Verification)

### Web App

- [ ] Navigate to `https://businessos.app`
- [ ] Sign in with Google OAuth
- [ ] Create a workspace
- [ ] Open chat interface
- [ ] Send: "Create a todo list app"
- [ ] Verify: SSE progress stream appears
- [ ] Verify: Generation completes (4 agent tasks)
- [ ] Verify: Files visible in generated apps list
- [ ] Verify: Can open app in Monaco editor
- [ ] Verify: Can view file contents

### Desktop App

- [ ] Install DMG / EXE / DEB
- [ ] Launch app
- [ ] Verify: Backend sidecar starts (system tray icon appears)
- [ ] Sign in with Google OAuth
- [ ] Create workspace
- [ ] Generate an app
- [ ] Verify: Same E2E flow as web
- [ ] Verify: Cmd+Shift+Space opens quick chat popup (macOS)
- [ ] Close and reopen app → verify data persists (SQLite)

---

## Environment Rotation Plan

### Three Environments

| Environment | Backend URL | Frontend URL | Database | Purpose |
|-------------|------------|-------------|----------|---------|
| **Local** | `localhost:8000` | `localhost:5173` | Local PostgreSQL | Development |
| **Staging** | `businessos-backend-staging-xxx.run.app` | `businessos-frontend-staging-xxx.run.app` | Cloud SQL (staging DB) | Testing before production |
| **Production** | `api.businessos.app` | `businessos.app` | Cloud SQL (production DB) | Live users |

### Promotion Flow

```
LOCAL (developer machines)
    │
    │ PR merge to main → CI tests pass
    │
    ▼
STAGING (auto-deploy on main merge)
    │
    │ QA verification + smoke test pass
    │ Manual approval by team lead
    │
    ▼
PRODUCTION (manual deploy or tag-triggered)
```

### Staging Setup

```bash
# Create staging Cloud SQL database
gcloud sql databases create businessos_staging --instance=businessos-db

# Deploy staging backend
gcloud run deploy businessos-backend-staging \
  --image=gcr.io/miosa-460433/businessos-backend:latest \
  --region=us-central1 \
  --set-env-vars="DATABASE_URL=...staging...||ENVIRONMENT=staging"

# Deploy staging frontend
gcloud run deploy businessos-frontend-staging \
  --image=gcr.io/miosa-460433/businessos-frontend:latest \
  --region=us-central1 \
  --set-env-vars="PUBLIC_API_URL=https://businessos-backend-staging-xxx.run.app"
```

---

## Rollback Plan

### Backend Rollback

```bash
# List recent revisions
gcloud run revisions list --service=businessos-backend --region=us-central1

# Route traffic to previous revision
gcloud run services update-traffic businessos-backend \
  --region=us-central1 \
  --to-revisions=businessos-backend-PREVIOUS_REVISION=100
```

### Frontend Rollback

```bash
gcloud run services update-traffic businessos-frontend \
  --region=us-central1 \
  --to-revisions=businessos-frontend-PREVIOUS_REVISION=100
```

### Database Rollback

```bash
# Apply rollback migration (if available)
psql "$DATABASE_URL" -f supabase/migrations/XXX_rollback.sql

# Or restore from backup
gcloud sql backups list --instance=businessos-db
gcloud sql backups restore BACKUP_ID --restore-instance=businessos-db
```

---

## Post-Launch (Day 1-3)

- [ ] Monitor error rates in Sentry/Cloud Run logs
- [ ] Monitor response times (p50 < 200ms, p99 < 2s)
- [ ] Monitor database connection pool usage
- [ ] Check for any 5xx errors
- [ ] Verify auto-scaling works under load
- [ ] Verify Redis session caching is working
- [ ] Test auto-update for desktop app (publish a minor version bump)
- [ ] Document any issues found and create Sprint 2 tasks

---

**Related Docs:**
- `docs/deployment/DEPLOYMENT.md` — Detailed GCP setup
- `docs/deployment/DEPLOYMENT_GUIDE.md` — Backend-specific deployment
- `docs/deployment/ELECTRON-PACKAGING-GUIDE.md` — Desktop app packaging
- `docs/deployment/CLOUD-INFRASTRUCTURE.md` — Cloud infrastructure details
- `docs/deployment/DISASTER_RECOVERY.md` — DR procedures
- `docs/deployment/MONITORING_SETUP.md` — Monitoring configuration
