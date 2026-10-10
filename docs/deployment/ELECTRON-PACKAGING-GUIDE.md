# Electron Packaging & Distribution Guide

> **How to build, sign, and distribute the BusinessOS desktop app.**
> Covers: DMG (macOS), EXE (Windows), DEB/RPM (Linux), auto-updates, and how the frontend connects to the Go backend inside Electron.
>
> Last Updated: 2026-02-23
> Status: COMPLETE GUIDE (some setup steps pending first execution)

---

## Architecture: How Frontend Connects to Backend in Electron

### The Sidecar Model

The desktop app bundles the Go backend as a **sidecar process**. Electron spawns the Go binary on startup and proxies all API calls to it.

```
┌──────────────────────────────────────────────────────────┐
│                    Electron Shell                         │
│                                                          │
│  ┌─────────────────┐     ┌────────────────────────────┐  │
│  │   Main Process   │────▶│   BackendManager           │  │
│  │   (index.ts)     │     │   (backend/manager.ts)     │  │
│  │                  │     │                            │  │
│  │   - Window mgmt  │     │   - Spawns Go binary       │  │
│  │   - System tray  │     │   - Health checks (5s)     │  │
│  │   - IPC handler  │     │   - Auto-restart (3x)      │  │
│  │   - Auto-updater │     │   - Graceful shutdown       │  │
│  └─────────┬────────┘     └─────────┬──────────────────┘  │
│            │                        │                     │
│            │ IPC                    │ localhost:18080      │
│            ▼                        ▼                     │
│  ┌─────────────────┐     ┌────────────────────────────┐  │
│  │  Renderer        │────▶│   Go Backend Binary        │  │
│  │  (SvelteKit)     │     │   (businessos-server)      │  │
│  │                  │     │                            │  │
│  │  - All UI        │     │   - REST API               │  │
│  │  - Stores        │     │   - WebSocket              │  │
│  │  - Components    │     │   - SSE streaming          │  │
│  │                  │     │   - SQLite (local DB)      │  │
│  └──────────────────┘     └────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

### Key Connection Details

| Setting | Value | Source |
|---------|-------|--------|
| Backend port | `18080` | `desktop/src/main/backend/manager.ts:7` |
| Health endpoint | `GET /health` | `manager.ts:73` |
| Health check interval | 5 seconds | `manager.ts:8` |
| Startup timeout | 30 seconds | `manager.ts:9` |
| Max auto-restarts | 3 | `manager.ts:17` |
| Database mode | SQLite (local) | Env: `DATABASE_MODE=sqlite` |
| SQLite path | `~/Library/Application Support/BusinessOS/businessos.db` | `manager.ts:140` |
| Dev mode backend | `http://localhost:8000` (external) | `manager.ts:131` |

### Platform Detection (Frontend)

The SvelteKit frontend detects whether it's running in Electron or browser:

```typescript
// frontend/src/lib/utils/platform.ts
const isElectron = typeof window !== 'undefined' && 'electron' in window;

// Electron: window.electron API available (contextBridge)
// Web: window.electron undefined → all core modules still work
```

| Feature | Electron Mode | Web Mode |
|---------|--------------|----------|
| Backend connection | `localhost:18080` (sidecar) | Cloud Run URL |
| Database | SQLite (local, offline-first) | PostgreSQL (Cloud SQL) |
| External apps | BrowserView (embedded) | New tab |
| File system | Native FS access | No FS access |
| Offline mode | Full offline support | Online required |
| Auto-updates | electron-updater | N/A (always latest) |
| Quick chat | Cmd+Shift+Space popup | N/A |
| System tray | Native tray icon | N/A |

---

## Prerequisites

### For All Platforms

```bash
# Node.js 20+ and npm
node --version  # v20.x+
npm --version   # v10.x+

# Go 1.24+ (for building backend binary)
go version  # go1.24.1+

# Electron Forge (installed as dev dependency)
cd desktop && npm install
```

### For macOS (Code Signing + Notarization)

```bash
# Apple Developer account ($99/year)
# Required for:
#   - Code signing (users can open without Gatekeeper warnings)
#   - Notarization (Apple verifies no malware)
#   - DMG distribution outside Mac App Store

# Environment variables needed:
export APPLE_ID="your@email.com"
export APPLE_PASSWORD="app-specific-password"  # NOT your Apple ID password
export APPLE_TEAM_ID="XXXXXXXXXX"
export APPLE_IDENTITY="Developer ID Application: Your Name (XXXXXXXXXX)"
```

**How to get an app-specific password:**
1. Go to https://appleid.apple.com/account/manage
2. Sign in → Security → App-Specific Passwords
3. Generate one labeled "BusinessOS Notarization"

### For Windows (Code Signing)

```bash
# Windows code signing certificate (from DigiCert, Sectigo, etc.)
# Required for:
#   - SmartScreen won't block the installer
#   - Users trust the application

# Environment variables:
export WIN_CSC_LINK="path/to/certificate.pfx"
export WIN_CSC_KEY_PASSWORD="certificate-password"
```

### For Linux

No signing required. DEB and RPM packages work without certificates.

---

## Building the Go Backend Binary

The Go backend must be cross-compiled for each target platform BEFORE packaging Electron.

### Build for Current Platform

```bash
cd desktop/backend-go

# Build for your current OS/arch
CGO_ENABLED=0 go build -ldflags="-s -w" -o ../desktop/resources/bin/$(go env GOOS)-$(go env GOARCH)/businessos-server ./cmd/server
```

### Cross-Compile for All Platforms

```bash
cd desktop/backend-go

# macOS (Apple Silicon)
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" \
  -o ../desktop/resources/bin/darwin-arm64/businessos-server ./cmd/server

# macOS (Intel)
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" \
  -o ../desktop/resources/bin/darwin-x64/businessos-server ./cmd/server

# Windows (x64)
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" \
  -o ../desktop/resources/bin/win32-x64/businessos-server.exe ./cmd/server

# Linux (x64)
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" \
  -o ../desktop/resources/bin/linux-x64/businessos-server ./cmd/server
```

### Verify Binary Locations

After building, the `resources/bin/` directory should look like:

```
desktop/resources/bin/
├── darwin-arm64/
│   └── businessos-server        # macOS Apple Silicon
├── darwin-x64/
│   └── businessos-server        # macOS Intel
├── win32-x64/
│   └── businessos-server.exe    # Windows
└── linux-x64/
    └── businessos-server        # Linux
```

The `BackendManager` resolves the correct binary at runtime:
```typescript
// manager.ts:47-66
const platformDir = `${process.platform}-${goArch}`;
return path.join(this.resourcesPath, 'bin', platformDir, binaryName);
```

---

## Building the Desktop App

### Development Mode

```bash
cd desktop
npm install  # First time only
npm start    # Starts Electron in dev mode
```

In dev mode:
- Frontend served by Vite dev server (hot reload)
- Backend expected at `http://localhost:8000` (run separately)
- No Go binary needed in resources/

### Production Build (DMG / EXE / DEB)

```bash
cd desktop

# Install dependencies
npm install

# Build Go binaries first (see section above)
# Then package:

# macOS DMG (unsigned)
npm run make -- --platform=darwin

# macOS DMG (signed + notarized)
APPLE_ID="you@email.com" APPLE_PASSWORD="xxx" APPLE_TEAM_ID="xxx" \
  npm run make -- --platform=darwin

# Windows EXE (Squirrel installer)
npm run make -- --platform=win32

# Linux DEB + RPM
npm run make -- --platform=linux
```

### Output Locations

```
desktop/out/
├── make/
│   ├── BusinessOS.dmg                    # macOS installer
│   ├── squirrel.windows/x64/
│   │   ├── BusinessOS-Setup.exe          # Windows installer
│   │   └── BusinessOS-x.x.x-full.nupkg  # Windows update package
│   ├── deb/x64/
│   │   └── businessos_x.x.x_amd64.deb   # Debian/Ubuntu
│   └── rpm/x64/
│       └── businessos-x.x.x.x86_64.rpm  # Fedora/RHEL
└── BusinessOS-darwin-arm64/              # Unpacked app (for testing)
    └── BusinessOS.app/
```

---

## Electron Forge Configuration

The packaging is configured in `desktop/forge.config.ts`:

### Makers (Installer Formats)

| Maker | Platform | Output |
|-------|----------|--------|
| `MakerDMG` | macOS | `.dmg` with custom background |
| `MakerZIP` | macOS | `.zip` (for auto-update) |
| `MakerSquirrel` | Windows | `.exe` installer + `.nupkg` update |
| `MakerDeb` | Linux | `.deb` package |
| `MakerRpm` | Linux | `.rpm` package |

### Code Signing (macOS)

Configured via environment variables (not hardcoded):

```typescript
// forge.config.ts:23-37
osxSign: process.env.APPLE_ID ? {
  identity: process.env.APPLE_IDENTITY,
  entitlements: './resources/entitlements.mac.plist',
  hardenedRuntime: true,
} : undefined,

osxNotarize: process.env.APPLE_ID ? {
  appleId: process.env.APPLE_ID,
  appleIdPassword: process.env.APPLE_PASSWORD,
  teamId: process.env.APPLE_TEAM_ID,
} : undefined,
```

**Without signing:** Set no env vars. DMG will build but users will see "unidentified developer" warning.
**With signing:** Set all 4 env vars. DMG will be signed, notarized, and stapled.

### Auto-Update Publisher

```typescript
// forge.config.ts:88-96
publishers: [
  new PublisherGithub({
    repository: {
      owner: process.env.BUSINESSOS_RELEASE_OWNER || 'Miosa-osa',
      name: process.env.BUSINESSOS_RELEASE_REPO || 'businessos-5',
    },
    prerelease: false,
    draft: true,
  }),
],
```

The default release target is `Miosa-osa/businessos-5`.
Set `BUSINESSOS_RELEASE_OWNER` and `BUSINESSOS_RELEASE_REPO` only when publishing a fork or staging release.

---

## Auto-Update System

### How It Works

1. On startup (10s delay) → check GitHub Releases for newer version
2. Every 4 hours → check again
3. If update found → show dialog: "Download now?" / "Later"
4. User clicks Download → background download with progress
5. Download complete → show dialog: "Restart now?" / "Later"
6. User clicks Restart → `quitAndInstall()`
7. App restarts with new version

### Configuration

| Setting | Value | File |
|---------|-------|------|
| Check delay | 10 seconds after startup | `auto-update.ts:26` |
| Check interval | Every 4 hours | `auto-update.ts:31` |
| Auto-download | `false` (asks user first) | `auto-update.ts:12` |
| Auto-install on quit | `true` | `auto-update.ts:13` |
| Update source | GitHub Releases | `forge.config.ts:88-96` |

### Publishing an Update

```bash
# 1. Bump version in desktop/package.json
npm version patch  # or minor, or major

# 2. Build for all platforms
npm run make -- --platform=darwin
npm run make -- --platform=win32
npm run make -- --platform=linux

# 3. Generate macOS update metadata
npm run release:metadata

# 4. Publish to GitHub Releases
npm run publish

# 5. Upload latest-mac.yml to the same GitHub release
npm run release:upload-metadata

# 6. Edit the release on GitHub (add release notes, unmark draft)
```

The auto-updater uses `electron-updater` against GitHub Releases.
For macOS, the release must include the ZIP artifact and `latest-mac.yml`.
The DMG is for manual installs.
The ZIP is what the running macOS app downloads for auto-update.
When you publish a newer version with those artifacts, packaged desktop apps detect it and offer the update.

Users see updates in two places:

- A desktop-only update card in Settings with current version, channel, latest version, release notes, progress, and install controls.
- A small in-app notification when a packaged desktop build detects an available or downloaded update.

Set `BUSINESSOS_UPDATE_CHANNEL` to publish or test a channel other than `stable`.
Set `BUSINESSOS_MIN_SUPPORTED_VERSION` when an old desktop build must show a required update state.

---

## Creating Required Resources

### Icons

You need icons in multiple formats:

```
desktop/resources/icons/
├── icon.icns     # macOS (1024x1024)
├── icon.ico      # Windows (256x256 multi-res)
├── icon.png      # Linux (512x512)
└── icon.svg      # Source (optional)
```

**Generate from a single 1024x1024 PNG:**
```bash
# macOS: Use iconutil
mkdir icon.iconset
sips -z 16 16   icon-1024.png --out icon.iconset/icon_16x16.png
sips -z 32 32   icon-1024.png --out icon.iconset/icon_16x16@2x.png
sips -z 32 32   icon-1024.png --out icon.iconset/icon_32x32.png
sips -z 64 64   icon-1024.png --out icon.iconset/icon_32x32@2x.png
sips -z 128 128 icon-1024.png --out icon.iconset/icon_128x128.png
sips -z 256 256 icon-1024.png --out icon.iconset/icon_128x128@2x.png
sips -z 256 256 icon-1024.png --out icon.iconset/icon_256x256.png
sips -z 512 512 icon-1024.png --out icon.iconset/icon_256x256@2x.png
sips -z 512 512 icon-1024.png --out icon.iconset/icon_512x512.png
sips -z 1024 1024 icon-1024.png --out icon.iconset/icon_512x512@2x.png
iconutil -c icns icon.iconset -o desktop/resources/icons/icon.icns

# Windows: Use ImageMagick
convert icon-1024.png -define icon:auto-resize=256,128,64,48,32,16 \
  desktop/resources/icons/icon.ico
```

### DMG Background

```
desktop/resources/dmg-background.png
```

A 600x400 PNG image shown in the DMG window. Typically shows the app icon on the left, an arrow, and the Applications folder on the right.

### Entitlements (macOS)

```
desktop/resources/entitlements.mac.plist
```

Required for Hardened Runtime. Declares permissions the app needs:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>com.apple.security.cs.allow-jit</key>
  <true/>
  <key>com.apple.security.cs.allow-unsigned-executable-memory</key>
  <true/>
  <key>com.apple.security.network.client</key>
  <true/>
  <key>com.apple.security.network.server</key>
  <true/>
  <key>com.apple.security.files.user-selected.read-write</key>
  <true/>
</dict>
</plist>
```

---

## CI/CD Pipeline for Desktop Builds

### GitHub Actions Workflow

Create `.github/workflows/build-desktop.yml`:

```yaml
name: Build Desktop App
on:
  push:
    tags: ['v*']  # Trigger on version tags

jobs:
  build-backend:
    strategy:
      matrix:
        include:
          - os: ubuntu-latest
            goos: linux
            goarch: amd64
            binary: businessos-server
            output: linux-x64
          - os: ubuntu-latest
            goos: darwin
            goarch: arm64
            binary: businessos-server
            output: darwin-arm64
          - os: ubuntu-latest
            goos: darwin
            goarch: amd64
            binary: businessos-server
            output: darwin-x64
          - os: ubuntu-latest
            goos: windows
            goarch: amd64
            binary: businessos-server.exe
            output: win32-x64
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - name: Build backend
        env:
          GOOS: ${{ matrix.goos }}
          GOARCH: ${{ matrix.goarch }}
          CGO_ENABLED: '0'
        run: |
          cd desktop/backend-go
          go build -ldflags="-s -w" -o ../../desktop/resources/bin/${{ matrix.output }}/${{ matrix.binary }} ./cmd/server
      - uses: actions/upload-artifact@v4
        with:
          name: backend-${{ matrix.output }}
          path: desktop/resources/bin/${{ matrix.output }}/

  build-electron:
    needs: build-backend
    strategy:
      matrix:
        include:
          - os: macos-latest
            platform: darwin
          - os: windows-latest
            platform: win32
          - os: ubuntu-latest
            platform: linux
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
      - uses: actions/download-artifact@v4
        with:
          pattern: backend-*
          path: desktop/resources/bin/
          merge-multiple: true
      - name: Install dependencies
        run: cd desktop && npm ci
      - name: Make
        env:
          APPLE_ID: ${{ secrets.APPLE_ID }}
          APPLE_PASSWORD: ${{ secrets.APPLE_PASSWORD }}
          APPLE_TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
          APPLE_IDENTITY: ${{ secrets.APPLE_IDENTITY }}
        run: cd desktop && npm run make -- --platform=${{ matrix.platform }}
      - uses: actions/upload-artifact@v4
        with:
          name: desktop-${{ matrix.platform }}
          path: desktop/out/make/
```

### Required GitHub Secrets for Desktop Builds

| Secret | Platform | Description |
|--------|----------|-------------|
| `APPLE_ID` | macOS | Apple Developer email |
| `APPLE_PASSWORD` | macOS | App-specific password (NOT Apple ID password) |
| `APPLE_TEAM_ID` | macOS | Apple Developer Team ID |
| `APPLE_IDENTITY` | macOS | "Developer ID Application: Name (TeamID)" |
| `WIN_CSC_LINK` | Windows | Base64-encoded .pfx certificate |
| `WIN_CSC_KEY_PASSWORD` | Windows | Certificate password |
| `GH_TOKEN` | All | GitHub token for publishing releases |

---

## Troubleshooting

### "BusinessOS can't be opened because it is from an unidentified developer"

**Cause:** App not code-signed.
**Fix:** Either sign the app (set APPLE_ID env vars) or tell users to: Right-click → Open → Open Anyway.

### Backend binary not found

**Cause:** Go binary wasn't built or placed in wrong directory.
**Fix:** Build the Go binary and place it in `desktop/resources/bin/<platform>-<arch>/businessos-server`.

### "Unable to connect to backend" on startup

**Cause:** Go binary crashed or port conflict on 18080.
**Fix:** Check if something else is using port 18080: `lsof -i :18080`. Kill it or change the port.

### Auto-update not working

**Cause:** GitHub publisher not configured or releases not published.
**Fix:** Update `forge.config.ts` publisher with correct `owner` and `name`. Publish release to GitHub.

### DMG background not showing

**Cause:** Missing `desktop/resources/dmg-background.png`.
**Fix:** Create a 600x400 PNG and place it at the expected path.

---

## Current Status & Action Items

| Item | Status | Action Required |
|------|--------|----------------|
| Electron Forge config | Configured | Defaults to `Miosa-osa/businessos-5` |
| macOS maker (DMG) | Configured | Create dmg-background.png if missing |
| Windows maker (Squirrel) | Configured | Get code signing certificate |
| Linux makers (DEB/RPM) | Configured | Ready to build |
| Backend sidecar | Code complete | Build Go binaries before packaging |
| Auto-updater | Code complete | Publish release artifacts plus `latest-mac.yml` |
| Code signing (macOS) | Config ready | Get Apple Developer account, set env vars |
| Code signing (Windows) | Config ready | Get certificate from CA |
| CI/CD for desktop | NOT CREATED | Create `.github/workflows/build-desktop.yml` |
| Icons | NEEDS VERIFICATION | Check `desktop/resources/icons/` has all formats |
| Entitlements plist | NEEDS VERIFICATION | Check `desktop/resources/entitlements.mac.plist` exists |
| `node_modules` | NOT INSTALLED | Run `cd desktop && npm install` |

---

**Related Docs:**
- `docs/deployment/DEPLOYMENT.md` — Web deployment (Cloud Run + Vercel)
- `docs/deployment/DEPLOYMENT_GUIDE.md` — Backend deployment (Cloud Run)
- `docs/deployment/CLOUD-INFRASTRUCTURE.md` — GCP infrastructure setup
