# BusinessOS Backend - Cloud Run Deployment Guide

Complete setup guide for deploying the Go backend to Google Cloud Run with
Cloud SQL (PostgreSQL), Memorystore (Redis), and GitHub Actions CI/CD.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [GCP Project Setup](#gcp-project-setup)
3. [Cloud SQL (PostgreSQL)](#cloud-sql-postgresql)
4. [Memorystore (Redis)](#memorystore-redis)
5. [Artifact Registry](#artifact-registry)
6. [Secret Manager](#secret-manager)
7. [Service Account for CI/CD](#service-account-for-cicd)
8. [GitHub Secrets Checklist](#github-secrets-checklist)
9. [Manual Deployment](#manual-deployment)
10. [Automated CI/CD (GitHub Actions)](#automated-cicd-github-actions)
11. [Post-Deployment Verification](#post-deployment-verification)
12. [Rollback Procedures](#rollback-procedures)
13. [Monitoring and Alerts](#monitoring-and-alerts)
14. [Cost Optimization](#cost-optimization)
15. [Troubleshooting](#troubleshooting)

---

## Prerequisites

- Google Cloud SDK (`gcloud`) installed and authenticated
- Docker installed locally (for manual builds)
- `gh` CLI (for GitHub Secrets setup, optional)
- A GCP project with billing enabled

```bash
# Verify gcloud is configured
gcloud config get-value project
gcloud auth list
```

---

## GCP Project Setup

```bash
# Set your project
export GCP_PROJECT_ID="your-project-id"
export GCP_REGION="us-central1"
gcloud config set project $GCP_PROJECT_ID

# Enable required APIs
gcloud services enable \
  run.googleapis.com \
  sqladmin.googleapis.com \
  redis.googleapis.com \
  secretmanager.googleapis.com \
  artifactregistry.googleapis.com \
  cloudbuild.googleapis.com \
  iam.googleapis.com
```

---

## Cloud SQL (PostgreSQL)

### Create Instance

```bash
# Create a Cloud SQL PostgreSQL 15 instance
gcloud sql instances create businessos-db \
  --database-version=POSTGRES_15 \
  --tier=db-f1-micro \
  --region=$GCP_REGION \
  --storage-auto-increase \
  --availability-type=zonal \
  --deletion-protection

# Set the postgres user password
DB_PASSWORD=$(openssl rand -base64 24)
echo "Save this password securely: $DB_PASSWORD"

gcloud sql users set-password postgres \
  --instance=businessos-db \
  --password="$DB_PASSWORD"

# Create the application database
gcloud sql databases create business_os \
  --instance=businessos-db
```

### Enable pgvector Extension

Connect to the database and enable pgvector (required for RAG embeddings):

```bash
# Connect via Cloud SQL Proxy or gcloud
gcloud sql connect businessos-db --user=postgres --database=business_os

# In the psql session:
CREATE EXTENSION IF NOT EXISTS vector;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
\q
```

### Connection String Format

For Cloud Run with Cloud SQL connector:

```
postgres://postgres:PASSWORD@/business_os?host=/cloudsql/PROJECT_ID:REGION:businessos-db
```

Example:
```
postgres://postgres:mypassword@/business_os?host=/cloudsql/my-project:us-central1:businessos-db
```

### Run Migrations

After creating the database, apply migrations:

```bash
# Option 1: Use the migration scripts in the repo
# Connect via Cloud SQL proxy first, then run:
cd desktop/backend-go
for f in migrations/*.sql; do
  echo "Applying $f..."
  psql "$DATABASE_URL" -f "$f"
done

# Option 2: The server auto-runs migrations on startup (if configured)
```

---

## Memorystore (Redis)

### Create Instance

```bash
# Create a Memorystore Redis instance
gcloud redis instances create businessos-redis \
  --size=1 \
  --region=$GCP_REGION \
  --redis-version=redis_7_0 \
  --tier=basic \
  --transit-encryption-mode=SERVER_AUTHENTICATION

# Get the Redis host IP (private IP - requires VPC connector)
gcloud redis instances describe businessos-redis \
  --region=$GCP_REGION \
  --format="value(host)"
```

### VPC Connector (Required for Redis)

Cloud Run needs a Serverless VPC Access connector to reach Memorystore:

```bash
# Create VPC connector
gcloud compute networks vpc-access connectors create businessos-vpc \
  --region=$GCP_REGION \
  --range=10.8.0.0/28 \
  --min-instances=2 \
  --max-instances=3

# Verify
gcloud compute networks vpc-access connectors describe businessos-vpc \
  --region=$GCP_REGION
```

### Redis Connection String

```
redis://REDIS_HOST:6379/0
```

The `REDIS_HOST` is the private IP from the `gcloud redis instances describe` command.

When `REDIS_TLS_ENABLED=true`, the Go client handles in-transit encryption automatically.

---

## Artifact Registry

```bash
# Create a Docker repository
gcloud artifacts repositories create businessos \
  --repository-format=docker \
  --location=us \
  --description="BusinessOS container images"

# Verify
gcloud artifacts repositories list --location=us
```

Image path format: `us-docker.pkg.dev/PROJECT_ID/businessos/backend`

---

## Secret Manager

Store all sensitive values in Secret Manager (not as plaintext env vars).

```bash
# Helper function to create or update a secret
create_secret() {
  local name=$1
  local value=$2
  echo -n "$value" | gcloud secrets create "$name" --data-file=- 2>/dev/null || \
  echo -n "$value" | gcloud secrets versions add "$name" --data-file=-
  echo "Secret '$name' set."
}

# --- Required secrets ---

# Database connection string
create_secret DATABASE_URL "postgres://postgres:PASSWORD@/business_os?host=/cloudsql/$GCP_PROJECT_ID:$GCP_REGION:businessos-db"

# Redis
create_secret REDIS_URL "redis://REDIS_HOST_IP:6379/0"
create_secret REDIS_PASSWORD "your-redis-auth-password"

# JWT / Auth
create_secret SECRET_KEY "$(openssl rand -base64 48)"
create_secret REDIS_KEY_HMAC_SECRET "$(openssl rand -base64 48)"
create_secret TOKEN_ENCRYPTION_KEY "$(openssl rand -base64 32)"

# Google OAuth
create_secret GOOGLE_CLIENT_ID "your-client-id.apps.googleusercontent.com"
create_secret GOOGLE_CLIENT_SECRET "your-google-secret"

# AI Provider
create_secret ANTHROPIC_API_KEY "sk-ant-..."

# --- Optional secrets ---
# create_secret GROQ_API_KEY "gsk_..."
# create_secret OPENAI_API_KEY "sk-..."
# create_secret BRAVE_SEARCH_API_KEY "BSA..."
# create_secret VAPID_PUBLIC_KEY "..."
# create_secret VAPID_PRIVATE_KEY "..."
# create_secret WEBHOOK_SIGNING_SECRET "$(openssl rand -base64 32)"
```

### Grant Cloud Run Access to Secrets

```bash
PROJECT_NUMBER=$(gcloud projects describe $GCP_PROJECT_ID --format="value(projectNumber)")
SA="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"

SECRETS=(
  DATABASE_URL REDIS_URL REDIS_PASSWORD SECRET_KEY
  REDIS_KEY_HMAC_SECRET TOKEN_ENCRYPTION_KEY
  GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET ANTHROPIC_API_KEY
)

for secret in "${SECRETS[@]}"; do
  gcloud secrets add-iam-policy-binding "$secret" \
    --member="serviceAccount:$SA" \
    --role="roles/secretmanager.secretAccessor" \
    --quiet
done
```

---

## Service Account for CI/CD

Create a dedicated service account for GitHub Actions:

```bash
# Create service account
gcloud iam service-accounts create github-actions-deploy \
  --display-name="GitHub Actions Deploy"

SA_EMAIL="github-actions-deploy@${GCP_PROJECT_ID}.iam.gserviceaccount.com"

# Grant required roles
ROLES=(
  "roles/run.admin"
  "roles/iam.serviceAccountUser"
  "roles/artifactregistry.writer"
  "roles/secretmanager.secretAccessor"
  "roles/cloudsql.client"
)

for role in "${ROLES[@]}"; do
  gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
    --member="serviceAccount:$SA_EMAIL" \
    --role="$role" \
    --quiet
done

# Create and download JSON key
gcloud iam service-accounts keys create sa-key.json \
  --iam-account=$SA_EMAIL

echo "Key saved to sa-key.json - add this as GCP_SA_KEY GitHub Secret"
echo "DELETE sa-key.json after uploading to GitHub Secrets!"
```

---

## GitHub Secrets Checklist

Configure these in your repo: **Settings > Secrets and variables > Actions**

### Required

| Secret Name | Description | How to get |
|-------------|-------------|------------|
| `GCP_PROJECT_ID` | GCP project ID | `gcloud config get-value project` |
| `GCP_SA_KEY` | Service account JSON key | See [Service Account](#service-account-for-cicd) |
| `GCP_REGION` | Deploy region (e.g. `us-central1`) | Your choice |
| `ALLOWED_ORIGINS` | CORS origins (comma-separated) | e.g. `https://app.businessos.com` |

### Required in GCP Secret Manager

These are accessed by Cloud Run via `--set-secrets` (not GitHub Secrets):

| Secret Name | Description | Generate with |
|-------------|-------------|---------------|
| `DATABASE_URL` | Cloud SQL connection string | See [Cloud SQL](#cloud-sql-postgresql) |
| `REDIS_URL` | Redis connection URL | See [Memorystore](#memorystore-redis) |
| `REDIS_PASSWORD` | Redis AUTH password | Cloud console or `openssl rand -base64 32` |
| `SECRET_KEY` | JWT signing key (min 32 chars) | `openssl rand -base64 48` |
| `REDIS_KEY_HMAC_SECRET` | HMAC for Redis key derivation | `openssl rand -base64 48` |
| `TOKEN_ENCRYPTION_KEY` | AES-256 encryption key | `openssl rand -base64 32` |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID | GCP Console > APIs & Services > Credentials |
| `GOOGLE_CLIENT_SECRET` | Google OAuth client secret | Same as above |
| `ANTHROPIC_API_KEY` | Claude API key | console.anthropic.com |

---

## Manual Deployment

For testing or one-off deployments without CI/CD:

### Build and Push Image

```bash
cd desktop/backend-go

# Build
docker build -t us-docker.pkg.dev/$GCP_PROJECT_ID/businessos/backend:manual .

# Configure Docker auth
gcloud auth configure-docker us-docker.pkg.dev --quiet

# Push
docker push us-docker.pkg.dev/$GCP_PROJECT_ID/businessos/backend:manual
```

### Deploy to Cloud Run

```bash
gcloud run deploy businessos-backend \
  --image=us-docker.pkg.dev/$GCP_PROJECT_ID/businessos/backend:manual \
  --region=$GCP_REGION \
  --platform=managed \
  --allow-unauthenticated \
  --port=8080 \
  --memory=512Mi \
  --cpu=1 \
  --min-instances=0 \
  --max-instances=10 \
  --concurrency=80 \
  --timeout=300 \
  --vpc-connector=businessos-vpc \
  --set-env-vars="\
ENVIRONMENT=production,\
SERVER_PORT=8080,\
AI_PROVIDER=anthropic,\
ENABLE_LOCAL_MODELS=false,\
REDIS_TLS_ENABLED=true,\
DATABASE_REQUIRED=true,\
ALLOWED_ORIGINS=https://app.businessos.com" \
  --set-secrets="\
DATABASE_URL=DATABASE_URL:latest,\
REDIS_URL=REDIS_URL:latest,\
REDIS_PASSWORD=REDIS_PASSWORD:latest,\
SECRET_KEY=SECRET_KEY:latest,\
REDIS_KEY_HMAC_SECRET=REDIS_KEY_HMAC_SECRET:latest,\
TOKEN_ENCRYPTION_KEY=TOKEN_ENCRYPTION_KEY:latest,\
GOOGLE_CLIENT_ID=GOOGLE_CLIENT_ID:latest,\
GOOGLE_CLIENT_SECRET=GOOGLE_CLIENT_SECRET:latest,\
ANTHROPIC_API_KEY=ANTHROPIC_API_KEY:latest" \
  --add-cloudsql-instances=$GCP_PROJECT_ID:$GCP_REGION:businessos-db

# Get deployed URL
gcloud run services describe businessos-backend \
  --region=$GCP_REGION \
  --format='value(status.url)'
```

### Quick Deploy Script

The repository includes `desktop/backend-go/deploy.sh` for interactive deployment:

```bash
cd desktop/backend-go

# First time: set up GCP project + database
./deploy.sh setup

# Configure secrets in Secret Manager
./deploy.sh secrets

# Build and deploy
./deploy.sh deploy
```

---

## Automated CI/CD (GitHub Actions)

The workflow at `.github/workflows/deploy-backend.yml` automatically:

1. **Triggers** on push to `main` when `desktop/backend-go/**` files change
2. **Tests** the codebase with `go test`
3. **Builds** a Docker image and pushes to Artifact Registry
4. **Captures** the current revision for rollback
5. **Deploys** to Cloud Run with secrets from Secret Manager
6. **Smoke tests** the `/health` and `/api/status` endpoints
7. **Rolls back** automatically if smoke tests fail

### Trigger a Deploy

```bash
# Automatic: push changes to main
git push origin main

# Manual: trigger via GitHub UI or CLI
gh workflow run deploy-backend.yml
```

---

## Post-Deployment Verification

```bash
SERVICE_URL=$(gcloud run services describe businessos-backend \
  --region=$GCP_REGION --format='value(status.url)')

# 1. Basic health
curl -s "$SERVICE_URL/health"
# Expected: {"status":"healthy"}

# 2. Detailed health (includes DB + Redis status)
curl -s "$SERVICE_URL/health/detailed" | jq .
# Expected: {"status":"healthy","components":{"database":{...},"redis":{...}}}

# 3. API status
curl -s "$SERVICE_URL/api/status"

# 4. Check logs
gcloud run services logs read businessos-backend \
  --region=$GCP_REGION --limit=50
```

---

## Rollback Procedures

### Automatic Rollback (CI/CD)

The GitHub Actions workflow automatically rolls back on smoke test failure.
No manual intervention needed.

### Manual Rollback - Traffic Shift

Route traffic back to a previous known-good revision:

```bash
# List recent revisions
gcloud run revisions list \
  --service=businessos-backend \
  --region=$GCP_REGION \
  --format="table(metadata.name, status.conditions[0].status, metadata.creationTimestamp)" \
  --limit=5

# Route 100% traffic to a specific revision
gcloud run services update-traffic businessos-backend \
  --region=$GCP_REGION \
  --to-revisions=businessos-backend-REVISION_ID=100
```

### Manual Rollback - Redeploy Previous Image

```bash
# List images in Artifact Registry
gcloud artifacts docker images list \
  us-docker.pkg.dev/$GCP_PROJECT_ID/businessos/backend \
  --sort-by=~UPDATE_TIME --limit=5

# Deploy a specific image tag (e.g., a previous commit SHA)
gcloud run deploy businessos-backend \
  --image=us-docker.pkg.dev/$GCP_PROJECT_ID/businessos/backend:PREVIOUS_SHA \
  --region=$GCP_REGION
```

### Emergency: Delete Bad Revision

```bash
# Delete a specific revision (cannot be undone)
gcloud run revisions delete REVISION_NAME \
  --region=$GCP_REGION
```

---

## Monitoring and Alerts

### View Logs

```bash
# Recent logs
gcloud run services logs read businessos-backend \
  --region=$GCP_REGION --limit=100

# Stream logs in real time
gcloud run services logs tail businessos-backend \
  --region=$GCP_REGION

# Filter for errors
gcloud logging read \
  "resource.type=cloud_run_revision AND resource.labels.service_name=businessos-backend AND severity>=ERROR" \
  --limit=20 --format=json
```

### Recommended Alerts (Cloud Monitoring)

Set up alerting policies for:

- **5xx error rate > 5%** over 5 minutes
- **Latency p99 > 5s** over 5 minutes
- **Container instance count = max (10)** (scaling ceiling)
- **Cloud SQL CPU > 80%** over 10 minutes
- **Redis memory > 80%** of max

---

## Cost Optimization

| Resource | Recommended Config | Monthly Est. |
|----------|-------------------|--------------|
| Cloud Run | min=0, max=10, 512Mi, 1 CPU | $5-50 (usage-based) |
| Cloud SQL | db-f1-micro (dev) / db-custom-1-3840 (prod) | $8-40 |
| Memorystore Redis | 1GB Basic tier | $35 |
| Artifact Registry | Standard storage | $0.10/GB |
| VPC Connector | 2 min instances | $15 |

Tips:
- Use `min-instances=0` to scale to zero when idle
- Use `db-f1-micro` for development/staging
- Consider Cloud SQL Auth Proxy sidecar for connection pooling

---

## Troubleshooting

### "Connection refused" to Cloud SQL

- Verify `--add-cloudsql-instances` flag in deploy command
- Check the DATABASE_URL format uses `/cloudsql/` socket path
- Ensure the Cloud SQL Admin API is enabled

### "Connection refused" to Redis

- Memorystore requires a VPC connector
- Add `--vpc-connector=businessos-vpc` to the deploy command
- Verify the connector is in the same region as Redis

### Container fails to start

```bash
# Check startup logs
gcloud run services logs read businessos-backend \
  --region=$GCP_REGION --limit=20

# Common causes:
# - Missing required env vars / secrets
# - Database migration needed
# - Secret Manager permissions not granted
```

### Health check fails

- The `/health` endpoint returns `{"status":"healthy"}` with HTTP 200
- Cloud Run uses its own TCP/HTTP health probes
- Check if the server port matches (`SERVER_PORT=8080` must match `--port=8080`)

### Secrets not accessible

```bash
# Verify secret exists
gcloud secrets describe SECRET_NAME

# Verify Cloud Run SA has access
gcloud secrets get-iam-policy SECRET_NAME

# If missing, grant access:
PROJECT_NUMBER=$(gcloud projects describe $GCP_PROJECT_ID --format="value(projectNumber)")
gcloud secrets add-iam-policy-binding SECRET_NAME \
  --member="serviceAccount:${PROJECT_NUMBER}-compute@developer.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"
```
