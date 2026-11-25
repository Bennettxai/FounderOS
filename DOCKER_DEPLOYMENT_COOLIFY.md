# FounderOS (on GladOS) - Docker Deployment for Coolify

This guide covers deploying FounderOS as an all-in-one container using Docker Compose, optimized for Coolify.

## Quick Start with Coolify

### 1. Prepare Your Repository

Push your code to a Git repository (GitHub, GitLab, etc.) that Coolify can access.

### 2. Create a New Application in Coolify

1. In Coolify, click **New Resource** → **Application**
2. Select your Git repository
3. Set **Build Pack** to **Docker Compose**
4. Set **Docker Compose File** to `docker-compose.yml`
5. Set **Base Directory** to `/` (root)

### 3. Configure Environment Variables

In Coolify's **Environment Variables** section, add the variables from `.env.docker.example`. At minimum, you need:

**Required for first deploy:**

- `NEXT_PUBLIC_APP_URL` - Your Coolify domain (e.g., `https://founderos.yourdomain.com`)
- `GLADOS_BRAND` - `founderos` or `glados`
- `GLADOS_DEFAULT_MODE` - `demo` (recommended for first deploy)

**Optional but recommended:**

- `INBOX_1_HOST`, `INBOX_1_USER`, `INBOX_1_PASS`, `INBOX_1_NAME` - For email integration
- `SLACK_BOT_TOKEN` - For Slack integration
- `STRIPE_SECRET_KEY` - For payments
- `AI_GATEWAY_API_KEY` - For AI chat features
- `APIFY_API_TOKEN` - For Content Lab

**Note for local testing**: The `docker-compose.yml` includes `env_file: .env` so you can test locally with a `.env` file before pushing to Coolify. Coolify ignores this and uses its own environment variables UI.

### 4. Configure Persistent Volumes

Coolify automatically creates Docker volumes for the services. The following data will persist across deployments:

| Volume                 | Purpose                                |
| ---------------------- | -------------------------------------- |
| `founderos-data`       | SQLite database, uploads, all app data |
| `founderos-brain`      | G-Brain knowledge store                |
| `founderos-creative`   | Creative Engine video/assets           |
| `founderos-automation` | Scheduled agent jobs                   |
| `founderos-mode`       | Demo/Real mode toggle                  |

### 5. Deploy

Click **Deploy** and wait for the build to complete. The first build takes 3-5 minutes.

### 6. Initial Setup

1. Open your deployed URL (e.g., `https://founderos.yourdomain.com`)
2. You'll be redirected to `/setup`
3. Set an **operator password** under **Security**
4. The app will generate `GLADOS_PASSWORD_HASH` and `GLADOS_SESSION_SECRET` automatically
5. Configure your integrations under **Settings → Keys**

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Coolify / Docker Host                  │
│                                                             │
│  ┌─────────────────────────────────────────────────────┐    │
│  │              founderos container                    │    │
│  │  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │    │
│  │  │ Next.js App  │  │ SQLite DB    │  │ File      │  │    │
│  │  │ (port 4100)  │  │ (/data)      │  │ Storage   │  │    │
│  │  └──────────────┘  └──────────────┘  └───────────┘  │    │
│  └─────────────────────────────────────────────────────┘    │
│           │                    │                   │        │
│           ▼                    ▼                   ▼        │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │ founderos-   │  │ founderos-   │  │ founderos-        │  │
│  │ data         │  │ brain        │  │ creative          │  │
│  │ (volume)     │  │ (volume)     │  │ (volume)          │  │
│  └──────────────┘  └──────────────┘  └───────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Important Notes

### Security

- **Never commit `.env.local` or secrets** - Use Coolify's environment variables UI
- The app **refuses to bind to non-loopback addresses** until a password is set
- First deploy must be accessed via the Coolify domain to set the password

### Data Persistence

All data lives in Docker volumes:

- SQLite database: `/data/glados.db`
- Demo data (filming mode): `/data/demo-data/`
- Real data: `/data/data/`
- Brain store: `/data/brain-store/`
- Creative assets: `/app/public/content-media/creative-engine/`

### Demo vs Real Mode

- **Demo mode** (default): Uses `demo-data/` with sample company data
- **Real mode**: Uses `data/` for your actual business data
- Toggle in **Settings → General → Demo Mode**
- The `.glados-mode` file persists this choice

### Health Checks

The container includes a health check hitting `/api/auth/check`. Coolify will:

- Wait up to 40s for the app to start
- Check every 30s
- Restart if 3 consecutive checks fail

### Resource Limits

Default limits in `docker-compose.yml`:

- Memory limit: 2GB
- Memory reservation: 512MB

Adjust in `deploy.resources` if needed.

## Updating

1. Push changes to your Git repository
2. In Coolify, click **Deploy** on the application
3. Coolify will rebuild and deploy with zero-downtime (if configured)

## Troubleshooting

### Build Fails

- Check build logs in Coolify
- Ensure all `npm ci` dependencies are available
- Native modules (better-sqlite3) need Python, make, g++ (included in Dockerfile)
- **Static generation errors on dev pages**: The `next.config.mjs` includes `staticPageGenerationTimeout: 120` and dev-only pages (`/dev/g-kit`, `/lab-frames/[state]`) use `export const dynamic = 'force-dynamic'` to skip static generation. If you add similar pages, add `dynamic = 'force-dynamic'` to avoid build failures.

### App Won't Start

- Check container logs in Coolify
- Verify `NEXT_PUBLIC_APP_URL` matches your domain
- Ensure volumes have correct permissions (handled by Dockerfile)

### Database Issues

- Data persists in `founderos-data` volume
- To reset: delete the volume in Coolify and redeploy
- Demo data rebuilds automatically on first start

### Port Conflicts

- Container exposes port 4100 internally
- Coolify maps this to your domain automatically
- Don't change `PORT=4100` in environment variables

## Advanced Configuration

### Custom Domain with SSL

Coolify handles SSL automatically via Let's Encrypt. Just set your domain in the application settings.

### Multiple Instances

For high availability, deploy multiple instances behind a load balancer:

- Set `GLADOS_CRON=1` on **only one** instance
- Use shared volumes (NFS, Longhorn, etc.) for data
- Configure session affinity if needed

### Backup Strategy

```bash
# Backup SQLite database
docker exec founderos sqlite3 /data/glados.db .dump > backup.sql

# Backup volumes
docker run --rm -v founderos-data:/data -v $(pwd):/backup alpine tar czf /backup/data-backup.tar.gz -C /data .
```

## Support

- Check logs in Coolify's **Logs** tab
- Application logs: `docker logs founderos`
- Health endpoint: `https://your-domain.com/api/auth/check`
- GitHub Issues: [FounderOS Issues](https://github.com/Bennettxai/FounderOS/issues)
