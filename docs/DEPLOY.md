# BusinessOS Deploy Runbook

Ordered, copy-pasteable procedure to ship all three artifacts and verify the live system.
Backend goes to Cloud Run, frontend to Cloudflare Pages, desktop to a macOS DMG.

> Conventions: `$` lines are commands. Do not paste secret values into this file or into
> shell history; pipe them from a file or `openssl rand`. No command here runs automatically.

## 0. Infra reference (no secrets)

| Thing | Value |
|-------|-------|
| GCP project | `business-os-481523` |
| Cloud Run service | `businessos-api` (region `us-central1`) |
| Backend public host | `https://api.businessos.dev` (routed to Cloud Run) |
| Backend image | `gcr.io/business-os-481523/businessos-api` |
| Frontend (web) | `https://app.businessos.dev` (Cloudflare Pages, project `businessos`) |
| Session cookie | `Domain=.businessos.dev` (shared across `app.` and `api.`) |
| Cloud SQL instance | `business-os-481523:us-central1:businessos-db` |
| Backend Dockerfile | `desktop/backend-go/Dockerfile` (binds `SERVER_PORT`, default 8080) |
| Cloud Build config | `desktop/backend-go/cloudbuild.yaml` |
| Frontend proxy | `frontend/functions/api/[[path]].js` (Pages Function) |

### Required backend environment

Set as Cloud Run env vars (non-secret) and Secret Manager secrets. Production
`config.Validate()` (`desktop/backend-go/internal/config/config_helpers.go`) **crashes the
service at boot** if any required secret below is missing.

Plain env vars (set by `cloudbuild.yaml`):

| Var | Value | Why |
|-----|-------|-----|
| `ENVIRONMENT` | `production` | turns on prod validation + secure cookies |
| `SERVER_PORT` | `8080` | backend binds `SERVER_PORT`, Cloud Run injects `PORT` (ignored), so 8080 must match the Dockerfile `EXPOSE` |
| `COOKIE_DOMAIN` | `.businessos.dev` | session cookie must be shared across `app.`/`api.` or **login does not stick** |
| `GOOGLE_REDIRECT_URI` | `https://api.businessos.dev/api/auth/google/callback/login` | must match the Google console authorized redirect URI |
| `ALLOWED_ORIGINS` | `https://app.businessos.dev,app://localhost,http://localhost:5173` | prod CORS allowlist (no wildcard, validator rejects `*`) |
| `AI_PROVIDER` | `anthropic` | |
| `ENABLE_LOCAL_MODELS` | `false` | |

Secrets in Secret Manager (all `:latest`):

| Secret | Constraint |
|--------|-----------|
| `DATABASE_URL` | not localhost, no `CHANGE_ME` |
| `GOOGLE_CLIENT_ID` | |
| `GOOGLE_CLIENT_SECRET` | |
| `SECRET_KEY` | >= 32 chars (64 recommended) |
| `ANTHROPIC_API_KEY` | |
| `TOKEN_ENCRYPTION_KEY` | >= 32 chars, REQUIRED in prod |
| `REDIS_KEY_HMAC_SECRET` | >= 32 chars, REQUIRED in prod |
| `INTERNAL_API_SECRET` | >= 32 chars, REQUIRED in prod |
| `WEBHOOK_SIGNING_SECRET` | >= 16 chars, REQUIRED in prod |

One-time secret creation (skip any that already exist):

```bash
$ for s in DATABASE_URL GOOGLE_CLIENT_ID GOOGLE_CLIENT_SECRET SECRET_KEY \
    ANTHROPIC_API_KEY TOKEN_ENCRYPTION_KEY REDIS_KEY_HMAC_SECRET \
    INTERNAL_API_SECRET WEBHOOK_SIGNING_SECRET; do
    gcloud secrets describe "$s" --project business-os-481523 >/dev/null 2>&1 \
      || echo "MISSING: $s (create with: gcloud secrets create $s --data-file=-)"
  done
```

### Frontend environment

Set in the Cloudflare Pages dashboard (project `businessos`):

| Var | Value | Used by |
|-----|-------|---------|
| `CLOUDFLARE_BUILD` | `true` | build command, selects the static adapter |
| `PUBLIC_ENVIRONMENT` | `production` | |
| `BUSINESSOS_BACKEND_URL` | `https://api.businessos.dev` | optional override for the Pages Function; defaults to the Cloud Run URL baked into `[[path]].js` |

The browser always calls the same origin (`/api/*`); the Pages Function proxies to the
backend, so the session cookie stays first-party. `VITE_BACKEND_URL` / `VITE_API_URL`
should be left UNSET for the web build (the runtime resolver returns `/api/v1`).

---

## A. Apply pending migrations to the cloud database

Migrations live in `desktop/backend-go/internal/database/migrations/`. The backend can
auto-apply on boot, but apply explicitly first so a bad migration fails before the new
revision serves traffic.

```bash
# From a host that can reach Cloud SQL (Cloud SQL Auth Proxy or an authorized network).
$ cd desktop/backend-go
# Inspect what is pending (compare files against the applied set in the DB).
$ ls internal/database/migrations/ | sort
# Apply with your migration tool of record (whatever the team uses, e.g. the proxy + psql,
# or the backend's built-in migrator). Confirm 104/105 (superadmin seed + autopromote)
# are present so roberto@businessos.dev becomes superadmin.
```

Verify the superadmin row exists after migrating:

```bash
$ psql "$DATABASE_URL" -c \
  "select email, role from users where email = 'roberto@businessos.dev';"
# Expect role = superadmin (or platform_admin per 104/105).
```

---

## B. Deploy backend to Cloud Run

```bash
$ cd desktop/backend-go
# Cloud Build builds desktop/backend-go/Dockerfile, pushes the image, and deploys.
$ gcloud builds submit --config cloudbuild.yaml --project business-os-481523 .
```

`cloudbuild.yaml` sets all env vars + secrets listed above and deploys with
`--allow-unauthenticated`, `--add-cloudsql-instances`, memory 512Mi, cpu 1,
min 0 / max 10 instances, concurrency 80.

Confirm the revision is serving and reachable through the routed host:

```bash
$ gcloud run services describe businessos-api \
    --region us-central1 --project business-os-481523 \
    --format='value(status.url, status.latestReadyRevisionName)'
$ curl -fsS https://api.businessos.dev/health && echo OK
```

If the new revision never becomes ready, check logs for a config validation panic
(missing secret) — that is the most common boot failure:

```bash
$ gcloud run services logs read businessos-api \
    --region us-central1 --project business-os-481523 --limit 50
```

---

## C. Build + deploy frontend to Cloudflare Pages

Preferred path is the Cloudflare Pages Git integration (build command
`CLOUDFLARE_BUILD=true npm run build`, output dir `build`, set the env vars above, then
push to the production branch). To deploy manually with Wrangler:

```bash
$ cd frontend
$ CLOUDFLARE_BUILD=true npm run build
# Output dir is build/ (pages_build_output_dir in wrangler.toml).
$ npx wrangler pages deploy build --project-name=businessos
```

The `frontend/functions/` directory is uploaded with the static assets and becomes the
Pages Function that proxies `/api/*`. Confirm both the SPA and the proxy:

```bash
$ curl -fsS -o /dev/null -w '%{http_code}\n' https://app.businessos.dev/        # 200
$ curl -fsS https://app.businessos.dev/api/v1/health && echo PROXY_OK           # via Function -> Cloud Run
```

---

## D. Build the desktop DMG

Build the frontend (Electron static output) and the Go sidecar binary, then make the DMG.
Run on macOS for a macOS DMG.

```bash
$ cd desktop
$ npm install
$ npm run build:all        # build:frontend (ELECTRON_BUILD=true -> src/renderer)
                           # + build:backend (cross-compiled -> resources/bin/<platform>)
$ npm run make             # electron-forge make -> DMG under out/make/
```

What gets bundled:
- Renderer: the ELECTRON_BUILD SvelteKit static build copied to `desktop/src/renderer`,
  picked up by the Forge Vite plugin (`vite.renderer.config.ts`).
- Backend sidecar: `resources/bin/darwin-arm64/businessos-server` (and `darwin-x64`),
  shipped via `packagerConfig.extraResource` and spawned by `BackendManager`.
- Native modules (`better-sqlite3`, `electron-store`, `node-pty`) copied by the
  `packageAfterCopy` hook and unpacked by `AutoUnpackNativesPlugin`.

The DMG appears in `desktop/out/make/`.

Code signing + notarization run only if `APPLE_ID` (and `APPLE_IDENTITY`, `APPLE_PASSWORD`,
`APPLE_TEAM_ID`) are exported; otherwise the DMG is unsigned (fine for local testing,
Gatekeeper will warn end users).

---

## E. Smoke test (do this after every deploy)

1. Backend health:
   ```bash
   $ curl -fsS https://api.businessos.dev/health && echo OK
   ```
2. Proxy health (web origin -> Function -> Cloud Run):
   ```bash
   $ curl -fsS https://app.businessos.dev/api/v1/health && echo OK
   ```
3. Login (browser):
   - Open `https://app.businessos.dev`, sign in (Google or email).
   - After login, DevTools > Application > Cookies: confirm the session cookie has
     `Domain=.businessos.dev`, `Secure`, `HttpOnly`. If the cookie is missing a Domain,
     `COOKIE_DOMAIN` was not set on the backend (see step B / the env table).
   - Confirm OAuth round-trips without a redirect_uri mismatch (that means
     `GOOGLE_REDIRECT_URI` matches the Google console).
4. Superadmin visibility:
   - Log in as `roberto@businessos.dev`. The `/admin` route must be visible.
   - If not, re-check migrations 104/105 applied (step A).
5. Desktop:
   - Open the built app from `desktop/out/make/`. The Go sidecar should start
     (no "Failed to start Go backend" in the console); the app should reach the cloud
     backend and complete login.

---

## Deploy bugs found and fixed

Fixed in this pass (in `desktop/backend-go/cloudbuild.yaml`):

1. **Missing required secrets — boot crash.** cloudbuild wired only 5 secrets, but
   production `config.Validate()` requires `TOKEN_ENCRYPTION_KEY`,
   `REDIS_KEY_HMAC_SECRET`, `INTERNAL_API_SECRET`, and `WEBHOOK_SIGNING_SECRET`.
   Without them the service panics at boot. Added all four to `--set-secrets`
   (`cloudbuild.yaml:78`) and to the prerequisites comment.
2. **`GOOGLE_REDIRECT_URI` malformed / wrong host.** Was
   `https://${_SERVICE_NAME}-${PROJECT_NUMBER}.${_REGION}.run.app/...`. `${PROJECT_NUMBER}`
   is not a Cloud Build built-in substitution, so it expanded empty and produced a
   broken URI on the raw run.app host. Changed to
   `https://api.businessos.dev/api/auth/google/callback/login` (`cloudbuild.yaml:84`).
3. **`ALLOWED_ORIGINS` pointed at the wrong domain.** Was `https://businessos.app,...`;
   the live web app is `app.businessos.dev`. Changed to
   `https://app.businessos.dev,app://localhost,http://localhost:5173`
   (`cloudbuild.yaml:89`).
4. **`COOKIE_DOMAIN` never set.** The backend reads `COOKIE_DOMAIN` to scope the session
   cookie to `.businessos.dev`; unset means the cookie has no Domain and does not survive
   the `app.` <-> `api.` hop, so login silently fails. Added
   `COOKIE_DOMAIN=.businessos.dev` to `--set-env-vars` (`cloudbuild.yaml:73`).

Flagged, NOT changed (out of my file ownership or not a clear bug):

- **`desktop/forge.config.ts:141-142`** — `PublisherGithub` repo is the placeholder
  `your-org/businessos-desktop`. Only affects `forge publish`, not `forge make` (the DMG
  build), so it does not block the DMG. Fix before using GitHub auto-publish.
- **`frontend/functions/api/[[path]].js`** — proxy logic is correct (forwards
  path+query, drops `Host`, streams the body with `duplex: half`, passes `Set-Cookie`
  through). No change. If a future content-encoding mismatch appears, strip
  `content-encoding`/`content-length` from the forwarded response headers.
- **Cloud Run `PORT` vs `SERVER_PORT`** — Cloud Run injects `PORT`; the backend binds
  `SERVER_PORT`. It works only because cloudbuild hardcodes `SERVER_PORT=8080`. Kept the
  explicit value and documented it. A backend-owned fix would be to fall back to `PORT`.
