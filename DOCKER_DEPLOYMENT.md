# FounderOS (on GladOS) - Generic Docker Deployment

This guide covers deploying FounderOS using Docker Compose on any platform.

## Quick Start

### 1. Prerequisites

- Docker Engine 24+
- Docker Compose v2+ (or `docker-compose` v1)
- 2GB+ RAM available

### 2. Configure Environment

```bash
# Copy the example environment file
cp .env.docker.example .env

# Edit with your values
nano .env
```

**Minimum required variables:**

```env
NEXT_PUBLIC_APP_URL=http://your-domain.com
GLADOS_BRAND=founderos
GLADOS_DEFAULT_MODE=demo
```

**For local Docker Desktop development**, you must also set a password in `.env` before building (the app refuses to bind to `0.0.0.0` without one):

```env
# Generate with: node -e "const crypto=require('crypto'); const p='yourpass'; const s=crypto.randomBytes(16).toString('hex'); const h=crypto.pbkdf2Sync(p,s,100000,64,'sha512').toString('hex'); console.log('GLADOS_PASSWORD_HASH='+s+':'+h); console.log('GLADOS_SESSION_SECRET='+crypto.randomBytes(32).toString('hex'))"
GLADOS_PASSWORD_HASH=...
GLADOS_SESSION_SECRET=...
```

### 3. Deploy

```bash
# Build and start
docker compose up -d --build

# View logs
docker compose logs -f

# Stop
docker compose down
```

### 4. Initial Setup

1. Open `http://your-domain.com:4100` (or your configured domain)
2. You'll be redirected to `/setup`
3. Set an **operator password** under **Security** (or use the one from `.env`)
4. The app generates `GLADOS_PASSWORD_HASH` and `GLADOS_SESSION_SECRET` automatically (if not pre-set)
5. Configure integrations under **Settings → Keys**

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                      Docker Host                           │
│                                                            │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              founderos container                    │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │   │
│  │  │ Next.js App  │  │ SQLite DB    │  │ File      │  │   │
│  │  │ (port 4100)  │  │ (/data)      │  │ Storage   │  │   │
│  │  └──────────────┘  └──────────────┘  └───────────┘  │   │
│  └─────────────────────────────────────────────────────┘   │
│           │                    │                    │      │
│           ▼                    ▼                    ▼      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐  │
│  │ founderos-   │  │ founderos-   │  │ founderos-       │  │
│  │ data         │  │ brain        │  │ creative         │  │
│  │ (volume)     │  │ (volume)     │  │ (volume)         │  │
│  └──────────────┘  └──────────────┘  └──────────────────┘  │
└────────────────────────────────────────────────────────────┘
```

## Persistent Volumes

| Volume | Mount Path | Purpose |
| --- | --- | --- |
| `founderos-data` | `/data` | SQLite database, uploads, all app data |
| `founderos-brain` | `/data/brain-store` | G-Brain knowledge store |
| `founderos-creative` | `/app/public/content-media/creative-engine` | Creative Engine video/assets |
| `founderos-automation` | `/data/Automation` | Scheduled agent jobs |
| `founderos-mode` | `/app/.glados-mode` | Demo/Real mode toggle |

## Data Persistence

All data lives in Docker volumes:

- SQLite database: `/data/glados.db`
- Demo data (filming mode): `/data/demo-data/`
- Real data: `/data/data/`
- Brain store: `/data/brain-store/`
- Creative assets: `/app/public/content-media/creative-engine/`

## Demo vs Real Mode

- **Demo mode** (default): Uses `demo-data/` with sample company data
- **Real mode**: Uses `data/` for your actual business data
- Toggle in **Settings → General → Demo Mode**
- The `.glados-mode` file persists this choice

## Health Checks

The container includes a health check hitting `/api/auth/check`:

- Waits up to 40s for the app to start
- Checks every 30s
- Restarts if 3 consecutive checks fail

## Resource Limits

Default limits in `docker-compose.yml`:

- Memory limit: 2GB
- Memory reservation: 512MB

Adjust in `deploy.resources` if needed.

## Platform-Specific Notes

### Docker Compose (Standalone)
```bash
docker compose up -d --build
```

### Docker Desktop (Local Development)

For local development on Docker Desktop, the container needs to bind to `0.0.0.0` for port forwarding to work. The app refuses this without a password. Two options:

**Option A: Pre-set password in `.env` (recommended for local dev)**
```bash
# 1. Generate password hash
node -e "const crypto=require('crypto'); const p='yourpass'; const s=crypto.randomBytes(16).toString('hex'); const h=crypto.pbkdf2Sync(p,s,100000,64,'sha512').toString('hex'); console.log('GLADOS_PASSWORD_HASH='+s+':'+h); console.log('GLADOS_SESSION_SECRET='+crypto.randomBytes(32).toString('hex'))"

# 2. Add to .env
GLADOS_PASSWORD_HASH=...
GLADOS_SESSION_SECRET=...

# 3. Build and run
docker compose up -d --build
```

**Option B: Use loopback only (no password needed)**
```yaml
# In docker-compose.yml, change:
ports:
  - "127.0.0.1:4100:4100"  # Only accessible from host
environment:
  - GLADOS_HOST=127.0.0.1
```
Then access at `http://localhost:4100` and set password via UI at `/setup`.

### Docker Swarm

```bash
docker stack deploy -c docker-compose.yml founderos
```

### Kubernetes (via Kompose)

```bash
kompose convert -f docker-compose.yml
kubectl apply -f .
```

### Portainer

1. Stacks → Add Stack
2. Paste `docker-compose.yml` content
3. Add environment variables
4. Deploy

### Rancher

1. Apps → Launch
2. Select "Custom" → Paste compose
3. Configure env vars
4. Launch

## Updating

```bash
# Pull latest code
git pull

# Rebuild and restart
docker compose up -d --build

# Or for zero-downtime (if using load balancer)
docker compose up -d --build --no-deps founderos
```

## Troubleshooting

### Build Fails

- Check `docker compose build` output
- Ensure all `npm ci` dependencies are available
- Native modules (better-sqlite3) need Python, make, g++ (included in Dockerfile)

### App Won't Start

- Check logs: `docker compose logs founderos`
- Verify `NEXT_PUBLIC_APP_URL` matches your domain
- Ensure volumes have correct permissions (handled by Dockerfile)
- **If you see "refuses to listen on 0.0.0.0: no operator password is set"**: You must either:
  1. Set `GLADOS_PASSWORD_HASH` and `GLADOS_SESSION_SECRET` in `.env` (see Docker Desktop section above), OR
  2. Change `GLADOS_HOST=127.0.0.1` and use `ports: "127.0.0.1:4100:4100"` in docker-compose.yml

### Database Issues

- Data persists in `founderos-data` volume
- To reset: `docker compose down -v && docker compose up -d --build`
- Demo data rebuilds automatically on first start

### Port Conflicts

- Container exposes port 4100 internally
- Change host port in `docker-compose.yml`: `"8080:4100"`
- Don't change `PORT=4100` in environment variables

## Backup Strategy

```bash
# Backup SQLite database
docker exec founderos sqlite3 /data/glados.db .dump > backup.sql

# Backup all volumes
docker run --rm \
  -v founderos-data:/data \
  -v founderos-brain:/brain \
  -v founderos-creative:/creative \
  -v founderos-automation:/automation \
  -v $(pwd):/backup \
  alpine tar czf /backup/full-backup-$(date +%Y%m%d).tar.gz -C /data . -C /brain . -C /creative . -C /automation .
```

## Security Notes

- **Never commit `.env` or secrets** - Use your platform's secret management
- The app **refuses to bind to non-loopback addresses** until a password is set
- First deploy must be accessed via the configured domain to set the password
- Run as non-root user (UID 1001) inside container

## Support

- Application logs: `docker compose logs founderos`
- Health endpoint: `http://your-domain.com:4100/api/auth/check`
- GitHub Issues: [FounderOS Issues](https://github.com/Bennettxai/FounderOS/issues)
