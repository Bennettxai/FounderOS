# BusinessOS - Go-Live SOP

The runbook for shipping BusinessOS to production.
Follow it top to bottom every time.
The whole point of this document is that nothing gets forgotten - especially the desktop DMG, which is not part of the web deploy and has been missed before.

## The golden rule

A release is not "live" until ALL FOUR are shipped and verified:
1. Backend (Cloud Run)
2. Frontend (Cloudflare Pages)
3. Desktop app (DMG rebuilt + uploaded)
4. Code pushed to all repos (origin + both agency mirrors, on `main`)

Web-only is not a release.
If you shipped the web and not the DMG, you are not done.

## 0. Pre-flight (do this first, every time)

Auth expires constantly - check it before you start, not mid-deploy.

```
gcloud auth list --filter=status:ACTIVE --format='value(account)'   # must show roberto@lunivate.com
gcloud config get-value project                                      # must be business-os-481523
npx wrangler whoami                                                   # must be logged in
```

If gcloud is expired: `gcloud auth login` (interactive - the deploy will silently fail otherwise).

Then confirm the code is sound:
```
cd desktop/backend-go && go build ./...        # must be clean
cd ../../frontend && npx svelte-check --threshold error   # must be 0 errors
```

## 1. Commit

Commit the change on `businessossync` with a clear message.
No `Co-Authored-By` line. No em dashes.

## 2. Backend -> Cloud Run

Use source-deploy. Do NOT use `cloudbuild.yaml` - its `gcr.io` push is IAM-denied and it silently fails.

```
gcloud run deploy businessos-api --source=desktop/backend-go --region=us-central1 --platform=managed
```

Source-deploy preserves existing env vars + secrets, so a plain redeploy is safe.
Watch for `serving 100 percent of traffic` and a new revision number.

## 3. Frontend -> Cloudflare Pages

```
cd frontend
CLOUDFLARE_BUILD=true VITE_API_URL="https://api.businessos.dev/api/v1" VITE_BACKEND_URL="https://api.businessos.dev" npm run build
npx wrangler pages deploy build --project-name=businessos-5 --branch=main --commit-dirty=true
```

The project is `businessos-5`, NOT `businessos`.
`--branch=main` makes it the production deployment (serves businessos.dev + app.businessos.dev).

## 4. Desktop DMG (THE STEP THAT GETS FORGOTTEN)

The DMG is a separate build. The web deploy does nothing to it.
Rebuild it whenever backend or frontend behavior the desktop app depends on changes.

```
cd desktop
npm run build                                  # or the forge package step
# package the .app -> .dmg (make is broken; use forge package + hdiutil)
```

Then upload and update the download link:
```
gsutil cp <built>.dmg gs://businessos-downloads/BusinessOS-<version>-arm64.dmg
# build + upload the x64 (Intel) DMG too - the landing page links arm64 only today
```

The landing-page download button is in `frontend/src/routes/+page.svelte` (points at `gs://businessos-downloads/...`).
Update the version/filename there if it changed, then re-run step 3.

Signing: SOLVED as of v1.0.1. The DMG is signed + notarized with the MIOSA LLC
Developer ID (Team `5UL79JP67U`). Credentials live in gitignored
`desktop/.env.signing` (APPLE_ID, APPLE_PASSWORD app-specific, APPLE_TEAM_ID,
APPLE_IDENTITY). To ship a signed release:
```
cd desktop && set -a && source .env.signing && set +a
npm run build:all && npx electron-forge package && npx electron-forge package --arch=x64
# forge signs + notarizes + staples the .app for each arch. Then per arch:
#   stage .app + /Applications symlink -> hdiutil create ...-<arch>.dmg
#   codesign --sign "$APPLE_IDENTITY" <dmg>
#   xcrun notarytool submit <dmg> --apple-id .. --team-id .. --password .. --wait
#   xcrun stapler staple <dmg>
# verify: spctl -a -vvv -t install <dmg>  (must say "accepted / Notarized Developer ID")
```
Notarization takes ~5-15 min per submission on Apple's side (both .app and .dmg).

## 5. Database migrations (only if the change adds/alters tables)

Local dev DB is separate from cloud.
Apply new migrations to the CLOUD DB explicitly - do not assume they ran.
Some tables self-bootstrap on first use (e.g. `knowledge_documents` via the sync endpoint); most do not.

```
CU=$(grep -E '^CLOUD_DATABASE_URL=' desktop/backend-go/.env | cut -d= -f2- | tr -d '"')
psql "$CU" -v ON_ERROR_STOP=1 -f desktop/backend-go/internal/database/migrations/<n>_<name>.sql
```

## 6. Push to all repos (on `main`)

The team pulls `robertohluna/agency-miosa-businessos` `main`.
Keep all three in sync.

```
git push origin businessossync            # Miosa-osa/businessos-5
git push origin businessossync:main       # fast-forward main
git push agency businessossync            # robertohluna/agency-miosa-businessos
git push agency-org businessossync        # Miosa-osa/agency-miosa-businessos
```

The agency repos carry extra agency content, so their `main` may need a real merge (in the `/Users/rhl/code/agency-miosa-businessos` clone) instead of a fast-forward - merge `businessossync`, keep agency content, keep canonical code, push `main` to both agency remotes.
Never force-push the agency repos.

## 7. Verify (prove it, don't assume)

```
# backend revision + a NEW endpoint from this release (401 = wired, 404 = old code still serving)
curl -s -o /dev/null -w '%{http_code}\n' https://businessos.dev/api/<new-endpoint>

# frontend fingerprint matches the deployment you just pushed
curl -s https://businessos.dev/ | grep -oE 'app\.[A-Za-z0-9_-]+\.js' | head -1

# health
curl -s -o /dev/null -w '%{http_code}\n' https://businessos-api-4lama7hpmq-uc.a.run.app/health   # 200
```

If you changed an authed flow, actually log in (web) and click it. A 200 is not proof it works.

## Landmines (learned the hard way)

- `api.businessos.dev` is DEAD (no Cloud Run domain mapping, returns 404). The backend is reached via `businessos.dev/api/*` through the Cloudflare Pages Function proxy (`functions/api/[[path]].js`). Any redirect/callback URL must use `businessos.dev`, not `api.businessos.dev`.
- Google OAuth: `GOOGLE_REDIRECT_URI` must be `https://businessos.dev/api/v1/auth/oauth/google/callback` AND that exact URI must be in the OAuth client `460433387676` authorized redirect URIs in Google Console (console-only, no gcloud/API for it). Localhost redirects are already authorized, so local dev + Google works.
- `ALLOWED_ORIGINS` must include `https://businessos.dev` (not just `app.`).
- gcloud auth expires often and fails SILENTLY in non-interactive deploys - always check in pre-flight.
- Never start the bundled `optimal-engine` (dev-local.sh is guarded) - it squats `:4200` and clobbers Roberto's real engine.
- Never `DROP SCHEMA` / reset the live DB casually.
- Session cookie is `better-auth.session_token`, `Domain=.businessos.dev` - the proxy forwards it, so app <-> api share the session.

## Rollback

Backend: `gcloud run services update-traffic businessos-api --region=us-central1 --to-revisions=<prev-revision>=100`
Frontend: redeploy the previous build, or roll back in the Cloudflare Pages dashboard.

## Versioning (added v1.0.1)

- The version lives in desktop/package.json + frontend/package.json (keep them equal).
- Every release: bump PATCH (1.0.1 -> 1.0.2), add a CHANGELOG.md section, tag the
  release commit `v<version>` and push the tag. DMG filenames carry the version
  (BusinessOS-<version>-<arch>.dmg) - update the landing page download links when it changes.
- MINOR for feature waves, MAJOR for breaking changes.

## Branded DMG build (headless, the working method)

`electron-forge make`'s MakerDMG uses `appdmg`, whose native `volume.node` is
broken in this repo, and `create-dmg` needs Finder/AppleScript (times out in a
non-GUI shell). The reliable headless path is **dmgbuild** (writes the .DS_Store
layout directly - no Finder):

```
# one-time: python3 -m pip install --user --break-system-packages dmgbuild ds_store mac_alias
# per release, AFTER forge has packaged + signed + notarized the .app:
APP_PATH=out/BusinessOS-darwin-<arch>/BusinessOS.app \
ICON=resources/icons/icon.icns BG=resources/dmg-background.png \
~/Library/Python/*/bin/dmgbuild -s /tmp/dmg_settings.py "BusinessOS" out/BusinessOS-<ver>-<arch>.dmg
# then: codesign --sign "$APPLE_IDENTITY" <dmg>; notarytool submit --wait; stapler staple
```

dmg_settings.py sets window 720x540, icon-size 128, app at (200,280),
Applications link at (520,280) - matching the 1440x1080 branded background
(resources/dmg-background.png, regenerated from /tmp/dmgbg.svg via rsvg-convert).

## Updating the bundled engine (before a DMG rebuild)

BusinessOS ships a VENDORED copy of OptimalEngine at `optimal-engine/` (not a git
submodule). The canonical source is the OptimalOS engine submodule at
`/Users/rhl/code/OptimalOS/engine`. When the canonical engine moves forward, refresh
the bundled copy BEFORE staging + rebuilding the DMG:

```
# 1. Sync canonical -> vendored, preserve the env-driven runtime.exs, build the release.
#    (Idempotent. Does NOT start the engine. Accepts an alt canonical dir as arg1.)
scripts/sync-engine.sh                          # or: scripts/sync-engine.sh /path/to/OptimalEngine

# 2. Stage the freshly built prod release into desktop/resources/engine/<platform>-<arch>/.
desktop/scripts/stage-engine.sh                 # defaults to host arch

# 3. Rebuild the DMG (see "Branded DMG build" above) so it carries the new engine.
```

Why the runtime.exs dance: the vendored `optimal-engine/config/runtime.exs` is env-driven
with NO personal paths, so a downloaded user gets a fresh self-contained engine. The
canonical `runtime.exs` is a developer-private overlay that hardcodes Roberto's workspace
path. `sync-engine.sh` backs up the vendored runtime.exs, rsyncs canonical over the top
(excluding `_build`, `deps`, `.git`, `node_modules`, `*.db`, `.optimal`, `tmp/`), then
restores the vendored runtime.exs - so the personal path is never re-leaked into the bundle.
